package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	v1 "github.com/chaitin/panda-wiki/api/auth/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/cache"
)

type AuthUsecase struct {
	AuthRepo *pg.AuthRepo
	logger   *log.Logger
	kbRepo   *pg.KnowledgeBaseRepository
	cache    *cache.Cache
}

func NewAuthUsecase(authRepo *pg.AuthRepo, logger *log.Logger, kbRepo *pg.KnowledgeBaseRepository, cache *cache.Cache) (*AuthUsecase, error) {
	u := &AuthUsecase{
		AuthRepo: authRepo,
		kbRepo:   kbRepo,
		logger:   logger.WithModule("usecase.auth"),
		cache:    cache,
	}
	return u, nil
}

type StateInfo struct {
	KbId        string `json:"kb_id"`
	RedirectUrl string `json:"redirect_url"`
	Verifier    string `json:"verifier"`
}

func (u *AuthUsecase) GetAuthBySourceType(ctx context.Context, sourceType consts.SourceType) (*domain.Auth, error) {
	return u.AuthRepo.GetAuthBySourceType(ctx, sourceType)
}

func (u *AuthUsecase) DeleteAuth(ctx context.Context, req v1.AuthDeleteReq) error {
	return u.AuthRepo.DeleteAuth(ctx, req.KbID, req.ID)
}

func (u *AuthUsecase) SetAuth(ctx context.Context, req v1.AuthSetReq) error {
	if err := u.AuthRepo.CreateAuthConfig(ctx, &domain.AuthConfig{
		AuthSetting: domain.AuthSetting{
			ClientID:     req.ClientID,
			ClientSecret: req.ClientSecret,
			Proxy:        req.Proxy,
		},
		KbID:       req.KBID,
		SourceType: req.SourceType,
	}); err != nil {
		return err
	}
	return nil
}

func (u *AuthUsecase) GetAuthInfo(ctx context.Context, kbId string, authId uint) (*domain.Auth, error) {

	auth, err := u.AuthRepo.GetAuthById(ctx, kbId, authId)
	if err != nil {
		return nil, err
	}

	return auth, nil
}

func (u *AuthUsecase) GetAuth(ctx context.Context, kbID string, sourceType consts.SourceType) (*v1.AuthGetResp, error) {
	authConfig, err := u.AuthRepo.GetAuthConfig(ctx, kbID, sourceType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	auths, err := u.AuthRepo.GetAuths(ctx, kbID, sourceType)
	if err != nil {
		return nil, err
	}
	as := make([]v1.AuthItem, 0, len(auths))

	for _, auth := range auths {
		as = append(as, v1.AuthItem{
			ID:            auth.ID,
			Username:      auth.UserInfo.Username,
			IP:            auth.IP,
			AvatarUrl:     auth.UserInfo.AvatarUrl,
			SourceType:    auth.SourceType,
			LastLoginTime: auth.LastLoginTime,
			CreatedAt:     auth.CreatedAt,
		})
	}

	resp := &v1.AuthGetResp{
		ClientID:     authConfig.AuthSetting.ClientID,
		ClientSecret: authConfig.AuthSetting.ClientSecret,
		SourceType:   authConfig.SourceType,
		Proxy:        authConfig.AuthSetting.Proxy,
		Auths:        as,
	}
	return resp, nil

}

func (u *AuthUsecase) ValidateRedirectUrl(ctx context.Context, kbId, redirectUrl string) (bool, error) {
	kb, err := u.kbRepo.GetKnowledgeBaseByID(ctx, kbId)
	if err != nil {
		return false, err
	}
	redirectURL, _ := url.Parse(redirectUrl)

	if kb.AccessSettings.BaseURL != "" {
		baseUrl, _ := url.Parse(kb.AccessSettings.BaseURL)
		if baseUrl.Hostname() != redirectURL.Hostname() {
			return false, nil
		}
	} else {
		if !slices.Contains(kb.AccessSettings.Hosts, redirectURL.Hostname()) {
			return false, nil
		}
	}

	return true, nil
}

func (u *AuthUsecase) genState(ctx context.Context, stateInfo StateInfo) (string, error) {
	state := uuid.New().String()

	stateInfoBytes, err := json.Marshal(stateInfo)
	if err != nil {
		return "", err
	}

	if err := u.cache.SetNX(ctx, state, stateInfoBytes, 15*time.Minute).Err(); err != nil {
		return "", err
	}

	return state, nil
}

func (u *AuthUsecase) SaveNewSession(c echo.Context, auth *domain.Auth) error {
	s := c.Get(domain.SessionCacheKey)
	if s == nil {
		return fmt.Errorf("failed to get session store")
	}
	store := s.(sessions.Store)

	newSess := sessions.NewSession(store, domain.SessionName)
	newSess.IsNew = true

	newSess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
	}

	newSess.Values["user_id"] = auth.ID
	newSess.Values["kb_id"] = auth.KBID

	if err := newSess.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	c.Logger().Info("session_saved:", newSess.Values)
	return nil
}

// HasAuthByID 校验 session 指向的 auth 绑定是否仍存在：
// 解绑第三方登录（OpenAuthDelete）后旧 session 应立即失效
func (u *AuthUsecase) HasAuthByID(ctx context.Context, kbID string, authID uint) bool {
	var count int64
	if err := u.AuthRepo.CountByKBIDAndID(ctx, kbID, authID, &count); err != nil {
		// 查询失败保守放行（不因 DB 抖动锁死全部企业认证用户）
		return true
	}
	return count > 0
}
