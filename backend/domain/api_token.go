package domain

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/chaitin/panda-wiki/consts"
)

type APIToken struct {
	ID                 string                  `json:"id" gorm:"primaryKey"`
	Name               string                  `json:"name" gorm:"not null"`
	UserID             string                  `json:"user_id" gorm:"not null"`
	Token              string                  `json:"token" gorm:"uniqueIndex;not null"`
	KbId               string                  `json:"kb_id" gorm:"not null"`
	Permission         consts.UserKBPermission `json:"permission" gorm:"not null"`
	RateLimitPerMinute int                     `json:"rate_limit_per_minute" gorm:"not null;default:0"`
	DailyQuota         int                     `json:"daily_quota" gorm:"not null;default:0"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

func (APIToken) TableName() string {
	return "api_tokens"
}

type CreateAPITokenReq struct {
	KBID               string                  `json:"kb_id" validate:"required"`
	Name               string                  `json:"name" validate:"required"`
	Permission         consts.UserKBPermission `json:"permission" validate:"required,oneof=full_control doc_manage data_operate"`
	RateLimitPerMinute int                     `json:"rate_limit_per_minute" validate:"gte=0"`
	DailyQuota         int                     `json:"daily_quota" validate:"gte=0"`
}

type APITokenListReq struct {
	KBID string `json:"kb_id" query:"kb_id" validate:"required"`
}

type APITokenListItem struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Token              string                  `json:"token"`
	Permission         consts.UserKBPermission `json:"permission"`
	RateLimitPerMinute int                     `json:"rate_limit_per_minute"`
	DailyQuota         int                     `json:"daily_quota"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

type UpdateAPITokenReq struct {
	ID                 string                   `json:"id" validate:"required"`
	KBID               string                   `json:"kb_id" validate:"required"`
	Name               *string                  `json:"name,omitempty"`
	Permission         *consts.UserKBPermission `json:"permission,omitempty" validate:"omitempty,oneof=full_control doc_manage data_operate"`
	RateLimitPerMinute *int                     `json:"rate_limit_per_minute,omitempty" validate:"omitempty,gte=0"`
	DailyQuota         *int                     `json:"daily_quota,omitempty" validate:"omitempty,gte=0"`
}

func (r UpdateAPITokenReq) HasUpdates() bool {
	return r.Name != nil || r.Permission != nil || r.RateLimitPerMinute != nil || r.DailyQuota != nil
}

type DeleteAPITokenReq struct {
	ID   string `json:"id" query:"id" validate:"required"`
	KBID string `json:"kb_id" query:"kb_id" validate:"required"`
}

func GenerateAPITokenValue() (string, error) {
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate api token failed: %w", err)
	}
	return hex.EncodeToString(randomBytes), nil
}

type CtxAuthInfo struct {
	IsToken    bool
	Permission consts.UserKBPermission
	UserId     string
	KBId       string
}

type contextKey string

const (
	CtxAuthInfoKey contextKey = "ctx_auth_info"
)

func GetAuthInfoFromCtx(c context.Context) *CtxAuthInfo {
	v := c.Value(CtxAuthInfoKey)
	if v == nil {
		return nil
	}
	ctxAuthInfo, ok := v.(*CtxAuthInfo)
	if !ok {
		return nil
	}
	return ctxAuthInfo
}
