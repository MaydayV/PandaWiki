package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/mq"
	"github.com/chaitin/panda-wiki/mq/types"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/rag"
	"github.com/chaitin/panda-wiki/usecase"
)

type RAGMQHandler struct {
	config       *config.Config
	consumer     mq.MQConsumer
	logger       *log.Logger
	rag          rag.RAGService
	nodeRepo     *pg.NodeRepository
	kbRepo       *pg.KnowledgeBaseRepository
	llmUsecase   *usecase.LLMUsecase
	modelUsecase *usecase.ModelUsecase
}

func NewRAGMQHandler(config *config.Config, consumer mq.MQConsumer, logger *log.Logger, rag rag.RAGService, nodeRepo *pg.NodeRepository, kbRepo *pg.KnowledgeBaseRepository, llmUsecase *usecase.LLMUsecase, modelUsecase *usecase.ModelUsecase) (*RAGMQHandler, error) {
	h := &RAGMQHandler{
		config:       config,
		consumer:     consumer,
		logger:       logger.WithModule("mq.rag"),
		rag:          rag,
		nodeRepo:     nodeRepo,
		kbRepo:       kbRepo,
		llmUsecase:   llmUsecase,
		modelUsecase: modelUsecase,
	}
	if !config.RunWorker {
		return h, nil
	}
	if config.RAG.Provider == "pg" {
		return h, nil
	}
	if err := consumer.RegisterHandler(domain.VectorTaskTopic, h.HandleNodeContentVectorRequest); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *RAGMQHandler) HandleNodeContentVectorRequest(ctx context.Context, msg types.Message) error {
	var request domain.NodeReleaseVectorRequest
	err := json.Unmarshal(msg.GetData(), &request)
	if err != nil {
		h.logger.Error("unmarshal node content vector request failed", log.Error(err))
		return nil
	}
	if err := h.ProcessVectorRequest(ctx, &request); err != nil {
		h.logger.Error("process vector request failed", log.Error(err))
	}
	return nil
}

func (h *RAGMQHandler) ProcessVectorRequest(ctx context.Context, request *domain.NodeReleaseVectorRequest) error {
	switch request.Action {
	case "update_group_ids":
		h.logger.Info("update node group request", log.Any("request", request), log.Any("group_id", request.GroupIds))
		kb, err := h.kbRepo.GetKnowledgeBaseByID(ctx, request.KBID)
		if err != nil {
			return fmt.Errorf("get kb failed: %w", err)
		}
		if err := h.rag.UpdateDocumentGroupIDs(ctx, kb.DatasetID, request.DocID, request.GroupIds); err != nil {
			return fmt.Errorf("update node group failed: %w", err)
		}
		h.logger.Info("update node group success", log.Any("doc_id", request.DocID), log.Any("group_ids", request.GroupIds))

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
		if h.config.RAG.Provider == "pg" {
			if err := h.markNodeRagSucceeded(ctx, nodeRelease.NodeID); err != nil {
				h.logger.Error("update node rag status failed", log.Error(err))
			}
		}

		h.logger.Info("upsert node content vector success", log.Any("updated_ids", request.NodeReleaseID))
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

		model, err := h.modelUsecase.GetChatModel(ctx)
		if err != nil {
			return fmt.Errorf("get chat model failed: %w", err)
		}

		summary, err := h.llmUsecase.SummaryNode(ctx, request.KBID, model, node.Name, node.Content)
		if err != nil {
			return fmt.Errorf("summary node content failed: %w", err)
		}
		if err := h.nodeRepo.UpdateNodeSummary(ctx, request.KBID, request.NodeID, summary); err != nil {
			return fmt.Errorf("update node summary failed: %w", err)
		}
		if node.Status == domain.NodeStatusPublished {
			if err := h.nodeRepo.UpdateNodeStatus(ctx, request.KBID, request.NodeID, domain.NodeStatusDraft); err != nil {
				return fmt.Errorf("update node status failed: %w", err)
			}
		}

		h.logger.Info("summary node content vector success", log.Any("summary_id", request.NodeReleaseID), log.Any("summary", summary))
	default:
		h.logger.Warn("unknown vector request action", log.String("action", request.Action))
	}

	return nil
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
