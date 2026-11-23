package utils

import (
	"context"
	"testing"
)

func TestResolveLocaleFromHTTP(t *testing.T) {
	tests := []struct {
		name           string
		queryLang      string
		headerXLang    string
		cookieLang     string
		acceptLanguage string
		fallback       string
		want           string
	}{
		{
			name:           "query wins over all",
			queryLang:      "zh-TW",
			headerXLang:    "en-US",
			cookieLang:     "en-US",
			acceptLanguage: "en-US,en;q=0.9",
			fallback:       "en-US",
			want:           "zh-CN",
		},
		{
			name:           "x-lang header wins over cookie and accept-language",
			headerXLang:    "en_GB",
			cookieLang:     "zh-CN",
			acceptLanguage: "zh-CN,zh;q=0.9",
			fallback:       "zh-CN",
			want:           "en-US",
		},
		{
			name:           "cookie wins over accept-language",
			cookieLang:     "zh-CN",
			acceptLanguage: "en-US,en;q=0.9",
			fallback:       "en-US",
			want:           "zh-CN",
		},
		{
			name:           "accept-language honors q weights",
			acceptLanguage: "fr-FR,fr;q=0.9,zh-CN;q=0.8,en-US;q=0.7",
			fallback:       "en-US",
			want:           "zh-CN",
		},
		{
			name:           "accept-language higher q en overrides zh",
			acceptLanguage: "zh-CN;q=0.6,en-US;q=0.9",
			fallback:       "zh-CN",
			want:           "en-US",
		},
		{
			name:           "unsupported accept-language falls back to fallback",
			acceptLanguage: "ja-JP,fr-FR;q=0.8",
			fallback:       "zh-CN",
			want:           "zh-CN",
		},
		{
			name:           "empty everything defaults to en-US",
			acceptLanguage: "",
			fallback:       "",
			want:           "en-US",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLocaleFromHTTP(tc.queryLang, tc.headerXLang, tc.cookieLang, tc.acceptLanguage, tc.fallback)
			if got != tc.want {
				t.Fatalf("ResolveLocaleFromHTTP() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRequestLanguageContextAndIsChineseLocale(t *testing.T) {
	if got := RequestLanguageFromContext(context.TODO()); got != "" {
		t.Fatalf("empty context should return empty string, got %q", got)
	}

	ctx := WithRequestLanguage(context.Background(), "zh_HK")
	if got := RequestLanguageFromContext(ctx); got != "zh-CN" {
		t.Fatalf("RequestLanguageFromContext = %q, want zh-CN", got)
	}
	if _, explicit := RequestLanguageMetaFromContext(ctx); explicit {
		t.Fatalf("WithRequestLanguage should not mark explicit=true")
	}

	explicitCtx := WithRequestLanguageMeta(context.Background(), "en-GB", true)
	lang, explicit := RequestLanguageMetaFromContext(explicitCtx)
	if lang != "en-US" || !explicit {
		t.Fatalf("RequestLanguageMetaFromContext = (%q, %v), want (en-US, true)", lang, explicit)
	}

	if !IsChineseLocale("zh-CN") || !IsChineseLocale("zh-tw") {
		t.Fatalf("expected IsChineseLocale true for Chinese tags")
	}
	if IsChineseLocale("en-US") || IsChineseLocale("") || IsChineseLocale("ja-JP") {
		t.Fatalf("expected IsChineseLocale false for non-Chinese tags")
	}
}
