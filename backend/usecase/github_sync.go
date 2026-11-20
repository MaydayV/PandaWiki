package usecase

import (
	"context"
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

// SaveConfig 保存配置（校验仓库可达，token 加密落库）
func (u *GitHubSyncUsecase) SaveConfig(ctx context.Context, kbID string, req *domain.GitHubSyncSetting) error {
	req.Repo = strings.TrimSpace(req.Repo)
	if req.Repo == "" || strings.Count(req.Repo, "/") != 1 {
		return fmt.Errorf("repo must be in owner/repo format")
	}
	req.Branch = strings.TrimSpace(req.Branch)
	if req.Branch == "" {
		req.Branch = githubDefaultBranch
	}
	if req.IntervalMinutes < 0 || req.IntervalMinutes > 1440 {
		return fmt.Errorf("interval_minutes must be between 0 and 1440")
	}

	// 读取已有配置保留旧 token（前端不回传明文 token）
	existing, err := u.repo.GetSetting(ctx, kbID)
	if err != nil {
		return err
	}
	if req.Token == "" {
		if existing != nil {
			req.Token = existing.Token
		}
	} else {
		// 新 token：保存前校验可达性，提前暴露 token/权限问题
		client := githubsync.NewClient(req.Token)
		if err := client.VerifyRepo(ctx, req.Repo); err != nil {
			return fmt.Errorf("github repo verify failed: %w", err)
		}
	}

	// token 加密落库（未配置 ENCRYPTION_KEY 时透传明文，兼容既有行为）
	req.Token = utils.EncryptSecret(req.Token)

	if err := u.repo.SaveSetting(ctx, kbID, req); err != nil {
		return err
	}

	// 开关被关闭时清理该 KB 的定时循环标记
	if !req.Enabled {
		u.running.Delete(kbID)
	}
	return nil
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

	// 2. 已同步节点映射（path -> 节点）
	synced, err := u.repo.GetSyncedNodes(ctx, cfg.KBID, cfg.Repo)
	if err != nil {
		return nil, err
	}
	syncedByPath := make(map[string]pgrepo.GitHubSyncedNode, len(synced))
	syncedPaths := make(map[string]struct{}, len(synced))
	for _, n := range synced {
		syncedByPath[n.Path] = n
		syncedPaths[n.Path] = struct{}{}
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
			if n, ok := syncedByPath[cur]; ok {
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
				Repo: cfg.Repo, Path: cur,
			}); err != nil {
				return "", err
			}
			syncedByPath[cur] = pgrepo.GitHubSyncedNode{ID: nodeID, Path: cur, Type: int(domain.NodeTypeFolder)}
			result.Created++
			parentID = nodeID
		}
		return parentID, nil
	}

	// 4. 逐文件处理
	for _, mdPath := range mdPaths {
		entry := treeByPath[mdPath]

		// sha 未变且节点已存在：跳过（省 LLM embed 成本）
		if n, ok := syncedByPath[mdPath]; ok && n.SHA == entry.SHA {
			result.Skipped++
			continue
		}

		if err := u.syncOneFile(ctx, client, cfg, mdPath, entry.SHA, navByName, getOrCreateNav, getOrCreateFolders, syncedByPath, result); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", mdPath, err))
			u.logger.Error("sync md file failed", log.String("path", mdPath), log.Error(err))
		}
	}

	// 5. 孤儿删除（git 为唯一真源，且开关开启时）
	if cfg.DeleteOrphans {
		deletedDocIDs := make([]string, 0)
		for p, n := range syncedByPath {
			if _, ok := currentPaths[p]; ok {
				continue
			}
			if n.Type == int(domain.NodeTypeDocument) {
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

	// 6. 自动发布：变更节点走一次 KB release，触发 RAG 向量更新
	changed := make([]string, 0)
	for p, n := range syncedByPath {
		if n.Type == int(domain.NodeTypeDocument) {
			if entry, ok := treeByPath[p]; ok && entry.Type == "blob" {
				changed = append(changed, n.ID)
			}
		}
	}
	if cfg.AutoPublish && len(changed) > 0 {
		sort.Strings(changed)
		if _, err := u.kbUsecase.CreateKBRelease(ctx, &domain.CreateKBReleaseReq{
			KBID:    cfg.KBID,
			Message: fmt.Sprintf("GitHub sync (%s)", trigger),
			Tag:     time.Now().Format("github-20060102-150405"),
			NodeIDs: changed,
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
	navByName map[string]string,
	getOrCreateNav func(string) (string, error),
	getOrCreateFolders func(context.Context, string, string) (string, error),
	syncedByPath map[string]pgrepo.GitHubSyncedNode,
	result *domain.GitHubSyncResult,
) error {
	raw, err := client.GetRaw(ctx, cfg.Repo, cfg.Branch, mdPath, githubMaxFileSize)
	if err != nil {
		return err
	}
	doc, err := githubsync.ParseFrontMatter(raw)
	if err != nil {
		return err
	}

	// 分类识别（优先级：front-matter category > 一级目录 > 根目录归入第一个 nav）
	dir := path.Dir(mdPath)
	var navID string
	navName := doc.FrontMatter.Category
	if navName == "" && dir != "." {
		navName = strings.SplitN(dir, "/", 2)[0]
	}
	navID, err = getOrCreateNav(navName)
	if err != nil {
		return err
	}

	// 图片转存：相对路径图片从仓库拉取后上传 OSS 并改写链接
	content := doc.Body
	for _, ref := range githubsync.ExtractImageRefs(content) {
		repoImagePath := githubsync.ResolveRelativePath(dir, ref.Path)
		img, err := client.GetRaw(ctx, cfg.Repo, cfg.Branch, repoImagePath, githubMaxFileSize)
		if err != nil {
			u.logger.Warn("sync image failed, keep original link", log.String("image", repoImagePath), log.Error(err))
			continue
		}
		key, err := u.fileUsecase.UploadFileFromReader(ctx, cfg.KBID, path.Base(repoImagePath), strings.NewReader(string(img)), int64(len(img)))
		if err != nil {
			u.logger.Warn("upload synced image failed", log.String("image", repoImagePath), log.Error(err))
			continue
		}
		content = strings.Replace(content, ref.FullText,
			fmt.Sprintf("![%s](/static-file/%s)", ref.Alt, key), 1)
	}

	title := doc.FrontMatter.Title
	if title == "" {
		title = githubsync.TitleFromPath(mdPath)
	}

	// folder 链（一级目录之外的部分归属文件夹节点）
	folderDir := ""
	if dir != "." && strings.Contains(strings.Trim(dir, "/"), "/") {
		// nav 由一级目录推导时，剩余深度作为文件夹链
		folderDir = strings.SplitN(strings.Trim(dir, "/"), "/", 2)[1]
	} else if dir != "." && doc.FrontMatter.Category != "" {
		// category 指定 nav 时，完整目录作为文件夹链
		folderDir = strings.Trim(dir, "/")
	}
	parentID, err := getOrCreateFolders(ctx, folderDir, navID)
	if err != nil {
		return err
	}

	syncInfo := &domain.NodeGitHubSync{Repo: cfg.Repo, Path: mdPath, SHA: sha}
	existing, hasExisting := syncedByPath[mdPath]

	if hasExisting && existing.Type == int(domain.NodeTypeDocument) {
		// 更新
		if err := u.nodeRepo.UpdateNodeContent(ctx, &domain.UpdateNodeReq{
			ID:      existing.ID,
			KBID:    cfg.KBID,
			Name:    &title,
			Content: &content,
			Emoji:   &doc.FrontMatter.Emoji,
		}, "github_sync"); err != nil {
			return fmt.Errorf("update node failed: %w", err)
		}
		if err := u.nodeRepo.UpdateNodeGitHubSync(ctx, cfg.KBID, existing.ID, syncInfo); err != nil {
			return err
		}
		result.Updated++
		return nil
	}

	// 创建
	nodeID, err := u.nodeRepo.Create(ctx, &domain.CreateNodeReq{
		KBID:     cfg.KBID,
		NavId:    navID,
		ParentID: parentID,
		Type:     domain.NodeTypeDocument,
		Name:     title,
		Content:  content,
		Emoji:    doc.FrontMatter.Emoji,
	}, "github_sync")
	if err != nil {
		return fmt.Errorf("create node failed: %w", err)
	}
	if err := u.nodeRepo.UpdateNodeGitHubSync(ctx, cfg.KBID, nodeID, syncInfo); err != nil {
		return err
	}
	syncedByPath[mdPath] = pgrepo.GitHubSyncedNode{ID: nodeID, Path: mdPath, SHA: sha, Type: int(domain.NodeTypeDocument)}
	result.Created++
	return nil
}

// recordSyncSuccess 回写成功状态
func (u *GitHubSyncUsecase) recordSyncSuccess(ctx context.Context, cfg *domain.GitHubSyncSetting, result *domain.GitHubSyncResult) {
	now := time.Now()
	cfg.LastSyncAt = &now
	summary := fmt.Sprintf("created=%d updated=%d skipped=%d deleted=%d failed=%d (%s)",
		result.Created, result.Updated, result.Skipped, result.Deleted, result.Failed, result.Duration)
	if len(result.Errors) > 0 {
		summary += " errors: " + strings.Join(result.Errors, "; ")
	}
	cfg.LastSyncResult = summary
	if err := u.repo.SaveSetting(ctx, cfg.KBID, cfg); err != nil {
		u.logger.Error("save sync result failed", log.Error(err))
	}
}

// recordSyncFailure 回写失败状态
func (u *GitHubSyncUsecase) recordSyncFailure(ctx context.Context, cfg *domain.GitHubSyncSetting, syncErr error, start time.Time) {
	now := time.Now()
	cfg.LastSyncAt = &now
	cfg.LastSyncResult = fmt.Sprintf("failed after %s: %v", time.Since(start).Round(time.Millisecond), syncErr)
	if err := u.repo.SaveSetting(ctx, cfg.KBID, cfg); err != nil {
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

		lastRun := make(map[string]time.Time)
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
					if last, ok := lastRun[cfg.KBID]; ok && time.Since(last) < time.Duration(cfg.IntervalMinutes)*time.Minute {
						continue
					}
					lastRun[cfg.KBID] = time.Now()
					if _, err := u.RunSync(ctx, cfg, "scheduled"); err != nil {
						u.logger.Error("scheduled github sync failed", log.String("kb_id", cfg.KBID), log.Error(err))
					}
				}
			}
		}
	}()
}
