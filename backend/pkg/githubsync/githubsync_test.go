package githubsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateRepoName(t *testing.T) {
	valid := []string{"owner/repo", "my-org_1/docs.repo", "A1/B2"}
	for _, r := range valid {
		require.NoError(t, ValidateRepoName(r), r)
	}

	invalid := []string{"", "owner", "a/b/c", "./repo", "owner/..", "owner/repo?x=1", "owner/repo#frag", "owner /repo"}
	for _, r := range invalid {
		require.Error(t, ValidateRepoName(r), r)
	}
}

func TestClientGetRawTreeAndVerify(t *testing.T) {
	var rawRequestURI string
	var rawAuth string
	rawSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rawRequestURI = r.URL.EscapedPath()
		rawAuth = r.Header.Get("Authorization")
		if strings.Contains(r.URL.EscapedPath(), "too-big") {
			_, _ = w.Write([]byte("0123456789abcdef"))
			return
		}
		if strings.Contains(r.URL.EscapedPath(), "missing") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte("# Hello\n"))
	}))
	defer rawSrv.Close()

	var treeRequestURI string
	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/repos/org/wiki":
			_, _ = w.Write([]byte(`{"full_name":"org/wiki"}`))
		case "/repos/org/wiki/branches/feat/docs":
			_, _ = w.Write([]byte(`{"name":"feat/docs"}`))
		case "/repos/org/wiki/git/trees/feat/docs":
			treeRequestURI = r.URL.EscapedPath()
			_, _ = w.Write([]byte(`{"tree":[{"path":"docs/说明 #1.md","sha":"abc","type":"blob","size":10}],"truncated":false}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer apiSrv.Close()

	client := NewClientWithBase("my-pat", apiSrv.URL, rawSrv.URL)
	ctx := context.Background()

	require.NoError(t, client.VerifyRepoAndBranch(ctx, "org/wiki", "feat/docs"))
	require.Error(t, client.VerifyRepoAndBranch(ctx, "org/wiki", "non-existent"))

	tree, err := client.GetTree(ctx, "org/wiki", "feat/docs")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Equal(t, "/repos/org/wiki/git/trees/feat/docs", treeRequestURI)

	data, err := client.GetRaw(ctx, "org/wiki", "feat/docs", "docs/中文 目录/说明 #1.md", 1024)
	require.NoError(t, err)
	require.Equal(t, "# Hello\n", string(data))
	require.Equal(t, "Bearer my-pat", rawAuth)
	// 确保 branch 没有重复拼接且路径段已按 URL 规则转义
	require.Equal(t, "/org/wiki/feat/docs/docs/%E4%B8%AD%E6%96%87%20%E7%9B%AE%E5%BD%95/%E8%AF%B4%E6%98%8E%20%231.md", rawRequestURI)

	_, err = client.GetRaw(ctx, "org/wiki", "main", "missing.md", 1024)
	require.Error(t, err)

	_, err = client.GetRaw(ctx, "org/wiki", "main", "too-big.md", 5)
	require.Error(t, err)
}

func TestFrontMatterAndPathHelpers(t *testing.T) {
	md := []byte("---\ntitle: 自定义标题\ncategory: 指南\nemoji: 📘\norder: 12.5\n---\n正文 ![pic](../assets/图.png) 和 ![cdn](https://cdn.example.com/a.png)\n")
	parsed, err := ParseFrontMatter(md)
	require.NoError(t, err)
	require.True(t, parsed.HasFront)
	require.Equal(t, "自定义标题", parsed.FrontMatter.Title)
	require.Equal(t, "指南", parsed.FrontMatter.Category)
	require.Equal(t, "📘", parsed.FrontMatter.Emoji)
	require.NotNil(t, parsed.FrontMatter.Order)
	require.InDelta(t, 12.5, *parsed.FrontMatter.Order, 1e-9)

	refs := ExtractImageRefs(parsed.Body)
	require.Len(t, refs, 1)
	require.Equal(t, "../assets/图.png", refs[0].Path)
	require.Equal(t, "docs/assets/图.png", ResolveRelativePath("docs/sub", refs[0].Path))
	require.Equal(t, "assets/root.png", ResolveRelativePath("docs/sub", "/assets/root.png"))

	require.Equal(t, "README", TitleFromPath("docs/README.MD"))
	require.Equal(t, "Guide", TitleFromPath("Guide.MARKDOWN"))
	require.True(t, IsImage("a/b.PNG"))
	require.False(t, IsImage("a/b.pdf"))
	require.True(t, IsMarkdown("a/b.MD"))
	require.True(t, ShouldIgnore(".github/workflows/ci.yml"))
	require.False(t, ShouldIgnore("docs/intro.md"))
}
