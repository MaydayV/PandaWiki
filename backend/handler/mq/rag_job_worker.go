package mq

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/log"
	pgRepo "github.com/chaitin/panda-wiki/repo/pg"
)

const (
	ragJobPollInterval = time.Second
	ragJobBatchSize    = 8
)

type RAGJobWorker struct {
	config  *config.Config
	logger  *log.Logger
	repo    *pgRepo.RagJobRepository
	handler *RAGMQHandler
}

func NewRAGJobWorker(
	config *config.Config,
	logger *log.Logger,
	repo *pgRepo.RagJobRepository,
	handler *RAGMQHandler,
) *RAGJobWorker {
	return &RAGJobWorker{
		config:  config,
		logger:  logger.WithModule("mq.rag_job_worker"),
		repo:    repo,
		handler: handler,
	}
}

func (w *RAGJobWorker) Start(ctx context.Context) {
	if !w.config.RunWorker || w.config.RAG.Provider != "pg" {
		return
	}
	workerID := uuid.NewString()
	w.logger.Info("start rag job worker", log.String("worker_id", workerID))

	ticker := time.NewTicker(ragJobPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("rag job worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx, workerID)
		}
	}
}

func (w *RAGJobWorker) processBatch(ctx context.Context, workerID string) {
	for i := 0; i < ragJobBatchSize; i++ {
		if ctx.Err() != nil {
			return
		}
		if !w.processOne(ctx, workerID) {
			return
		}
	}
}

func (w *RAGJobWorker) processOne(ctx context.Context, workerID string) bool {
	job, req, err := w.repo.Claim(ctx, workerID)
	if errors.Is(err, pgRepo.ErrNoRagJob) {
		return false
	}
	if err != nil {
		w.logger.Error("claim rag job failed", log.Error(err))
		return false
	}

	if err := w.handler.ProcessVectorRequest(ctx, req); err != nil {
		w.logger.Error("process rag job failed",
			log.String("job_id", job.ID),
			log.Error(err))
		if markErr := w.repo.MarkFailed(ctx, job.ID, job.Attempts, job.MaxAttempts, err); markErr != nil {
			w.logger.Error("mark rag job failed", log.String("job_id", job.ID), log.Error(markErr))
		}
		return true
	}

	if err := w.repo.MarkDone(ctx, job.ID); err != nil {
		w.logger.Error("mark rag job done failed", log.String("job_id", job.ID), log.Error(err))
	}
	return true
}
