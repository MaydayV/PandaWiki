package utils

import (
	"context"
	"sort"
	"strconv"
	"strings"
)

type requestLanguageKey struct{}

type requestLanguageMeta struct {
	language string
	explicit bool
}

// WithRequestLanguage 将请求解析出的语言绑定到 context 中，供下层 usecase 按需读取。
func WithRequestLanguage(ctx context.Context, lang string) context.Context {
	return WithRequestLanguageMeta(ctx, lang, false)
}

// WithRequestLanguageMeta 在保留语言的同时标记是否由请求显式指定（如 query/X-Lang/cookie）。
func WithRequestLanguageMeta(ctx context.Context, lang string, explicit bool) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := NormalizeLanguageCode(lang)
	return context.WithValue(ctx, requestLanguageKey{}, requestLanguageMeta{
		language: normalized,
		explicit: explicit && normalized != "",
	})
}

// RequestLanguageFromContext 读取绑定到 context 的语言代码。
func RequestLanguageFromContext(ctx context.Context) string {
	lang, _ := RequestLanguageMetaFromContext(ctx)
	return lang
}

// RequestLanguageMetaFromContext 读取绑定到 context 的语言代码及是否由请求显式指定。
func RequestLanguageMetaFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	if meta, ok := ctx.Value(requestLanguageKey{}).(requestLanguageMeta); ok {
		return meta.language, meta.explicit
	}
	return "", false
}

// IsChineseLocale 判断给定语言代码是否为简体/繁体中文。
func IsChineseLocale(lang string) bool {
	return NormalizeLanguageCode(lang) == "zh-CN"
}

func matchShareUILocale(candidate string) string {
	switch NormalizeLanguageCode(candidate) {
	case "zh-CN":
		return "zh-CN"
	case "en-US":
		return "en-US"
	default:
		return ""
	}
}

// ResolveMatchedLocaleFromHTTP 从 HTTP 请求输入中按优先级解析匹配的前台 UI 语言（zh-CN 或 en-US），
// 并在结果中返回是否由 query/header/cookie 显式指定。未命中任何受支持语言时返回 ("", false)。
func ResolveMatchedLocaleFromHTTP(queryLang, headerXLang, cookieLang, acceptLanguage string) (string, bool) {
	for _, candidate := range []string{queryLang, headerXLang, cookieLang} {
		if matched := matchShareUILocale(candidate); matched != "" {
			return matched, true
		}
	}
	for _, accepted := range ParseAcceptLanguage(acceptLanguage) {
		if matched := matchShareUILocale(accepted); matched != "" {
			return matched, false
		}
	}
	return "", false
}

// ResolveLocaleFromHTTP 从 HTTP 请求上下文解析前台 UI 语言（仅返回 "zh-CN" 或 "en-US"）。
// 优先级：queryLang -> headerXLang -> cookieLang -> Accept-Language -> fallback -> "en-US"。
func ResolveLocaleFromHTTP(queryLang, headerXLang, cookieLang, acceptLanguage, fallback string) string {
	if matched, _ := ResolveMatchedLocaleFromHTTP(queryLang, headerXLang, cookieLang, acceptLanguage); matched != "" {
		return matched
	}
	if matched := matchShareUILocale(fallback); matched != "" {
		return matched
	}
	return "en-US"
}

func NormalizeLanguageCode(language string) string {
	token := strings.TrimSpace(language)
	if token == "" {
		return ""
	}
	if idx := strings.Index(token, ","); idx >= 0 {
		token = token[:idx]
	}
	if idx := strings.Index(token, ";"); idx >= 0 {
		token = token[:idx]
	}
	token = strings.TrimSpace(strings.ReplaceAll(token, "_", "-"))
	if token == "" || token == "*" {
		return ""
	}
	lower := strings.ToLower(token)
	switch {
	case lower == "auto":
		return ""
	case strings.HasPrefix(lower, "zh"):
		return "zh-CN"
	case strings.HasPrefix(lower, "en"):
		return "en-US"
	}

	parts := strings.Split(token, "-")
	if len(parts) <= 1 {
		return token
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i])
	}
	return strings.Join(parts, "-")
}

func ParseAcceptLanguage(acceptLanguage string) []string {
	type item struct {
		language string
		q        float64
		order    int
	}
	parts := strings.Split(acceptLanguage, ",")
	items := make([]item, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segments := strings.Split(part, ";")
		lang := NormalizeLanguageCode(segments[0])
		if lang == "" {
			continue
		}
		weight := 1.0
		for _, seg := range segments[1:] {
			seg = strings.TrimSpace(seg)
			if !strings.HasPrefix(seg, "q=") {
				continue
			}
			value := strings.TrimSpace(strings.TrimPrefix(seg, "q="))
			if q, err := strconv.ParseFloat(value, 64); err == nil {
				weight = q
			}
		}
		items = append(items, item{
			language: lang,
			q:        weight,
			order:    i,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].q == items[j].q {
			return items[i].order < items[j].order
		}
		return items[i].q > items[j].q
	})

	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if _, ok := seen[it.language]; ok {
			continue
		}
		seen[it.language] = struct{}{}
		result = append(result, it.language)
	}
	return result
}

func UniqueLanguages(languages []string) []string {
	result := make([]string, 0, len(languages))
	seen := make(map[string]struct{}, len(languages))
	for _, language := range languages {
		normalized := NormalizeLanguageCode(language)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func ContainsLanguage(languages []string, target string) bool {
	normalizedTarget := NormalizeLanguageCode(target)
	if normalizedTarget == "" {
		return false
	}
	for _, language := range languages {
		if NormalizeLanguageCode(language) == normalizedTarget {
			return true
		}
	}
	return false
}
