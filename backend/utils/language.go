package utils

import (
	"sort"
	"strconv"
	"strings"
)

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
