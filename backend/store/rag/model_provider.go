package rag

import (
	"context"

	"github.com/chaitin/panda-wiki/domain"
)

// ModelProvider supplies configured models for pgvector embedding/rerank/rewrite.
type ModelProvider interface {
	GetModelByType(ctx context.Context, modelType domain.ModelType) (*domain.Model, error)
	GetChatModel(ctx context.Context) (*domain.Model, error)
}
