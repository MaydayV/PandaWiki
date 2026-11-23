// Package githubsync 提供 GitHub 私有仓库 MD 文章同步的基础能力：
// REST 客户端、front-matter 解析、Markdown 图片链接改写。
// 同步编排（分类映射、发布联动）在 usecase 层完成。
package githubsync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	defaultAPIBase = "https://api.github.com"
	defaultRawBase = "https://raw.githubusercontent.com"
)

var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// Client 是 GitHub REST API 客户端（github.com 固定域名，无需 SSRF 校验；
// Token 为用户配置的 PAT，仅要求 repo 只读权限即可）。
type Client struct {
	token   string
	apiBase string
	rawBase string
	http    *http.Client
}

func NewClient(token string) *Client {
	return NewClientWithBase(token, defaultAPIBase, defaultRawBase)
}

// NewClientWithBase 允许指定自定义 API/Raw Base URL（用于单元测试）。
func NewClientWithBase(token, apiBase, rawBase string) *Client {
	if apiBase == "" {
		apiBase = defaultAPIBase
	}
	if rawBase == "" {
		rawBase = defaultRawBase
	}
	return &Client{
		token:   strings.TrimSpace(token),
		apiBase: strings.TrimRight(apiBase, "/"),
		rawBase: strings.TrimRight(rawBase, "/"),
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ValidateRepoName 校验仓库标识是否符合合法的 owner/repo 格式，防止路径穿越或查询注入。
func ValidateRepoName(repo string) error {
	repo = strings.TrimSpace(repo)
	if !repoNamePattern.MatchString(repo) {
		return fmt.Errorf("repo must be in valid owner/repo format")
	}
	parts := strings.Split(repo, "/")
	for _, seg := range parts {
		if seg == "." || seg == ".." {
			return fmt.Errorf("repo must be in valid owner/repo format")
		}
	}
	return nil
}

// escapeGitHubPath 按路径段转义分支名或文件路径（保留斜杠分隔符）。
func escapeGitHubPath(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return ""
	}
	segs := strings.Split(p, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	return strings.Join(segs, "/")
}

// TreeEntry 是仓库文件树中的一项
type TreeEntry struct {
	Path string `json:"path"`
	SHA  string `json:"sha"`
	Type string `json:"type"` // blob / tree
	Size int    `json:"size"`
}

// treeResponse 是 git/trees API 的响应（截取所需字段）
type treeResponse struct {
	Tree      []TreeEntry `json:"tree"`
	Truncated bool        `json:"truncated"`
}

// GetTree 拉取指定分支的完整文件树（recursive）
func (c *Client) GetTree(ctx context.Context, repo, branch string) ([]TreeEntry, error) {
	data, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", c.apiBase, repo, escapeGitHubPath(branch)))
	if err != nil {
		return nil, err
	}
	var resp treeResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal tree response failed: %w", err)
	}
	if resp.Truncated {
		return nil, fmt.Errorf("repository tree too large (truncated), reduce file count or split repository")
	}
	return resp.Tree, nil
}

// GetRaw 拉取指定路径的原始文件内容（blob）
func (c *Client) GetRaw(ctx context.Context, repo, branch, path string, maxSize int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("%s/%s/%s/%s", c.rawBase, repo, escapeGitHubPath(branch), escapeGitHubPath(path)), nil)
	if err != nil {
		return nil, fmt.Errorf("create raw request failed: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s failed: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file %s not found on branch %s", path, branch)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s failed: status %d", path, resp.StatusCode)
	}

	// 读取内容并限制大小
	limited := io.LimitReader(resp.Body, maxSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read %s failed: %w", path, err)
	}
	if int64(len(data)) > maxSize {
		return nil, fmt.Errorf("file %s exceeds size limit (%d bytes)", path, maxSize)
	}
	return data, nil
}

// GetBase64 拉取 contents API 的 base64 编码内容（大文件与 raw.githubusercontent 分流用）
func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("github authentication failed: check token")
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("github access forbidden: token lacks read permission or rate limited")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github request failed: status %d body: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(io.LimitReader(resp.Body, 32<<20))
}

// VerifyRepo 校验仓库可达（保存配置前调用，提前暴露 token/权限问题）
func (c *Client) VerifyRepo(ctx context.Context, repo string) error {
	if err := ValidateRepoName(repo); err != nil {
		return err
	}
	_, err := c.get(ctx, fmt.Sprintf("%s/repos/%s", c.apiBase, repo))
	return err
}

// VerifyRepoAndBranch 校验仓库及目标分支均可达。
func (c *Client) VerifyRepoAndBranch(ctx context.Context, repo, branch string) error {
	if err := c.VerifyRepo(ctx, repo); err != nil {
		return err
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return fmt.Errorf("branch is required")
	}
	if _, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/branches/%s", c.apiBase, repo, escapeGitHubPath(branch))); err != nil {
		return fmt.Errorf("verify branch %q failed: %w", branch, err)
	}
	return nil
}
