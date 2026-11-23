package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/githubsync"
	mq "github.com/chaitin/panda-wiki/repo/mq"
	pgrepo "github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/cache"
	"github.com/chaitin/panda-wiki/utils"
)

// GitHubSyncUsecase 实现 git 为唯一内容真源的同步模式：
// GitHub 私有仓库的 MD 文章定时/手动同步进知识库，分类按目录结构与
// front-matter 推导，图片自动转存 OSS，变更自动发布触发 RAG 更新。
type GitHubSyncUsecase struct {
	repo        *pgrepo.GitHubSyncRepo
	nodeRepo    *pgrepo.NodeRepository
	navRepo     *pgrepo.NavRepository
	ragRepo     *mq.RAGRepository
	fileUsecase *FileUsecase
	kbUsecase   *KnowledgeBaseUsecase
	cache       *cache.Cache
	logger      *log.Logger
	running     sync.Map // kbID -> struct{}，进程内防重入（单机架构）
}

func NewGitHubSyncUsecase(
	repo *pgrepo.GitHubSyncRepo,
	nodeRepo *pgrepo.NodeRepository,
	navRepo *pgrepo.NavRepository,
	ragRepo *mq.RAGRepository,
	fileUsecase *FileUsecase,
	kbUsecase *KnowledgeBaseUsecase,
	cache *cache.Cache,
	logger *log.Logger,
) *GitHubSyncUsecase {
	return &GitHubSyncUsecase{
		repo:        repo,
		nodeRepo:    nodeRepo,
		navRepo:     navRepo,
		ragRepo:     ragRepo,
		fileUsecase: fileUsecase,
		kbUsecase:   kbUsecase,
		cache:       cache,
		logger:      logger.WithModule("usecase.github_sync"),
	}
}

const (
	githubDefaultBranch    = "main"
	githubDefaultRootNav   = "未分类"
	githubMaxFileSize      = 10 << 20 // 单文件 10MB
	githubSyncWaitInterval = time.Minute
	githubSyncLockTTL      = 10 * time.Minute
	githubSyncRunTimeout   = 5 * time.Minute
	githubSyncMaxConc      = 4
)

// GetConfig 读取配置（token 脱敏）与最近同步状态
func (u *GitHubSyncUsecase) GetConfig(ctx context.Context, kbID string) (*domain.GitHubSyncSetting, error) {
	cfg, err := u.repo.GetSetting(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return &domain.GitHubSyncSetting{
			KBID:            kbID,
			Branch:          githubDefaultBranch,
			AutoPublish:     true,
			IntervalMinutes: 0,
		}, nil
	}
	hasToken := cfg.Token != ""
	cfg.Token = ""
	cfg.HasToken = hasToken
	return cfg, nil
}

// SaveConfig 保存配置（校验仓库及分支可达，token 加密落库）
func (u *GitHubSyncUsecase) SaveConfig(ctx context.Context, kbID string, req *domain.GitHubSyncSetting) error {
	req.Repo = strings.TrimSpace(req.Repo)
	if err := githubsync.ValidateRepoName(req.Repo); err != nil {
		return err
	}
	req.Branch = strings.TrimSpace(req.Branch)
	if req.Branch == "" {
		req.Branch = githubDefaultBranch
	}
	if req.IntervalMinutes < 0 || req.IntervalMinutes > 1440 {
		return fmt.Errorf("interval_minutes must be between 0 and 1440")
	}

	// 读取已有配置：保留历史同步状态与已有 token（前端不回传明文 token）
	existing, err := u.repo.GetSetting(ctx, kbID)
	if err != nil {
		return err
	}
	if existing != nil {
		req.LastSyncAt = existing.LastSyncAt
		req.LastSyncResult = existing.LastSyncResult
	}

	rawToken := strings.TrimSpace(req.Token)
	plainToken := rawToken
	if plainToken == "" && existing != nil {
		plainToken = utils.DecryptSecret(existing.Token)
	}

	repoChanged := existing == nil || existing.Repo != req.Repo || existing.Branch != req.Branch
	if req.Enabled || rawToken != "" || repoChanged {
		client := githubsync.NewClient(plainToken)
		if err := client.VerifyRepoAndBranch(ctx, req.Repo, req.Branch); err != nil {
			return fmt.Errorf("github repo verify failed: %w", err)
		}
	}

	req.HasToken = false
	// token 加密落库（未配置 ENCRYPTION_KEY 时透传明文，兼容既有行为）
	req.Token = utils.EncryptSecret(plainToken)

	return u.repo.SaveSetting(ctx, kbID, req)
}

// TriggerSync 手动触发一次同步（同步执行，完成后返回结果）
func (u *GitHubSyncUsecase) TriggerSync(ctx context.Context, kbID, userID string) (*domain.GitHubSyncResult, error) {
	cfg, err := u.repo.GetSetting(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.Repo == "" {
		return nil, fmt.Errorf("github sync is not configured for this kb")
	}
	return u.RunSync(ctx, cfg, fmt.Sprintf("manual:%s", userID))
}

// RunSync 执行一次完整同步
func (u *GitHubSyncUsecase) RunSync(ctx context.Context, cfg *domain.GitHubSyncSetting, trigger string) (*domain.GitHubSyncResult, error) {
	if cfg.Repo == "" {
		return nil, fmt.Errorf("repo is not configured")
	}
	if _, loaded := u.running.LoadOrStore(cfg.KBID, struct{}{}); loaded {
		return nil, fmt.Errorf("sync already running for this kb")
	}
	defer u.running.Delete(cfg.KBID)

	if u.cache != nil {
		lockKey := "github_sync_lock:" + cfg.KBID
		lockToken := uuid.NewString()
		acquired, lockErr := u.cache.AcquireLockWithTTL(ctx, lockKey, lockToken, githubSyncLockTTL)
		if lockErr != nil {
			u.logger.Warn("acquire github sync distributed lock failed", log.String("kb_id", cfg.KBID), log.Error(lockErr))
		} else if !acquired {
			return nil, fmt.Errorf("sync already running for this kb")
		} else {
			defer func() {
				releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer cancel()
				if err := u.cache.ReleaseLockWithToken(releaseCtx, lockKey, lockToken); err != nil {
					u.logger.Warn("release github sync distributed lock failed", log.String("kb_id", cfg.KBID), log.Error(err))
				}
			}()
		}
	}

	start := time.Now()
	result := &domain.GitHubSyncResult{}

	token := utils.DecryptSecret(cfg.Token)
	client := githubsync.NewClient(token)

	// 1. 拉取仓库文件树
	tree, err := client.GetTree(ctx, cfg.Repo, cfg.Branch)
	if err != nil {
		u.recordSyncFailure(ctx, cfg, err, start)
		return nil, err
	}

	// 当前仓库路径集合（文件 + 目录，用于孤儿检测）
	currentPaths := make(map[string]struct{}, len(tree))
	mdPaths := make([]string, 0, len(tree))
	treeByPath := make(map[string]githubsync.TreeEntry, len(tree))
	for _, entry := range tree {
		currentPaths[entry.Path] = struct{}{}
		treeByPath[entry.Path] = entry
		if entry.Type == "blob" && githubsync.IsMarkdown(entry.Path) && !githubsync.ShouldIgnore(entry.Path) {
			mdPaths = append(mdPaths, entry.Path)
		}
	}
	// 稳定处理顺序（文件名字母序）
	sort.Strings(mdPaths)

	// 2. 已同步节点映射（文档以仓库路径为键，目录以 folder:<navID>/<dir> 为键防跨栏目碰撞）
	synced, err := u.repo.GetSyncedNodes(ctx, cfg.KBID, cfg.Repo)
	if err != nil {
		return nil, err
	}
	syncedByPath := make(map[string]pgrepo.GitHubSyncedNode, len(synced))
	for _, n := range synced {
		key := n.Path
		if n.Type == int(domain.NodeTypeFolder) && !strings.HasPrefix(key, "folder:") {
			key = "folder:" + n.NavId + "/" + key
		}
		syncedByPath[key] = n
	}

	// 3. 现有 nav 列表（按名称匹配复用）
	existingNavs, err := u.navRepo.GetList(ctx, cfg.KBID)
	if err != nil {
		return nil, err
	}
	navByName := make(map[string]string, len(existingNavs))
	for _, n := range existingNavs {
		navByName[n.Name] = n.ID
	}

	// getOrCreateNav 按名称取/建栏目
	getOrCreateNav := func(name string) (string, error) {
		name = strings.TrimSpace(name)
		if name == "" {
			name = githubDefaultRootNav
		}
		if id, ok := navByName[name]; ok {
			return id, nil
		}
		nav := &domain.Nav{ID: uuid.NewString(), KbID: cfg.KBID, Name: name}
		if err := u.navRepo.Create(ctx, nav, nil); err != nil {
			return "", fmt.Errorf("create nav %q failed: %w", name, err)
		}
		navByName[name] = nav.ID
		return nav.ID, nil
	}

	changedNodeIDs := make([]string, 0)

	// getOrCreateFolders 按目录链取/建文件夹节点，返回最深一级的 parent 节点 id
	getOrCreateFolders := func(ctx context.Context, dir, navID string) (string, error) {
		parentID := ""
		cur := ""
		segs := strings.Split(strings.Trim(dir, "/"), "/")
		for _, seg := range segs {
			if seg == "" {
				continue
			}
			cur = path.Join(cur, seg)
			folderKey := "folder:" + navID + "/" + cur
			if n, ok := syncedByPath[folderKey]; ok {
				parentID = n.ID
				continue
			}
			nodeID, err := u.nodeRepo.Create(ctx, &domain.CreateNodeReq{
				KBID:     cfg.KBID,
				NavId:    navID,
				ParentID: parentID,
				Type:     domain.NodeTypeFolder,
				Name:     seg,
			}, "github_sync")
			if err != nil {
				return "", fmt.Errorf("create folder %q failed: %w", cur, err)
			}
			if err := u.nodeRepo.UpdateNodeGitHubSync(ctx, cfg.KBID, nodeID, &domain.NodeGitHubSync{
				Repo: cfg.Repo, Path: folderKey,
			}); err != nil {
				return "", err
			}
			syncedByPath[folderKey] = pgrepo.GitHubSyncedNode{ID: nodeID, NavId: navID, ParentID: parentID, Path: folderKey, Type: int(domain.NodeTypeFolder)}
			changedNodeIDs = append(changedNodeIDs, nodeID)
			result.Created++
			parentID = nodeID
		}
		return parentID, nil
	}

	// 4. 逐文件处理，仅收集本次实际创建或更新的节点用于自动发布
	for _, mdPath := range mdPaths {
		entry := treeByPath[mdPath]

		// sha 未变且节点已存在：跳过（省 LLM embed 成本）
		if n, ok := syncedByPath[mdPath]; ok && n.SHA == entry.SHA {
			result.Skipped++
			continue
		}

		nodeID, changed, err := u.syncOneFile(ctx, client, cfg, mdPath, entry.SHA, treeByPath, getOrCreateNav, getOrCreateFolders, syncedByPath, result)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", mdPath, err))
			u.logger.Error("sync md file failed", log.String("path", mdPath), log.Error(err))
			continue
		}
		if changed && nodeID != "" {
			changedNodeIDs = append(changedNodeIDs, nodeID)
		}
	}

	// 5. 孤儿删除（git 为唯一真源，且开关开启时）
	if cfg.DeleteOrphans {
		deletedDocIDs := make([]string, 0)
		for p, n := range syncedByPath {
			if strings.HasPrefix(p, "folder:") || n.Type != int(domain.NodeTypeDocument) {
				continue
			}
			if _, ok := currentPaths[p]; ok {
				continue
			}
			docIDs, err := u.nodeRepo.Delete(ctx, cfg.KBID, []string{n.ID})
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("delete %s: %v", p, err))
				continue
			}
			deletedDocIDs = append(deletedDocIDs, docIDs...)
			result.Deleted++
			delete(syncedByPath, p)
		}
		if len(deletedDocIDs) > 0 {
			delReqs := make([]*domain.NodeReleaseVectorRequest, 0, len(deletedDocIDs))
			for _, docID := range deletedDocIDs {
				delReqs = append(delReqs, &domain.NodeReleaseVectorRequest{
					KBID:   cfg.KBID,
					DocID:  docID,
					Action: "delete",
				})
			}
			if err := u.ragRepo.AsyncUpdateNodeReleaseVector(ctx, delReqs); err != nil {
				u.logger.Error("enqueue vector delete after sync failed", log.Error(err))
			}
		}
	}

	// 6. 自动发布：将本次新建/更新的变更节点走一次 KB release（或孤儿删除时刷新快照），触发 RAG 向量更新
	if cfg.AutoPublish && (len(changedNodeIDs) > 0 || result.Deleted > 0) {
		sort.Strings(changedNodeIDs)
		if _, err := u.kbUsecase.CreateKBRelease(ctx, &domain.CreateKBReleaseReq{
			KBID:    cfg.KBID,
			Message: fmt.Sprintf("GitHub sync (%s)", trigger),
			Tag:     time.Now().Format("github-20060102-150405"),
			NodeIDs: changedNodeIDs,
		}, "github_sync"); err != nil {
			u.logger.Error("auto publish after sync failed", log.Error(err))
			result.Errors = append(result.Errors, fmt.Sprintf("auto publish: %v", err))
		}
	}

	result.Duration = time.Since(start).Round(time.Millisecond).String()
	u.recordSyncSuccess(ctx, cfg, result)
	return result, nil
}

// syncOneFile 处理单个 MD 文件：拉取 → front-matter → 图片转存 → 分类 → 建/更节点
func (u *GitHubSyncUsecase) syncOneFile(
	ctx context.Context,
	client *githubsync.Client,
	cfg *domain.GitHubSyncSetting,
	mdPath, sha string,
	treeByPath map[string]githubsync.TreeEntry,
	getOrCreateNav func(string) (string, error),
	getOrCreateFolders func(context.Context, string, string) (string, error),
	syncedByPath map[string]pgrepo.GitHubSyncedNode,
	result *domain.GitHubSyncResult,
) (string, bool, error) {
	raw, err := client.GetRaw(ctx, cfg.Repo, cfg.Branch, mdPath, githubMaxFileSize)
	if err != nil {
		return "", false, err
	}
	doc, err := githubsync.ParseFrontMatter(raw)
	if err != nil {
		return "", false, err
	}
	if doc.FrontMatter.Order != nil && (*doc.FrontMatter.Order < 0 || *doc.FrontMatter.Order > domain.MaxPosition) {
		return "", false, fmt.Errorf("front-matter order %v out of range [0, %v]", *doc.FrontMatter.Order, domain.MaxPosition)
	}

	// 分类识别（优先级：front-matter category > 一级目录 > 默认栏目）
	dir := path.Dir(mdPath)
	navName := doc.FrontMatter.Category
	if navName == "" && dir != "." {
		navName = strings.SplitN(dir, "/", 2)[0]
	}
	navID, err := getOrCreateNav(navName)
	if err != nil {
		return "", false, err
	}

	existing, hasExisting := syncedByPath[mdPath]
	existingImages := make(map[string]string)
	if hasExisting && len(existing.ImagesJSON) > 0 {
		_ = json.Unmarshal(existing.ImagesJSON, &existingImages)
	}
	newImages := make(map[string]string)

	// 图片转存：相对路径图片按 blob sha 复用已有 OSS 对象，未缓存再拉取上传并改写链接
	content := doc.Body
	for _, ref := range githubsync.ExtractImageRefs(content) {
		repoImagePath := githubsync.ResolveRelativePath(dir, ref.Path)
		if !githubsync.IsImage(repoImagePath) {
			u.logger.Warn("skip non-image reference in markdown", log.String("image", repoImagePath))
			continue
		}
		imgEntry, hasImgEntry := treeByPath[repoImagePath]
		cacheKey := ""
		if hasImgEntry && imgEntry.SHA != "" {
			cacheKey = repoImagePath + ":" + imgEntry.SHA
		}
		if cacheKey != "" {
			if cachedURL, ok := existingImages[cacheKey]; ok && cachedURL != "" {
				newImages[cacheKey] = cachedURL
				content = strings.Replace(content, ref.FullText,
					fmt.Sprintf("![%s](%s)", ref.Alt, cachedURL), 1)
				continue
			}
		}

		img, err := client.GetRaw(ctx, cfg.Repo, cfg.Branch, repoImagePath, githubMaxFileSize)
		if err != nil {
			u.logger.Warn("sync image failed, keep original link", log.String("image", repoImagePath), log.Error(err))
			continue
		}
		key, err := u.fileUsecase.UploadFileFromBytes(ctx, cfg.KBID, path.Base(repoImagePath), img)
		if err != nil {
			u.logger.Warn("upload synced image failed", log.String("image", repoImagePath), log.Error(err))
			continue
		}
		staticURL := "/static-file/" + key
		if cacheKey != "" {
			newImages[cacheKey] = staticURL
		}
		content = strings.Replace(content, ref.FullText,
			fmt.Sprintf("![%s](%s)", ref.Alt, staticURL), 1)
	}

	title := doc.FrontMatter.Title
	if title == "" {
		title = githubsync.TitleFromPath(mdPath)
	}

	// folder 链（category 指定 nav 时完整目录归属文件夹链，否则一级目录之外的部分归属文件夹链）
	folderDir := ""
	if dir != "." {
		if doc.FrontMatter.Category != "" {
			folderDir = strings.Trim(dir, "/")
		} else if strings.Contains(strings.Trim(dir, "/"), "/") {
			folderDir = strings.SplitN(strings.Trim(dir, "/"), "/", 2)[1]
		}
	}
	parentID, err := getOrCreateFolders(ctx, folderDir, navID)
	if err != nil {
		return "", false, err
	}

	contentType := domain.ContentTypeMD
	syncInfo := &domain.NodeGitHubSync{
		Repo:   cfg.Repo,
		Path:   mdPath,
		SHA:    sha,
		Images: newImages,
	}

	if hasExisting && existing.Type == int(domain.NodeTypeDocument) {
		// 更新（包含栏目/父目录/排序/content_type）
		if err := u.nodeRepo.UpdateNodeContent(ctx, &domain.UpdateNodeReq{
			ID:          existing.ID,
			KBID:        cfg.KBID,
			NavId:       &navID,
			ParentID:    &parentID,
			Name:        &title,
			Content:     &content,
			Emoji:       &doc.FrontMatter.Emoji,
			ContentType: &contentType,
			Position:    doc.FrontMatter.Order,
		}, "github_sync"); err != nil {
			return "", false, fmt.Errorf("update node failed: %w", err)
		}
		if err := u.nodeRepo.UpdateNodeGitHubSync(ctx, cfg.KBID, existing.ID, syncInfo); err != nil {
			return "", false, err
		}
		syncedByPath[mdPath] = pgrepo.GitHubSyncedNode{
			ID:       existing.ID,
			NavId:    navID,
			ParentID: parentID,
			Path:     mdPath,
			SHA:      sha,
			Type:     int(domain.NodeTypeDocument),
		}
		result.Updated++
		return existing.ID, true, nil
	}

	// 创建
	nodeID, err := u.nodeRepo.Create(ctx, &domain.CreateNodeReq{
		KBID:        cfg.KBID,
		NavId:       navID,
		ParentID:    parentID,
		Type:        domain.NodeTypeDocument,
		Name:        title,
		Content:     content,
		Emoji:       doc.FrontMatter.Emoji,
		ContentType: &contentType,
		Position:    doc.FrontMatter.Order,
	}, "github_sync")
	if err != nil {
		return "", false, fmt.Errorf("create node failed: %w", err)
	}
	if err := u.nodeRepo.UpdateNodeGitHubSync(ctx, cfg.KBID, nodeID, syncInfo); err != nil {
		return "", false, err
	}
	syncedByPath[mdPath] = pgrepo.GitHubSyncedNode{
		ID:       nodeID,
		NavId:    navID,
		ParentID: parentID,
		Path:     mdPath,
		SHA:      sha,
		Type:     int(domain.NodeTypeDocument),
	}
	result.Created++
	return nodeID, true, nil
}

// recordSyncSuccess 回写成功状态
func (u *GitHubSyncUsecase) recordSyncSuccess(ctx context.Context, cfg *domain.GitHubSyncSetting, result *domain.GitHubSyncResult) {
	now := time.Now()
	summary := fmt.Sprintf("created=%d updated=%d skipped=%d deleted=%d failed=%d (%s)",
		result.Created, result.Updated, result.Skipped, result.Deleted, result.Failed, result.Duration)
	if len(result.Errors) > 0 {
		summary += " errors: " + strings.Join(result.Errors, "; ")
	}
	if err := u.repo.UpdateSyncStatus(ctx, cfg.KBID, now, summary); err != nil {
		u.logger.Error("save sync result failed", log.Error(err))
	}
}

// recordSyncFailure 回写失败状态
func (u *GitHubSyncUsecase) recordSyncFailure(ctx context.Context, cfg *domain.GitHubSyncSetting, syncErr error, start time.Time) {
	now := time.Now()
	summary := fmt.Sprintf("failed after %s: %v", time.Since(start).Round(time.Millisecond), syncErr)
	if err := u.repo.UpdateSyncStatus(ctx, cfg.KBID, now, summary); err != nil {
		u.logger.Error("save sync failure failed", log.Error(err))
	}
}

// StartSyncLoop 启动定时同步循环（每分钟检查一次到期配置），随主进程优雅退出
func (u *GitHubSyncUsecase) StartSyncLoop(ctx context.Context) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				u.logger.Error("github sync loop panicked", log.Any("panic", r))
			}
		}()
		ticker := time.NewTicker(githubSyncWaitInterval)
		defer ticker.Stop()

		var lastRun sync.Map
		sem := make(chan struct{}, githubSyncMaxConc)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cfgs, err := u.repo.ListEnabledSettings(ctx)
				if err != nil {
					u.logger.Error("list github sync settings failed", log.Error(err))
					continue
				}
				for _, cfg := range cfgs {
					if cfg.IntervalMinutes <= 0 {
						continue
					}
					var last time.Time
					if v, ok := lastRun.Load(cfg.KBID); ok {
						last = v.(time.Time)
					}
					if cfg.LastSyncAt != nil && cfg.LastSyncAt.After(last) {
						last = *cfg.LastSyncAt
					}
					if !last.IsZero() && time.Since(last) < time.Duration(cfg.IntervalMinutes)*time.Minute {
						continue
					}
					if _, busy := u.running.Load(cfg.KBID); busy {
						continue
					}
					lastRun.Store(cfg.KBID, time.Now())

					cfgCopy := *cfg
					go func(c domain.GitHubSyncSetting) {
						defer func() {
							if r := recover(); r != nil {
								u.logger.Error("scheduled github sync panicked", log.String("kb_id", c.KBID), log.Any("panic", r))
							}
						}()
						select {
						case sem <- struct{}{}:
							defer func() { <-sem }()
						case <-ctx.Done():
							return
						}
						syncCtx, cancel := context.WithTimeout(ctx, githubSyncRunTimeout)
						defer cancel()
						if _, err := u.RunSync(syncCtx, &c, "scheduled"); err != nil {
							u.logger.Error("scheduled github sync failed", log.String("kb_id", c.KBID), log.Error(err))
						}
					}(cfgCopy)
				}
			}
		}
	}()
}
