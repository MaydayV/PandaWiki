package usecase

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeExportFilename(t *testing.T) {
	require.Equal(t, "untitled", sanitizeExportFilename("  "))
	require.Equal(t, "文档_v1", sanitizeExportFilename("文档/v1"))
	require.Equal(t, "a_b_c", sanitizeExportFilename("a\\b:c"))
}

func TestWrapHTMLDocumentEscapesTitle(t *testing.T) {
	got := wrapHTMLDocument(`A <b>"title"</b> & more`, "<p>body</p>")
	require.Contains(t, got, "<title>A &lt;b&gt;&quot;title&quot;&lt;/b&gt; &amp; more</title>")
	require.Contains(t, got, "<p>body</p>")
	require.True(t, strings.HasPrefix(got, "<!DOCTYPE html>"))
}

func TestConvertHTMLToMarkdown(t *testing.T) {
	got, err := convertHTMLToMarkdown("<p>hello <strong>world</strong></p>")
	require.NoError(t, err)
	require.Contains(t, got, "hello")
	require.Contains(t, got, "**world**")
}

func TestConvertMDToHTML(t *testing.T) {
	u := &NodeUsecase{}
	got := u.convertMDToHTML("# hello\n\n**world**")
	require.Contains(t, got, "<h1")
	require.Contains(t, got, "hello")
	require.Contains(t, got, "<strong>world</strong>")
}
