package mq

import (
	"context"

	"github.com/google/uuid"

	"github.com/chaitin/panda-wiki/domain"
	pgRepo "github.com/chaitin/panda-wiki/repo/pg"
)

type RAGRepository struct {
	ragJobRepo *pgRepo.RagJobRepository
}

func NewRAGRepository(ragJobRepo *pgRepo.RagJobRepository) *RAGRepository {
	return &RAGRepository{
		ragJobRepo: ragJobRepo,
	}
}

func (r *RAGRepository) AsyncUpdateNodeReleaseVector(ctx context.Context, request []*domain.NodeReleaseVectorRequest) error {
	requestID := uuid.NewString()
	for _, req := range request {
		if req.RequestID == "" {
			req.RequestID = requestID
		}
	}
	return r.ragJobRepo.Enqueue(ctx, request)
}
