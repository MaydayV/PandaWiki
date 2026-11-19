package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/store/pg"
)

type AuthRepo struct {
	db     *pg.DB
	logger *log.Logger
	cache  *cache.Cache
}

func NewAuthRepo(db *pg.DB, logger *log.Logger, cache *cache.Cache) *AuthRepo {
	return &AuthRepo{
		db:     db,
		logger: logger,
		cache:  cache,
	}
}

func (r *AuthRepo) GetAuthUserinfoByIDs(ctx context.Context, authIDs []uint) (map[uint]*domain.AuthInfo, error) {
	if len(authIDs) == 0 {
		return nil, nil
	}

	var authUserInfo = []domain.AuthInfo{}
	err := r.db.WithContext(ctx).Table("auths").
		Select("id,user_info as auth_user_info").
		Where("id IN (?) ", authIDs).
		Where("source_type NOT IN (?)", consts.BotSourceTypes).
		Find(&authUserInfo).Error
	if err != nil {
		return nil, err
	}
	//set map
	result := make(map[uint]*domain.AuthInfo, 0)
	for _, a := range authUserInfo {
		result[a.ID] = &a
	}
	return result, nil
}

func (r *AuthRepo) GetAuthGroupByAuthId(ctx context.Context, authID uint) ([]domain.AuthGroup, error) {
	authGroups := make([]domain.AuthGroup, 0)
	err := r.db.WithContext(ctx).Model(&domain.AuthGroup{}).
		Where("? = ANY(auth_ids)", authID).
		Find(&authGroups).Error
	if err != nil {
		return nil, err
	}

	return authGroups, nil
}

// getAllAuthGroupsAsMap fetches all auth groups and returns them as a map for quick lookup
func (r *AuthRepo) getAllAuthGroupsAsMap(ctx context.Context) (map[uint]*domain.AuthGroup, error) {
	var allGroups []domain.AuthGroup
	err := r.db.WithContext(ctx).Find(&allGroups).Error
	if err != nil {
		return nil, err
	}

	groupMap := lo.SliceToMap(allGroups, func(group domain.AuthGroup) (uint, *domain.AuthGroup) {
		return group.ID, &group
	})

	return groupMap, nil
}

// getAuthGroupsWithParentsByAuthId is a helper method that retrieves user's auth groups and all parent groups
func (r *AuthRepo) getAuthGroupsWithParentsByAuthId(ctx context.Context, authID uint) (map[uint]domain.AuthGroup, error) {
	// Get user's direct auth groups
	var directGroups []domain.AuthGroup
	err := r.db.WithContext(ctx).Model(&domain.AuthGroup{}).
		Where("? = ANY(auth_ids)", authID).
		Find(&directGroups).Error
	if err != nil {
		return nil, err
	}

	if len(directGroups) == 0 {
		return make(map[uint]domain.AuthGroup), nil
	}

	groupMap, err := r.getAllAuthGroupsAsMap(ctx)
	if err != nil {
		return nil, err
	}

	resultGroups := make(map[uint]domain.AuthGroup)
	visited := make(map[uint]bool)

	var findParents func(uint)
	findParents = func(groupID uint) {
		if visited[groupID] {
			return // Avoid circular reference
		}
		visited[groupID] = true

		group, exists := groupMap[groupID]
		if !exists {
			return // Group not found, end search
		}

		resultGroups[group.ID] = *group

		if group.ParentID != nil {
			findParents(*group.ParentID)
		}
	}

	// Process user's direct groups and their parent groups
	for _, group := range directGroups {
		resultGroups[group.ID] = group
		if group.ParentID != nil {
			findParents(*group.ParentID)
		}
	}

	return resultGroups, nil
}

// GetAuthGroupWithParentsByAuthId retrieves user's auth groups and all parent groups as slice
func (r *AuthRepo) GetAuthGroupWithParentsByAuthId(ctx context.Context, authID uint) ([]domain.AuthGroup, error) {
	groupsMap, err := r.getAuthGroupsWithParentsByAuthId(ctx, authID)
	if err != nil {
		return nil, err
	}

	result := make([]domain.AuthGroup, 0, len(groupsMap))
	for _, group := range groupsMap {
		result = append(result, group)
	}

	return result, nil
}

func (r *AuthRepo) GetAuthGroupIdsByAuthId(ctx context.Context, authID uint) ([]int, error) {
	groupIds := make([]int, 0)
	err := r.db.WithContext(ctx).Model(&domain.AuthGroup{}).
		Where("? = ANY(auth_ids)", authID).
		Pluck("id", &groupIds).Error
	if err != nil {
		return nil, err
	}

	return groupIds, nil
}

// GetAuthGroupIdsWithParentsByAuthId retrieves user's auth group IDs and all parent group IDs (for permission inheritance)
func (r *AuthRepo) GetAuthGroupIdsWithParentsByAuthId(ctx context.Context, authID uint) ([]int, error) {
	groupsMap, err := r.getAuthGroupsWithParentsByAuthId(ctx, authID)
	if err != nil {
		return nil, err
	}

	result := make([]int, 0, len(groupsMap))
	for _, group := range groupsMap {
		result = append(result, int(group.ID))
	}

	return result, nil
}

func (r *AuthRepo) GetAuthBySourceType(ctx context.Context, sourceType consts.SourceType) (*domain.Auth, error) {
	var auth *domain.Auth
	if err := r.db.WithContext(ctx).Model(&domain.Auth{}).Where("source_type = ?", string(sourceType)).First(&auth).Error; err != nil {
		return nil, err
	}
	return auth, nil
}

func (r *AuthRepo) GetAuthByKBIDAndSourceType(ctx context.Context, kbID string, sourceType consts.SourceType) (*domain.Auth, error) {
	var auth *domain.Auth
	if err := r.db.WithContext(ctx).Model(&domain.Auth{}).Where("kb_id = ? AND source_type = ?", kbID, string(sourceType)).First(&auth).Error; err != nil {
		return nil, err
	}
	return auth, nil
}

func (r *AuthRepo) CreateAuth(ctx context.Context, auth *domain.Auth) error {
	return r.db.WithContext(ctx).Model(&domain.Auth{}).Create(auth).Error
}

func (r *AuthRepo) DeleteAuth(ctx context.Context, kbID string, authId int64) error {
	return r.db.WithContext(ctx).Where("kb_id = ? and id = ?", kbID, authId).Delete(&domain.Auth{}).Error
}

func (r *AuthRepo) CreateAuthConfig(ctx context.Context, authConfig *domain.AuthConfig) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.AuthConfig
		err := tx.Model(&domain.AuthConfig{}).
			Where("kb_id = ?", authConfig.KbID).
			Where("source_type = ?", authConfig.SourceType).
			First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Model(&domain.AuthConfig{}).
					Create(authConfig).Error; err != nil {
					return err
				}
				return nil
			}
			return err
		}

		// 已存在则更新
		if err := tx.Model(&domain.AuthConfig{}).
			Where("kb_id = ?", authConfig.KbID).
			Where("source_type = ?", authConfig.SourceType).
			Updates(authConfig).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *AuthRepo) GetAuthById(ctx context.Context, kbID string, id uint) (*domain.Auth, error) {
	var auth domain.Auth
	if err := r.db.WithContext(ctx).
		Model(&domain.Auth{}).
		Where("kb_id = ?", kbID).
		Where("id = ?", id).
		First(&auth).Error; err != nil {
		return nil, err
	}
	return &auth, nil
}

func (r *AuthRepo) GetAuthConfig(ctx context.Context, kbID string, sourceType consts.SourceType) (*domain.AuthConfig, error) {
	var authConfig domain.AuthConfig

	if err := r.db.WithContext(ctx).
		Model(&domain.AuthConfig{}).
		Where("kb_id = ?", kbID).
		Where("source_type = ?", string(sourceType)).
		Order("created_at DESC").
		Limit(1).
		First(&authConfig).Error; err != nil {
		return nil, err
	}
	return &authConfig, nil
}

func (r *AuthRepo) GetAuths(ctx context.Context, kbID string, sourceType consts.SourceType) ([]domain.Auth, error) {
	auths := make([]domain.Auth, 0)

	if err := r.db.WithContext(ctx).
		Model(&domain.Auth{}).
		Where("kb_id = ?", kbID).
		Where("source_type in (?)", append(consts.BotSourceTypes, sourceType)).
		Order("last_login_time DESC").
		Find(&auths).Error; err != nil {
		return nil, err
	}
	return auths, nil
}

func (r *AuthRepo) GetOrCreateAuth(ctx context.Context, auth *domain.Auth, sourceType consts.SourceType) (*domain.Auth, error) {

	licenseEdition, _ := ctx.Value(consts.ContextKeyEdition).(consts.LicenseEdition)

	if licenseEdition < consts.LicenseEditionEnterprise {
		rdsKey := fmt.Sprintf("GetOrCreateAuth:%s", auth.KBID)
		if !r.cache.AcquireLock(ctx, rdsKey) {
			return nil, errors.New("rate limit exceeded, please try again later")
		}
		defer r.cache.ReleaseLock(ctx, rdsKey)
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing domain.Auth
		err := tx.Model(&domain.Auth{}).
			Where("kb_id = ?", auth.KBID).
			Where("source_type = ?", auth.SourceType).
			Where("union_id = ?", auth.UnionID).
			First(&existing).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				var count int64
				// 统计时排除机器人类型的认证，机器人不占用license限制名额
				if err := tx.Model(&domain.Auth{}).
					Where("kb_id = ?", auth.KBID).
					Where("source_type NOT IN (?)", consts.BotSourceTypes).
					Count(&count).Error; err != nil {
					return err
				}

				if int(count) >= domain.GetBaseEditionLimitation(ctx).MaxSSOUser {
					return fmt.Errorf("exceed max auth limit for kb %s, current count: %d, max limit: %d", auth.KBID, count, domain.GetBaseEditionLimitation(ctx).MaxSSOUser)
				}

				auth.LastLoginTime = time.Now()
				if err := tx.Model(&domain.Auth{}).Create(auth).Error; err != nil {
					return err
				}
				return nil
			}
			return err
		}

		updateMap := map[string]interface{}{
			"last_login_time": time.Now(),
			"user_info":       auth.UserInfo,
		}
		if err := tx.Model(&domain.Auth{}).Where("id = ?", existing.ID).Updates(updateMap).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	err := r.db.Model(&domain.Auth{}).
		Where("kb_id = ?", auth.KBID).
		Where("source_type = ?", auth.SourceType).
		Where("union_id = ?", auth.UnionID).
		First(&auth).Error
	if err != nil {
		return nil, err
	}

	return auth, nil
}

// CountByKBIDAndID 统计指定 auth 记录数（session 回查用）
func (r *AuthRepo) CountByKBIDAndID(ctx context.Context, kbID string, id uint, count *int64) error {
	return r.db.WithContext(ctx).Model(&domain.Auth{}).
		Where("kb_id = ? AND id = ?", kbID, id).
		Count(count).Error
}
