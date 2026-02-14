package pg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/store/pg"
)

type APITokenRepo struct {
	db     *pg.DB
	logger *log.Logger
	cache  *cache.Cache
}

func NewAPITokenRepo(db *pg.DB, logger *log.Logger, cache *cache.Cache) *APITokenRepo {
	return &APITokenRepo{
		db:     db,
		logger: logger,
		cache:  cache,
	}
}

func (r *APITokenRepo) GetByTokenWithCache(ctx context.Context, token string) (*domain.APIToken, error) {
	cacheKey := apiTokenCacheKey(token)

	cachedData, err := r.cache.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		var apiToken domain.APIToken
		if err := json.Unmarshal([]byte(cachedData), &apiToken); err == nil {
			return &apiToken, nil
		}
	}

	// 缓存未命中，从数据库查询
	var apiToken domain.APIToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&apiToken).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get api token by token failed: %w", err)
	}

	if tokenData, err := json.Marshal(&apiToken); err == nil {
		if err := r.cache.Set(ctx, cacheKey, tokenData, 30*time.Minute).Err(); err != nil {
			r.logger.Warn("failed to cache API token", log.Error(err))
		}
	}

	return &apiToken, nil
}

func (r *APITokenRepo) Create(ctx context.Context, apiToken *domain.APIToken) error {
	if err := r.db.WithContext(ctx).Create(apiToken).Error; err != nil {
		return fmt.Errorf("create api token failed: %w", err)
	}
	return nil
}

func (r *APITokenRepo) ListByKBAndUser(ctx context.Context, kbID, userID string) ([]*domain.APIToken, error) {
	var apiTokens []*domain.APIToken
	if err := r.db.WithContext(ctx).
		Where("kb_id = ? AND user_id = ?", kbID, userID).
		Order("created_at DESC").
		Find(&apiTokens).Error; err != nil {
		return nil, fmt.Errorf("list api tokens failed: %w", err)
	}
	return apiTokens, nil
}

func (r *APITokenRepo) GetByID(ctx context.Context, id, kbID, userID string) (*domain.APIToken, error) {
	var apiToken domain.APIToken
	if err := r.db.WithContext(ctx).
		Where("id = ? AND kb_id = ? AND user_id = ?", id, kbID, userID).
		First(&apiToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get api token by id failed: %w", err)
	}
	return &apiToken, nil
}

func (r *APITokenRepo) Update(ctx context.Context, req *domain.UpdateAPITokenReq, userID string) error {
	currentToken, err := r.GetByID(ctx, req.ID, req.KBID, userID)
	if err != nil {
		return err
	}
	if currentToken == nil {
		return gorm.ErrRecordNotFound
	}

	updateMap := map[string]interface{}{}
	if req.Name != nil {
		updateMap["name"] = *req.Name
	}
	if req.Permission != nil {
		updateMap["permission"] = *req.Permission
	}
	if req.RateLimitPerMinute != nil {
		updateMap["rate_limit_per_minute"] = *req.RateLimitPerMinute
	}
	if req.DailyQuota != nil {
		updateMap["daily_quota"] = *req.DailyQuota
	}
	if len(updateMap) == 0 {
		return nil
	}

	if err := r.db.WithContext(ctx).
		Model(&domain.APIToken{}).
		Where("id = ? AND kb_id = ? AND user_id = ?", req.ID, req.KBID, userID).
		Updates(updateMap).Error; err != nil {
		return fmt.Errorf("update api token failed: %w", err)
	}

	if err := r.invalidateTokenCache(ctx, currentToken.Token); err != nil {
		return err
	}
	return nil
}

func (r *APITokenRepo) Delete(ctx context.Context, id, kbID, userID string) error {
	currentToken, err := r.GetByID(ctx, id, kbID, userID)
	if err != nil {
		return err
	}
	if currentToken == nil {
		return gorm.ErrRecordNotFound
	}

	if err := r.db.WithContext(ctx).
		Where("id = ? AND kb_id = ? AND user_id = ?", id, kbID, userID).
		Delete(&domain.APIToken{}).Error; err != nil {
		return fmt.Errorf("delete api token failed: %w", err)
	}

	if err := r.invalidateTokenCache(ctx, currentToken.Token); err != nil {
		return err
	}
	return nil
}

func (r *APITokenRepo) invalidateTokenCache(ctx context.Context, token string) error {
	if token == "" || r.cache == nil {
		return nil
	}
	if err := r.cache.Del(ctx, apiTokenCacheKey(token)).Err(); err != nil {
		return fmt.Errorf("invalidate api token cache failed: %w", err)
	}
	return nil
}

func apiTokenCacheKey(token string) string {
	return fmt.Sprintf("api_token:%s", token)
}
