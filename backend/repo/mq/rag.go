package mq

import (
	"context"

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
	return r.ragJobRepo.Enqueue(ctx, request)
}
