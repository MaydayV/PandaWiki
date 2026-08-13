package mq

import (
	"context"
	"encoding/json"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/mq"
	pgRepo "github.com/chaitin/panda-wiki/repo/pg"
)

type RAGRepository struct {
	producer   mq.MQProducer
	config     *config.Config
	ragJobRepo *pgRepo.RagJobRepository
}

func NewRAGRepository(producer mq.MQProducer, config *config.Config, ragJobRepo *pgRepo.RagJobRepository) *RAGRepository {
	return &RAGRepository{
		producer:   producer,
		config:     config,
		ragJobRepo: ragJobRepo,
	}
}

func (r *RAGRepository) AsyncUpdateNodeReleaseVector(ctx context.Context, request []*domain.NodeReleaseVectorRequest) error {
	if r.config.RAG.Provider == "pg" {
		return r.ragJobRepo.Enqueue(ctx, request)
	}
	for _, req := range request {
		requestBytes, err := json.Marshal(req)
		if err != nil {
			return err
		}
		if err := r.producer.Produce(ctx, domain.VectorTaskTopic, "", requestBytes); err != nil {
			return err
		}
	}
	return nil
}
