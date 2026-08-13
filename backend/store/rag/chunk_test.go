package rag

import (
	"strings"
	"testing"

	"github.com/chaitin/panda-wiki/utils"
	"github.com/pkoukk/tiktoken-go"
)

func TestSplitMarkdownChunks(t *testing.T) {
	tiktoken.SetBpeLoader(&utils.Localloader{})

	chunks, err := SplitMarkdownChunks("# Title\n\nparagraph one.\n\n## Section\n\nparagraph two.", 512)
	if err != nil {
		t.Fatalf("SplitMarkdownChunks failed: %v", err)
	}
	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0], "Title") {
		t.Fatalf("first chunk should contain title section")
	}
}

func TestSplitMarkdownChunksEmpty(t *testing.T) {
	chunks, err := SplitMarkdownChunks("   ", 512)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(chunks) != 0 {
		t.Fatalf("expected no chunks for empty input")
	}
}
