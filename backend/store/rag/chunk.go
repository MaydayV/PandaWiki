package rag

import (
	"strings"

	"github.com/pkoukk/tiktoken-go"
)

const defaultChunkTokenLimit = 512

// SplitMarkdownChunks splits markdown by headings, then by token limit within each section.
func SplitMarkdownChunks(markdown string, maxTokens int) ([]string, error) {
	if maxTokens <= 0 {
		maxTokens = defaultChunkTokenLimit
	}
	text := strings.TrimSpace(markdown)
	if text == "" {
		return nil, nil
	}

	sections := splitMarkdownSections(text)
	chunks := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		parts, err := splitByTokenLimit(section, maxTokens)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, parts...)
	}
	return chunks, nil
}

func splitMarkdownSections(markdown string) []string {
	lines := strings.Split(markdown, "\n")
	var sections []string
	var current strings.Builder

	flush := func() {
		if current.Len() == 0 {
			return
		}
		sections = append(sections, current.String())
		current.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isMarkdownHeading(trimmed) && current.Len() > 0 {
			flush()
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	flush()
	if len(sections) == 0 {
		return []string{markdown}
	}
	return sections
}

func isMarkdownHeading(line string) bool {
	if line == "" {
		return false
	}
	if strings.HasPrefix(line, "#") {
		return true
	}
	if (strings.HasPrefix(line, "==") || strings.HasPrefix(line, "--")) && len(line) >= 2 {
		return true
	}
	return false
}

func splitByTokenLimit(text string, maxTokens int) ([]string, error) {
	encoding, err := tiktoken.GetEncoding("cl100k_base")
	if err != nil {
		return nil, err
	}
	tokens := encoding.Encode(text, nil, nil)
	if len(tokens) <= maxTokens {
		return []string{text}, nil
	}

	numChunks := (len(tokens) + maxTokens - 1) / maxTokens
	result := make([]string, 0, numChunks)
	for i := 0; i < len(tokens); i += maxTokens {
		end := i + maxTokens
		if end > len(tokens) {
			end = len(tokens)
		}
		result = append(result, encoding.Decode(tokens[i:end]))
	}
	return result, nil
}
