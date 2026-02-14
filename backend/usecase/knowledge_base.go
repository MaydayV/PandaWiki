package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	v1 "github.com/chaitin/panda-wiki/api/kb/v1"
	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/cache"
	"github.com/chaitin/panda-wiki/repo/mq"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/rag"
)

type KnowledgeBaseUsecase struct {
	repo      *pg.KnowledgeBaseRepository
	nodeRepo  *pg.NodeRepository
	ragRepo   *mq.RAGRepository
	userRepo  *pg.UserRepository
	tokenRepo *pg.APITokenRepo
	rag       rag.RAGService
	kbCache   *cache.KBRepo
	logger    *log.Logger
	config    *config.Config
}

func NewKnowledgeBaseUsecase(repo *pg.KnowledgeBaseRepository, nodeRepo *pg.NodeRepository, ragRepo *mq.RAGRepository, userRepo *pg.UserRepository, tokenRepo *pg.APITokenRepo, rag rag.RAGService, kbCache *cache.KBRepo, logger *log.Logger, config *config.Config) (*KnowledgeBaseUsecase, error) {
	u := &KnowledgeBaseUsecase{
		repo:      repo,
		nodeRepo:  nodeRepo,
		ragRepo:   ragRepo,
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		rag:       rag,
		logger:    logger.WithModule("usecase.knowledge_base"),
		config:    config,
		kbCache:   kbCache,
	}
	return u, nil
}

func (u *KnowledgeBaseUsecase) CreateKnowledgeBase(ctx context.Context, req *domain.CreateKnowledgeBaseReq) (string, error) {
	// create kb in vector store
	datasetID, err := u.rag.CreateKnowledgeBase(ctx)
	if err != nil {
		return "", err
	}
	kbID := uuid.New().String()
	kb := &domain.KnowledgeBase{
		ID:        kbID,
		Name:      req.Name,
		DatasetID: datasetID,
		AccessSettings: domain.AccessSettings{
			Ports:      req.Ports,
			SSLPorts:   req.SSLPorts,
			PublicKey:  req.PublicKey,
			PrivateKey: req.PrivateKey,
			Hosts:      req.Hosts,
		},
	}

	if err := u.repo.CreateKnowledgeBase(ctx, req.MaxKB, kb); err != nil {
		return "", err
	}
	return kbID, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBaseList(ctx context.Context) ([]*domain.KnowledgeBaseListItem, error) {
	knowledgeBases, err := u.repo.GetKnowledgeBaseList(ctx)
	if err != nil {
		return nil, err
	}
	return knowledgeBases, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBaseListByUserId(ctx context.Context) ([]*domain.KnowledgeBaseListItem, error) {
	knowledgeBases, err := u.repo.GetKnowledgeBaseListByUserId(ctx)
	if err != nil {
		return nil, err
	}
	return knowledgeBases, nil
}

func (u *KnowledgeBaseUsecase) UpdateKnowledgeBase(ctx context.Context, req *domain.UpdateKnowledgeBaseReq) error {
	isChange, err := u.repo.UpdateKnowledgeBase(ctx, req)
	if err != nil {
		return err
	}

	if isChange {
		if err := u.kbCache.ClearSession(ctx); err != nil {
			return err
		}
	}

	if err := u.kbCache.DeleteKB(ctx, req.ID); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBase(ctx context.Context, kbID string) (*domain.KnowledgeBase, error) {
	kb, err := u.kbCache.GetKB(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb != nil {
		return kb, nil
	}
	kb, err = u.repo.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if err := u.kbCache.SetKB(ctx, kbID, kb); err != nil {
		return nil, err
	}
	return kb, nil
}

func (u *KnowledgeBaseUsecase) GetKnowledgeBasePerm(ctx context.Context, kbID string) (consts.UserKBPermission, error) {

	perm, err := u.repo.GetKBPermByUserId(ctx, kbID)
	if err != nil {
		return "", err
	}

	return perm, nil
}

func (u *KnowledgeBaseUsecase) DeleteKnowledgeBase(ctx context.Context, kbID string) error {
	if err := u.repo.DeleteKnowledgeBase(ctx, kbID); err != nil {
		return err
	}
	// delete vector store
	if err := u.rag.DeleteKnowledgeBase(ctx, kbID); err != nil {
		return err
	}
	if err := u.kbCache.DeleteKB(ctx, kbID); err != nil {
		return err
	}
	return nil
}

func (u *KnowledgeBaseUsecase) CreateKBRelease(ctx context.Context, req *domain.CreateKBReleaseReq, userId string) (string, error) {
	if len(req.NodeIDs) > 0 {
		// create published nodes
		releaseIDs, err := u.nodeRepo.CreateNodeReleases(ctx, req.KBID, userId, req.NodeIDs)
		if err != nil {
			return "", fmt.Errorf("failed to create published nodes: %w", err)
		}
		if len(releaseIDs) > 0 {
			// async upsert vector content via mq
			nodeContentVectorRequests := make([]*domain.NodeReleaseVectorRequest, 0)
			for _, releaseID := range releaseIDs {
				nodeContentVectorRequests = append(nodeContentVectorRequests, &domain.NodeReleaseVectorRequest{
					KBID:          req.KBID,
					NodeReleaseID: releaseID,
					Action:        "upsert",
				})
			}
			if err := u.ragRepo.AsyncUpdateNodeReleaseVector(ctx, nodeContentVectorRequests); err != nil {
				return "", err
			}
		}
	}

	release := &domain.KBRelease{
		ID:          uuid.New().String(),
		KBID:        req.KBID,
		Message:     req.Message,
		Tag:         req.Tag,
		PublisherId: userId,
		CreatedAt:   time.Now(),
	}
	if err := u.repo.CreateKBRelease(ctx, release); err != nil {
		return "", fmt.Errorf("failed to create kb release: %w", err)
	}

	return release.ID, nil
}

func (u *KnowledgeBaseUsecase) GetKBReleaseList(ctx context.Context, req *domain.GetKBReleaseListReq) (*domain.GetKBReleaseListResp, error) {
	total, releases, err := u.repo.GetKBReleaseList(ctx, req.KBID, req.Offset(), req.Limit())
	if err != nil {
		return nil, err
	}

	return domain.NewPaginatedResult(releases, uint64(total)), nil
}

func (u *KnowledgeBaseUsecase) GetKBUserList(ctx context.Context, req v1.KBUserListReq) ([]v1.KBUserListItemResp, error) {
	users, err := u.repo.GetKBUserlist(ctx, req.KBId)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u *KnowledgeBaseUsecase) KBUserInvite(ctx context.Context, req v1.KBUserInviteReq) error {
	user, err := u.userRepo.GetUser(ctx, req.UserId)
	if err != nil {
		return err
	}
	if user.Role == consts.UserRoleAdmin {
		return fmt.Errorf("knowledge base can not invite to admin user")
	}

	if err := u.repo.CreateKBUser(ctx, &domain.KBUsers{
		KBId:      req.KBId,
		UserId:    req.UserId,
		Perm:      req.Perm,
		CreatedAt: time.Now(),
	}); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) UpdateUserKB(ctx context.Context, req v1.KBUserUpdateReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}

	kbUser, err := u.repo.GetKBUser(ctx, req.KBId, req.UserId)
	if err != nil {
		return err
	}
	if authInfo.IsToken {
		if authInfo.KBId != req.KBId {
			return fmt.Errorf("invalid knowledge base token")
		}
		if authInfo.Permission != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can update user from knowledge base")
		}
	} else {
		user, err := u.userRepo.GetUser(ctx, authInfo.UserId)
		if err != nil {
			return err
		}
		if user.Role != consts.UserRoleAdmin && kbUser.Perm != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can update user from knowledge base")
		}
	}
	return u.repo.UpdateKBUserPerm(ctx, req.KBId, req.UserId, req.Perm)
}

func (u *KnowledgeBaseUsecase) KBUserDelete(ctx context.Context, req v1.KBUserDeleteReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}

	kbUser, err := u.repo.GetKBUser(ctx, req.KBId, req.UserId)
	if err != nil {
		return err
	}
	if authInfo.IsToken {
		if authInfo.KBId != req.KBId {
			return fmt.Errorf("knowledge base can not delete user from knowledge base")
		}
		if authInfo.Permission != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can delete user from knowledge base")
		}
	} else {
		user, err := u.userRepo.GetUser(ctx, authInfo.UserId)
		if err != nil {
			return err
		}
		if user.Role != consts.UserRoleAdmin && kbUser.Perm != consts.UserKBPermissionFullControl {
			return fmt.Errorf("only admin can delete user from knowledge base")
		}
	}
	if err := u.repo.DeleteKBUser(ctx, req.KBId, req.UserId); err != nil {
		return err
	}

	return nil
}

func (u *KnowledgeBaseUsecase) CreateAPIToken(ctx context.Context, req *domain.CreateAPITokenReq) (*domain.APITokenListItem, error) {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return nil, fmt.Errorf("authInfo not found in context")
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("api token name is required")
	}

	token, err := domain.GenerateAPITokenValue()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	apiToken := &domain.APIToken{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		UserID:             authInfo.UserId,
		Token:              token,
		KbId:               req.KBID,
		Permission:         req.Permission,
		RateLimitPerMinute: req.RateLimitPerMinute,
		DailyQuota:         req.DailyQuota,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := u.tokenRepo.Create(ctx, apiToken); err != nil {
		return nil, err
	}

	return &domain.APITokenListItem{
		ID:                 apiToken.ID,
		Name:               apiToken.Name,
		Token:              apiToken.Token,
		Permission:         apiToken.Permission,
		RateLimitPerMinute: apiToken.RateLimitPerMinute,
		DailyQuota:         apiToken.DailyQuota,
		CreatedAt:          apiToken.CreatedAt,
		UpdatedAt:          apiToken.UpdatedAt,
	}, nil
}

func (u *KnowledgeBaseUsecase) ListAPIToken(ctx context.Context, req domain.APITokenListReq) ([]*domain.APITokenListItem, error) {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return nil, fmt.Errorf("authInfo not found in context")
	}
	apiTokens, err := u.tokenRepo.ListByKBAndUser(ctx, req.KBID, authInfo.UserId)
	if err != nil {
		return nil, err
	}

	items := make([]*domain.APITokenListItem, 0, len(apiTokens))
	for _, item := range apiTokens {
		items = append(items, &domain.APITokenListItem{
			ID:                 item.ID,
			Name:               item.Name,
			Token:              item.Token,
			Permission:         item.Permission,
			RateLimitPerMinute: item.RateLimitPerMinute,
			DailyQuota:         item.DailyQuota,
			CreatedAt:          item.CreatedAt,
			UpdatedAt:          item.UpdatedAt,
		})
	}
	return items, nil
}

func (u *KnowledgeBaseUsecase) UpdateAPIToken(ctx context.Context, req *domain.UpdateAPITokenReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}
	if !req.HasUpdates() {
		return fmt.Errorf("nothing to update")
	}

	if req.Name != nil {
		trimmedName := strings.TrimSpace(*req.Name)
		if trimmedName == "" {
			return fmt.Errorf("api token name is required")
		}
		req.Name = &trimmedName
	}

	if err := u.tokenRepo.Update(ctx, req, authInfo.UserId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("api token not found")
		}
		return err
	}
	return nil
}

func (u *KnowledgeBaseUsecase) DeleteAPIToken(ctx context.Context, req domain.DeleteAPITokenReq) error {
	authInfo := domain.GetAuthInfoFromCtx(ctx)
	if authInfo == nil {
		return fmt.Errorf("authInfo not found in context")
	}
	if err := u.tokenRepo.Delete(ctx, req.ID, req.KBID, authInfo.UserId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("api token not found")
		}
		return err
	}
	return nil
}
