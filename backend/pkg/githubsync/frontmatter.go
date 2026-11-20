package githubsync

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatter 是 MD 文件头部 YAML front-matter 的同步相关字段
type FrontMatter struct {
	Title    string   `yaml:"title,omitempty"`    // 覆盖文件名作为文档标题
	Category string   `yaml:"category,omitempty"` // 覆盖目录推导的栏目（nav 名）
	Emoji    string   `yaml:"emoji,omitempty"`
	Order    *float64 `yaml:"order,omitempty"` // 同目录内排序
	ID       string   `yaml:"id,omitempty"`    // 稳定标识（预留）
}

// ParsedDoc 是一个 MD 文件的解析结果
type ParsedDoc struct {
	FrontMatter FrontMatter
	Body        string // 去掉 front-matter 后的正文
	HasFront    bool
}

var frontMatterPattern = regexp.MustCompile(`(?s)\A---\r?\n(.*?)\r?\n---\r?\n?`)

// ParseFrontMatter 解析 YAML front-matter（无 front-matter 时整个内容为正文）
func ParseFrontMatter(content []byte) (*ParsedDoc, error) {
	m := frontMatterPattern.FindSubmatch(content)
	if m == nil {
		return &ParsedDoc{Body: string(content)}, nil
	}
	var fm FrontMatter
	if err := yaml.Unmarshal(bytes.TrimSpace(m[1]), &fm); err != nil {
		return nil, fmt.Errorf("parse front-matter failed: %w", err)
	}
	return &ParsedDoc{
		FrontMatter: fm,
		Body:        string(content[len(m[0]):]),
		HasFront:    true,
	}, nil
}

// imageRefPattern 匹配 markdown 图片引用：![alt](path "title")
var imageRefPattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)

// ImageRef 是一次图片引用
type ImageRef struct {
	Alt      string
	Path     string // MD 中的原始路径（可能相对）
	FullText string // 整个 ![..](..) 片段
}

// ExtractImageRefs 提取 MD 内容中的图片引用（仅相对/根相对路径，
// 绝对 http(s) URL 不需要转存）
func ExtractImageRefs(content string) []ImageRef {
	refs := []ImageRef{}
	for _, m := range imageRefPattern.FindAllStringSubmatch(content, -1) {
		path := m[2]
		if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") ||
			strings.HasPrefix(path, "/static-file/") {
			continue
		}
		refs = append(refs, ImageRef{Alt: m[1], Path: path, FullText: m[0]})
	}
	return refs
}

// ResolveRelativePath 解析相对路径（基于 MD 文件所在目录），
// 返回仓库内的绝对路径（无前导 /）
func ResolveRelativePath(docDir, refPath string) string {
	clean := strings.TrimPrefix(refPath, "./")
	if strings.HasPrefix(refPath, "/") {
		return strings.TrimPrefix(refPath, "/")
	}
	// 逐段处理 ../
	base := strings.Split(strings.Trim(docDir, "/"), "/")
	if docDir == "" || docDir == "." {
		base = nil
	}
	for _, seg := range strings.Split(clean, "/") {
		switch seg {
		case "..":
			if len(base) > 0 {
				base = base[:len(base)-1]
			}
		case ".", "":
			// skip
		default:
			base = append(base, seg)
		}
	}
	return strings.Join(base, "/")
}

// TitleFromPath 从文件路径推导默认标题（去目录、去扩展名）
func TitleFromPath(path string) string {
	idx := strings.LastIndex(path, "/")
	name := path
	if idx >= 0 {
		name = path[idx+1:]
	}
	name = strings.TrimSuffix(name, ".md")
	name = strings.TrimSuffix(name, ".markdown")
	return name
}

// IsMarkdown 判断路径是否为受支持的 markdown 文件
func IsMarkdown(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown")
}

// IsImage 判断路径是否为可转存的图片文件
func IsImage(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".ico"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// ShouldIgnore 判断路径是否应被同步忽略
func ShouldIgnore(path string) bool {
	if strings.HasSuffix(path, "/") || path == "" {
		return true
	}
	first := strings.SplitN(path, "/", 2)[0]
	switch first {
	case ".github", ".gitignore", ".gitattributes", "LICENSE", "LICENSE.md":
		return true
	}
	return false
}
