package usecase

import (
	"testing"
	"time"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplateUsesDefaultsAndEscapesMarkdown(t *testing.T) {
	u := &PushUsecase{}
	kb := &domain.KnowledgeBase{Name: "Docs *A*"}
	release := &domain.KBRelease{
		Tag:       "v1.0_beta",
		Message:   "fix #1 | update",
		CreatedAt: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC),
	}

	got := u.renderTemplate("", kb, release)
	require.Contains(t, got, "Docs \\*A\\*")
	require.Contains(t, got, "v1.0\\_beta")
	require.Contains(t, got, "fix \\#1 \\| update")
	require.Contains(t, got, "2026-06-20")
}

func TestRenderTemplateCustomContent(t *testing.T) {
	u := &PushUsecase{}
	kb := &domain.KnowledgeBase{Name: "KB"}
	release := &domain.KBRelease{
		Tag:       "v2",
		Message:   "msg",
		CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	got := u.renderTemplate("更新 {kb_name}/{tag}: {message} @ {release_time}", kb, release)
	require.Equal(t, "更新 KB/v2: msg @ 2026-01-02 11:04:05", got)
}

func TestEscapeMarkdown(t *testing.T) {
	require.Equal(t, "a\\*b\\_c", escapeMarkdown("a*b_c"))
}
