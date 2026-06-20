package cache

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/redis/go-redis/v9"
)

type KBRepo struct {
	cache *cache.Cache
}

func NewKBRepo(cache *cache.Cache) *KBRepo {
	return &KBRepo{cache: cache}
}

func (r *KBRepo) GetKB(ctx context.Context, kbID string) (*domain.KnowledgeBase, error) {
	kbStr, err := r.cache.Get(ctx, kbID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	if kbStr == "" {
		return nil, nil
	}

	var kb domain.KnowledgeBase
	err = json.Unmarshal([]byte(kbStr), &kb)
	if err != nil {
		return nil, err
	}
	return &kb, nil
}

func (r *KBRepo) SetKB(ctx context.Context, kbID string, kb *domain.KnowledgeBase) error {
	kbStr, err := json.Marshal(kb)
	if err != nil {
		return err
	}
	return r.cache.Set(ctx, kbID, kbStr, 10*time.Minute).Err()
}

func (r *KBRepo) DeleteKB(ctx context.Context, kbID string) error {
	return r.cache.Del(ctx, kbID).Err()
}

func (r *KBRepo) ClearSession(ctx context.Context) error {
	return r.cache.DeleteKeysWithPrefix(ctx, "session_")
}
