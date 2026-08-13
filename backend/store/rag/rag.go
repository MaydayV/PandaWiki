package rag

import (
	"context"

	"github.com/cloudwego/eino/schema"
	"github.com/google/wire"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	storepg "github.com/chaitin/panda-wiki/store/pg"
)

type QueryRecordsRequest struct {
	DatasetID           string
	Query               string
	GroupIDs            []int
	Tags                []string
	SimilarityThreshold float64
	HistoryMsgs         []*schema.Message
	MaxChunksPerDoc     int
}

type UpsertRecordsRequest struct {
	ID        string
	DatasetID string
	DocID     string
	Title     string
	Content   string
	GroupIDs  []int
	Tags      []string
}

type DocumentMetadata struct {
	GroupIDs []int `json:"group_ids"`
}

type Document struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	DatasetID   string           `json:"dataset_id"`
	Status      string           `json:"status"`
	ProgressMsg string           `json:"progress_msg"`
	MetaData    DocumentMetadata `json:"meta_data"`
	Tags        []string         `json:"tags"`
}

type RAGService interface {
	CreateKnowledgeBase(ctx context.Context) (string, error)
	UpsertRecords(ctx context.Context, req *UpsertRecordsRequest) (string, error)
	QueryRecords(ctx context.Context, req *QueryRecordsRequest) (string, []*domain.NodeContentChunk, error)
	DeleteRecords(ctx context.Context, datasetID string, docIDs []string) error
	DeleteKnowledgeBase(ctx context.Context, datasetID string) error
	UpdateDocumentGroupIDs(ctx context.Context, datasetID string, docID string, groupIds []int) error
	ListDocuments(ctx context.Context, datasetID string, documentIDs []string) ([]Document, error)

	GetModelList(ctx context.Context) ([]*domain.Model, error)
	AddModel(ctx context.Context, model *domain.Model) (string, error)
	UpdateModel(ctx context.Context, model *domain.Model) error
	UpsertModel(ctx context.Context, model *domain.Model) error
	DeleteModel(ctx context.Context, model *domain.Model) error
}

func NewRAGService(config *config.Config, logger *log.Logger, db *storepg.DB, models ModelProvider) (RAGService, error) {
	return NewPGRAG(config, logger, db, models)
}

var ProviderSet = wire.NewSet(NewRAGService)
