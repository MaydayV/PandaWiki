package rag

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	modelkitDomain "github.com/chaitin/ModelKit/v2/domain"
	modelkit "github.com/chaitin/ModelKit/v2/usecase"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	storepg "github.com/chaitin/panda-wiki/store/pg"
	"github.com/chaitin/panda-wiki/utils"
)

const (
	pgVectorTopK      = 30
	pgVectorRetrieveK = 10
	pgRewritePrompt   = "根据对话历史，将用户的最新问题改写为适合知识库检索的独立查询。只输出改写后的查询，不要解释。"
)

type ragChunkRow struct {
	ID        string          `gorm:"column:id;primaryKey"`
	DatasetID string          `gorm:"column:dataset_id"`
	DocID     string          `gorm:"column:doc_id"`
	Content   string          `gorm:"column:content"`
	Embedding pgvector.Vector `gorm:"column:embedding"`
	GroupIDs  pq.Int64Array   `gorm:"column:group_ids;type:int[]"`
	Tags      pq.StringArray  `gorm:"column:tags;type:text[]"`
	Seq       int             `gorm:"column:seq"`
}

func (ragChunkRow) TableName() string { return "rag_chunks" }

type chunkCandidate struct {
	ID         string
	Content    string
	DocID      string
	Similarity float64
}

type PGRAG struct {
	db              *gorm.DB
	logger          *log.Logger
	mdConv          *converter.Converter
	models          ModelProvider
	modelkit        *modelkit.ModelKit
	config          *config.Config
	chunkTokenLimit int
	embeddingDim    int
}

func NewPGRAG(config *config.Config, logger *log.Logger, db *storepg.DB, models ModelProvider) (*PGRAG, error) {
	return &PGRAG{
		db:              db.DB,
		logger:          logger.WithModule("store.rag.pg"),
		mdConv:          NewHTML2MDConverter(),
		models:          models,
		modelkit:        modelkit.NewModelKit(logger.Logger),
		config:          config,
		chunkTokenLimit: config.RAG.PG.ChunkTokenLimit,
		embeddingDim:    config.RAG.PG.EmbeddingDim,
	}, nil
}

func (s *PGRAG) CreateKnowledgeBase(ctx context.Context) (string, error) {
	return uuid.NewString(), nil
}

func (s *PGRAG) DeleteKnowledgeBase(ctx context.Context, datasetID string) error {
	return s.db.WithContext(ctx).Where("dataset_id = ?", datasetID).Delete(&ragChunkRow{}).Error
}

func (s *PGRAG) DeleteRecords(ctx context.Context, datasetID string, docIDs []string) error {
	if len(docIDs) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).
		Where("dataset_id = ? AND doc_id IN ?", datasetID, docIDs).
		Delete(&ragChunkRow{}).Error
}

func (s *PGRAG) UpdateDocumentGroupIDs(ctx context.Context, datasetID string, docID string, groupIds []int) error {
	groupArray := make(pq.Int64Array, len(groupIds))
	for i, id := range groupIds {
		groupArray[i] = int64(id)
	}
	return s.db.WithContext(ctx).
		Model(&ragChunkRow{}).
		Where("dataset_id = ? AND doc_id = ?", datasetID, docID).
		Update("group_ids", groupArray).Error
}

func (s *PGRAG) UpsertRecords(ctx context.Context, req *UpsertRecordsRequest) (string, error) {
	markdown := req.Content
	if utils.IsLikelyHTML(req.Content) {
		var err error
		markdown, err = s.mdConv.ConvertString(req.Content)
		if err != nil {
			return "", fmt.Errorf("convert html to markdown failed: %w", err)
		}
	}

	chunks, err := SplitMarkdownChunks(markdown, s.chunkTokenLimit)
	if err != nil {
		return "", fmt.Errorf("split markdown chunks failed: %w", err)
	}
	if len(chunks) == 0 {
		chunks = []string{strings.TrimSpace(markdown)}
	}

	if err := s.DeleteRecords(ctx, req.DatasetID, []string{req.DocID}); err != nil {
		return "", err
	}

	embeddings, err := s.embedTexts(ctx, chunks)
	if err != nil {
		return "", err
	}

	groupArray := make(pq.Int64Array, len(req.GroupIDs))
	for i, id := range req.GroupIDs {
		groupArray[i] = int64(id)
	}
	tagArray := pq.StringArray(req.Tags)

	rows := make([]ragChunkRow, 0, len(chunks))
	for i, content := range chunks {
		vec, err := toPgVector(embeddings[i], s.embeddingDim)
		if err != nil {
			return "", err
		}
		rows = append(rows, ragChunkRow{
			ID:        uuid.NewString(),
			DatasetID: req.DatasetID,
			DocID:     req.DocID,
			Content:   content,
			Embedding: vec,
			GroupIDs:  groupArray,
			Tags:      tagArray,
			Seq:       i,
		})
	}

	if err := s.db.WithContext(ctx).Create(&rows).Error; err != nil {
		return "", fmt.Errorf("insert rag chunks failed: %w", err)
	}
	return req.DocID, nil
}

func (s *PGRAG) QueryRecords(ctx context.Context, req *QueryRecordsRequest) (string, []*domain.NodeContentChunk, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return query, nil, nil
	}

	rewritten := query
	if len(req.HistoryMsgs) > 0 {
		var err error
		rewritten, err = s.rewriteQuery(ctx, req.HistoryMsgs, query)
		if err != nil {
			s.logger.Warn("rewrite query failed, fallback to original", log.Error(err))
			rewritten = query
		}
	}

	queryEmbedding, err := s.embedTexts(ctx, []string{rewritten})
	if err != nil {
		return rewritten, nil, err
	}
	vec, err := toPgVector(queryEmbedding[0], s.embeddingDim)
	if err != nil {
		return rewritten, nil, err
	}

	candidates, err := s.searchVector(ctx, req.DatasetID, vec, req.GroupIDs, pgVectorTopK)
	if err != nil {
		return rewritten, nil, err
	}
	if len(candidates) == 0 {
		return rewritten, nil, nil
	}

	reranked, err := s.rerankChunks(ctx, rewritten, candidates)
	if err != nil {
		s.logger.Warn("rerank failed, fallback to vector order", log.Error(err))
		reranked = candidates
	}

	filtered := make([]chunkCandidate, 0, len(reranked))
	for _, item := range reranked {
		if req.SimilarityThreshold > 0 && item.Similarity < req.SimilarityThreshold {
			continue
		}
		filtered = append(filtered, item)
	}

	maxPerDoc := req.MaxChunksPerDoc
	if maxPerDoc <= 0 {
		maxPerDoc = 3
	}
	docCount := make(map[string]int)
	result := make([]*domain.NodeContentChunk, 0, pgVectorRetrieveK)
	for _, item := range filtered {
		if docCount[item.DocID] >= maxPerDoc {
			continue
		}
		result = append(result, &domain.NodeContentChunk{
			ID:      item.ID,
			Content: item.Content,
			DocID:   item.DocID,
		})
		docCount[item.DocID]++
		if len(result) >= pgVectorRetrieveK {
			break
		}
	}

	return rewritten, result, nil
}

func (s *PGRAG) ListDocuments(ctx context.Context, datasetID string, documentIDs []string) ([]Document, error) {
	if len(documentIDs) == 0 {
		return nil, nil
	}
	type docAgg struct {
		DocID string
		Count int64
	}
	var rows []docAgg
	if err := s.db.WithContext(ctx).
		Model(&ragChunkRow{}).
		Select("doc_id, COUNT(*) as count").
		Where("dataset_id = ? AND doc_id IN ?", datasetID, documentIDs).
		Group("doc_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	countMap := lo.SliceToMap(rows, func(item docAgg) (string, int64) {
		return item.DocID, item.Count
	})

	docs := make([]Document, 0, len(documentIDs))
	for _, docID := range documentIDs {
		status := "processing"
		msg := "waiting for embedding"
		if countMap[docID] > 0 {
			status = "completed"
			msg = "indexed in pgvector"
		}
		docs = append(docs, Document{
			ID:          docID,
			DatasetID:   datasetID,
			Status:      status,
			ProgressMsg: msg,
			MetaData:    DocumentMetadata{},
		})
	}
	return docs, nil
}

func (s *PGRAG) GetModelList(ctx context.Context) ([]*domain.Model, error) {
	return nil, nil
}

func (s *PGRAG) AddModel(ctx context.Context, model *domain.Model) (string, error) {
	return model.ID, nil
}

func (s *PGRAG) UpdateModel(ctx context.Context, model *domain.Model) error {
	return nil
}

func (s *PGRAG) UpsertModel(ctx context.Context, model *domain.Model) error {
	return nil
}

func (s *PGRAG) DeleteModel(ctx context.Context, model *domain.Model) error {
	return nil
}

func (s *PGRAG) embedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	model, err := s.models.GetModelByType(ctx, domain.ModelTypeEmbedding)
	if err != nil {
		return nil, fmt.Errorf("get embedding model failed: %w", err)
	}
	modelkitModel, err := model.ToModelkitModel()
	if err != nil {
		return nil, err
	}
	embedder, err := s.modelkit.GetEmbedder(ctx, modelkitModel)
	if err != nil {
		return nil, fmt.Errorf("get embedder failed: %w", err)
	}
	resp, err := s.modelkit.UseEmbedder(ctx, embedder, texts)
	if err != nil {
		return nil, fmt.Errorf("embed texts failed: %w", err)
	}
	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: got %d want %d", len(resp.Embeddings), len(texts))
	}

	out := make([][]float32, len(resp.Embeddings))
	for i, item := range resp.Embeddings {
		vec := make([]float32, len(item.Embedding))
		for j, v := range item.Embedding {
			vec[j] = float32(v)
		}
		out[i] = vec
	}
	return out, nil
}

func (s *PGRAG) rewriteQuery(ctx context.Context, history []*schema.Message, query string) (string, error) {
	model, err := s.models.GetChatModel(ctx)
	if err != nil {
		return query, err
	}
	modelkitModel, err := model.ToModelkitModel()
	if err != nil {
		return query, err
	}
	chatModel, err := s.modelkit.GetChatModel(ctx, modelkitModel)
	if err != nil {
		return query, err
	}

	msgs := []*schema.Message{schema.SystemMessage(pgRewritePrompt)}
	for _, item := range history {
		switch item.Role {
		case schema.User, schema.Assistant:
			msgs = append(msgs, item)
		}
	}
	msgs = append(msgs, schema.UserMessage(query))

	resp, err := chatModel.Generate(ctx, msgs)
	if err != nil {
		return query, err
	}
	rewritten := strings.TrimSpace(resp.Content)
	if rewritten == "" {
		return query, nil
	}
	return rewritten, nil
}

func (s *PGRAG) searchVector(ctx context.Context, datasetID string, queryVec pgvector.Vector, groupIDs []int, limit int) ([]chunkCandidate, error) {
	rows, err := s.db.WithContext(ctx).Raw(`
SELECT id, doc_id, content, 1 - (embedding <=> ?) AS similarity
FROM rag_chunks
WHERE dataset_id = ?
  AND (
    cardinality(group_ids) = 0
    OR cardinality(?::int[]) = 0
    OR group_ids && ?::int[]
  )
ORDER BY embedding <=> ?
LIMIT ?
`, queryVec, datasetID, pq.Array(groupIDs), pq.Array(groupIDs), queryVec, limit).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	candidates := make([]chunkCandidate, 0, limit)
	for rows.Next() {
		var item chunkCandidate
		if err := rows.Scan(&item.ID, &item.DocID, &item.Content, &item.Similarity); err != nil {
			return nil, err
		}
		candidates = append(candidates, item)
	}
	return candidates, rows.Err()
}

func (s *PGRAG) rerankChunks(ctx context.Context, query string, candidates []chunkCandidate) ([]chunkCandidate, error) {
	model, err := s.models.GetModelByType(ctx, domain.ModelTypeRerank)
	if err != nil {
		return candidates, err
	}
	modelkitModel, err := model.ToModelkitModel()
	if err != nil {
		return candidates, err
	}
	reranker, err := s.modelkit.GetReranker(ctx, modelkitModel)
	if err != nil {
		return candidates, err
	}

	documents := lo.Map(candidates, func(item chunkCandidate, _ int) string {
		return item.Content
	})
	resp, err := reranker.Rerank(ctx, modelkitDomain.RerankRequest{
		Query:           query,
		Documents:       documents,
		ReturnDocuments: false,
	})
	if err != nil {
		return candidates, err
	}

	reranked := make([]chunkCandidate, 0, len(resp.Results))
	for _, result := range resp.Results {
		if result.Index < 0 || result.Index >= len(candidates) {
			continue
		}
		item := candidates[result.Index]
		item.Similarity = result.RelevanceScore
		reranked = append(reranked, item)
	}
	if len(reranked) == 0 {
		return candidates, nil
	}
	return reranked, nil
}

func toPgVector(values []float32, dim int) (pgvector.Vector, error) {
	if len(values) == 0 {
		return pgvector.Vector{}, fmt.Errorf("empty embedding vector")
	}
	if dim > 0 && len(values) != dim {
		return pgvector.Vector{}, fmt.Errorf("embedding dimension mismatch: got %d want %d", len(values), dim)
	}
	float64s := make([]float32, len(values))
	copy(float64s, values)
	norm := float32(0)
	for _, v := range float64s {
		norm += v * v
	}
	if norm > 0 {
		norm = float32(1 / math.Sqrt(float64(norm)))
		for i := range float64s {
			float64s[i] *= norm
		}
	}
	return pgvector.NewVector(float64s), nil
}
