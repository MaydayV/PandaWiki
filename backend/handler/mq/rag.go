package mq

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/rag"
	"github.com/chaitin/panda-wiki/usecase"
)

type RAGMQHandler struct {
	config       *config.Config
	logger       *log.Logger
	rag          rag.RAGService
	nodeRepo     *pg.NodeRepository
	kbRepo       *pg.KnowledgeBaseRepository
	llmUsecase   *usecase.LLMUsecase
	modelUsecase *usecase.ModelUsecase
}

func NewRAGMQHandler(config *config.Config, logger *log.Logger, rag rag.RAGService, nodeRepo *pg.NodeRepository, kbRepo *pg.KnowledgeBaseRepository, llmUsecase *usecase.LLMUsecase, modelUsecase *usecase.ModelUsecase) (*RAGMQHandler, error) {
	return &RAGMQHandler{
		config:       config,
		logger:       logger.WithModule("mq.rag"),
		rag:          rag,
		nodeRepo:     nodeRepo,
		kbRepo:       kbRepo,
		llmUsecase:   llmUsecase,
		modelUsecase: modelUsecase,
	}, nil
}

func (h *RAGMQHandler) ProcessVectorRequest(ctx context.Context, request *domain.NodeReleaseVectorRequest) error {
	switch request.Action {
	case "update_group_ids":
		h.logger.Info("update node group request", log.Any("request", request), log.String("node_id", request.NodeID))
		if err := h.syncDocumentGroupIDs(ctx, *request); err != nil {
			return fmt.Errorf("update node group failed: %w", err)
		}

	case "upsert":
		h.logger.Debug("upsert node content vector request", "request", request)
		nodeRelease, err := h.nodeRepo.GetNodeReleaseWithDirPathByID(ctx, request.NodeReleaseID)
		if err != nil {
			return fmt.Errorf("get node release failed: %w", err)
		}
		if nodeRelease.Type == domain.NodeTypeFolder {
			h.logger.Info("node is folder, skip upsert", log.Any("node_release_id", request.NodeReleaseID))
			return nil
		}
		isLatest, err := h.nodeRepo.IsLatestNodeRelease(ctx, nodeRelease.NodeID, nodeRelease.ID)
		if err != nil {
			return fmt.Errorf("check latest node release failed: %w", err)
		}
		if !isLatest {
			h.logger.Info("node release is superseded, skip upsert",
				log.String("node_id", nodeRelease.NodeID),
				log.String("node_release_id", nodeRelease.ID))
			return nil
		}
		if request.RequestID != "" && nodeRelease.Meta.LastVectorRequestID == request.RequestID {
			h.logger.Debug("vector request already processed, skip",
				log.String("node_release_id", request.NodeReleaseID),
				log.String("request_id", request.RequestID))
			return nil
		}
		// 幂等键由发布快照本身派生，而不是每批随机生成的 request_id。
		// 这样调用方重复发起同一逻辑操作时也能被拦截。
		dedupeKey := nodeReleaseVectorKey(nodeRelease.ID, nodeRelease.Content)
		if !request.Force && nodeRelease.Meta.LastVectorContentKey == dedupeKey {
			h.logger.Debug("upsert already processed, skip",
				log.Any("node_release_id", request.NodeReleaseID),
				log.String("dedupe_key", dedupeKey))
			return nil
		}
		kb, err := h.kbRepo.GetKnowledgeBaseByID(ctx, request.KBID)
		if err != nil {
			return fmt.Errorf("get kb failed: %w", err)
		}

		groupIds, err := h.nodeRepo.GetNodeAuthGroupIdsByNodeId(ctx, nodeRelease.NodeID, consts.NodePermNameAnswerable)
		if err != nil {
			return fmt.Errorf("get group ids failed: %w", err)
		}

		if h.config.RAG.Provider == "pg" {
			if err := h.markNodeRagRunning(ctx, nodeRelease.NodeID); err != nil {
				h.logger.Error("update node rag running failed", log.Error(err))
			}
		}

		docID, err := h.rag.UpsertRecords(ctx, &rag.UpsertRecordsRequest{
			ID:        nodeRelease.ID,
			Title:     nodeRelease.Name,
			DatasetID: kb.DatasetID,
			DocID:     nodeRelease.DocID,
			Content:   nodeRelease.Content,
			GroupIDs:  groupIds,
		})
		if err != nil {
			if h.config.RAG.Provider == "pg" {
				_ = h.markNodeRagFailed(ctx, nodeRelease.NodeID, err.Error())
			}
			return fmt.Errorf("upsert node content vector failed: %w", err)
		}
		if err := h.nodeRepo.UpdateNodeReleaseDocID(ctx, request.NodeReleaseID, docID); err != nil {
			if h.config.RAG.Provider == "pg" {
				_ = h.markNodeRagFailed(ctx, nodeRelease.NodeID, err.Error())
			}
			return fmt.Errorf("update node release doc_id failed: %w", err)
		}
		oldDocIDs, err := h.nodeRepo.GetOldNodeDocIDsByNodeID(ctx, nodeRelease.ID, nodeRelease.NodeID)
		if err != nil {
			if h.config.RAG.Provider == "pg" {
				_ = h.markNodeRagFailed(ctx, nodeRelease.NodeID, err.Error())
			}
			return fmt.Errorf("get old doc ids failed: %w", err)
		}
		if len(oldDocIDs) > 0 {
			if err := h.rag.DeleteRecords(ctx, kb.DatasetID, oldDocIDs); err != nil {
				if h.config.RAG.Provider == "pg" {
					_ = h.markNodeRagFailed(ctx, nodeRelease.NodeID, err.Error())
				}
				return fmt.Errorf("delete old rag records failed: %w", err)
			}
		}
		if err := h.nodeRepo.UpdateNodeReleaseVectorState(ctx, request.NodeReleaseID, dedupeKey, request.RequestID); err != nil {
			if h.config.RAG.Provider == "pg" {
				_ = h.markNodeRagFailed(ctx, nodeRelease.NodeID, err.Error())
			}
			return fmt.Errorf("update node release vector state failed: %w", err)
		}
		if h.config.RAG.Provider == "pg" {
			if err := h.markNodeRagSucceeded(ctx, nodeRelease.NodeID); err != nil {
				h.logger.Error("update node rag status failed", log.Error(err))
			}
		}

		h.logger.Info("upsert node content vector success",
			log.Any("updated_ids", request.NodeReleaseID),
			log.String("request_id", request.RequestID),
			log.String("dedupe_key", dedupeKey))
	case "delete":
		h.logger.Info("delete node content vector request", log.Any("request", request))
		kb, err := h.kbRepo.GetKnowledgeBaseByID(ctx, request.KBID)
		if err != nil {
			return fmt.Errorf("get kb failed: %w", err)
		}
		if err := h.rag.DeleteRecords(ctx, kb.DatasetID, []string{request.DocID}); err != nil {
			return fmt.Errorf("delete node content vector failed: %w", err)
		}
		h.logger.Info("delete node content vector success", log.Any("deleted_id", request.NodeReleaseID), log.Any("deleted_doc_id", request.DocID))
	case "summary":
		h.logger.Info("summary node content vector request", log.Any("request", request))
		node, err := h.nodeRepo.GetNodeByID(ctx, request.NodeID)
		if err != nil {
			return fmt.Errorf("get node failed: %w", err)
		}
		if node.Type == domain.NodeTypeFolder {
			h.logger.Info("node is folder, skip summary", log.Any("node_id", request.NodeID))
			return nil
		}
		if request.RequestID != "" && node.Meta.LastSummaryRequestID == request.RequestID {
			h.logger.Debug("summary request already processed, skip",
				log.String("node_id", request.NodeID),
				log.String("request_id", request.RequestID))
			return nil
		}
		// 以节点当前内容作为幂等键：内容不变时重复消息直接跳过，内容变化后允许重新生成。
		dedupeKey := summaryContentKey(node)
		if !request.Force && node.Meta.LastSummaryContentKey == dedupeKey {
			h.logger.Debug("summary already processed, skip",
				log.Any("node_id", request.NodeID),
				log.String("dedupe_key", dedupeKey))
			return nil
		}

		model, err := h.modelUsecase.GetChatModel(ctx)
		if err != nil {
			return fmt.Errorf("get chat model failed: %w", err)
		}

		summary, err := h.llmUsecase.SummaryNode(ctx, request.KBID, model, node.Name, node.Content)
		if err != nil {
			return fmt.Errorf("summary node content failed: %w", err)
		}
		// 摘要内容、幂等标记和发布态回退在同一条 UPDATE 中完成，
		// 避免"标记已写入但状态未回退"时重试被幂等检查跳过。
		if err := h.nodeRepo.UpdateNodeSummary(ctx, request.KBID, request.NodeID, summary, dedupeKey, request.RequestID); err != nil {
			return fmt.Errorf("update node summary failed: %w", err)
		}

		h.logger.Info("summary node content vector success", log.Any("node_id", request.NodeID), log.Any("summary", summary))
	default:
		h.logger.Warn("unknown vector request action", log.String("action", request.Action))
	}

	return nil
}

// syncDocumentGroupIDs 在消费时读取当前权限，避免延迟重投的旧消息覆盖后续权限变更。
func (h *RAGMQHandler) syncDocumentGroupIDs(ctx context.Context, request domain.NodeReleaseVectorRequest) error {
	docID := request.DocID
	groupIDs := request.GroupIds
	nodeID := request.NodeID

	// 兼容升级前没有 NodeID 的消息：通过 doc_id 反查节点，再读取最新权限。
	if nodeID == "" && docID != "" {
		releases, err := h.nodeRepo.GetNodeReleasesByDocIDs(ctx, []string{docID})
		if err != nil {
			return fmt.Errorf("get node release by doc id failed: %w", err)
		}
		if release, ok := releases[docID]; ok {
			nodeID = release.NodeID
		}
	}

	if nodeID != "" {
		releases, err := h.nodeRepo.GetLatestNodeReleaseByNodeIDs(ctx, request.KBID, []string{nodeID})
		if err != nil {
			return fmt.Errorf("get latest node release failed: %w", err)
		}
		if len(releases) == 0 {
			h.logger.Info("node has no release, skip group update", log.String("node_id", nodeID))
			return nil
		}
		// 最新快照可能尚未完成索引。此时仍要更新请求中旧文档的权限，
		// 避免新快照索引失败后旧文档继续保留过期权限。
		docID = groupUpdateDocID(releases[0].DocID, request.DocID)
		if releases[0].DocID == "" {
			h.logger.Info("latest node release is not indexed, fallback to requested document",
				log.String("node_id", nodeID),
				log.String("doc_id", docID))
		}
		groupIDs, err = h.nodeRepo.GetNodeAuthGroupIdsByNodeId(ctx, nodeID, consts.NodePermNameAnswerable)
		if err != nil {
			return fmt.Errorf("get current node group ids failed: %w", err)
		}
	}

	if docID == "" {
		h.logger.Info("group update has no document id, skip", log.Any("request", request))
		return nil
	}
	kb, err := h.kbRepo.GetKnowledgeBaseByID(ctx, request.KBID)
	if err != nil {
		return fmt.Errorf("get kb failed: %w", err)
	}
	if err := h.rag.UpdateDocumentGroupIDs(ctx, kb.DatasetID, docID, groupIDs); err != nil {
		return fmt.Errorf("update document group ids failed: %w", err)
	}
	h.logger.Info("update node group success", log.String("doc_id", docID), log.Any("group_ids", groupIDs))
	return nil
}

func groupUpdateDocID(latestDocID, requestedDocID string) string {
	if latestDocID != "" {
		return latestDocID
	}
	return requestedDocID
}

// nodeReleaseVectorKey 返回发布快照的确定性幂等键。
// 快照内容不可变，因此同一 release 无论被重复投递多少次都只会成功索引一次。
func nodeReleaseVectorKey(nodeReleaseID, content string) string {
	sum := sha256.Sum256([]byte(nodeReleaseID + ":" + content))
	return fmt.Sprintf("upsert:%x", sum)
}

// summaryContentKey 以节点当前内容作为幂等键，内容变化后允许重新生成摘要。
func summaryContentKey(node *domain.Node) string {
	sum := sha256.Sum256([]byte(node.ID + ":" + node.Content))
	return fmt.Sprintf("summary:%x", sum)
}

func (h *RAGMQHandler) markNodeRagRunning(ctx context.Context, nodeID string) error {
	return h.nodeRepo.Update(ctx, nodeID, map[string]interface{}{
		"rag_info": domain.RagInfo{
			Status:  consts.NodeRagStatusRunning,
			Message: "indexing in pgvector",
		},
	})
}

func (h *RAGMQHandler) markNodeRagSucceeded(ctx context.Context, nodeID string) error {
	return h.nodeRepo.Update(ctx, nodeID, map[string]interface{}{
		"rag_info": domain.RagInfo{
			Status:   consts.NodeRagStatusSucceeded,
			Message:  "indexed in pgvector",
			SyncedAt: time.Now(),
		},
	})
}

func (h *RAGMQHandler) markNodeRagFailed(ctx context.Context, nodeID string, message string) error {
	return h.nodeRepo.Update(ctx, nodeID, map[string]interface{}{
		"rag_info": domain.RagInfo{
			Status:   consts.NodeRagStatusFailed,
			Message:  message,
			SyncedAt: time.Now(),
		},
	})
}
