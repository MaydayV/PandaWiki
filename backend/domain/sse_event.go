package domain

type SSEEvent struct {
	Type        string               `json:"type"`
	Content     string               `json:"content"`
	ChunkResult *NodeContentChunkSSE `json:"chunk_result,omitempty"`
	Usage       *SSETokenUsage       `json:"usage,omitempty"`
	Error       string               `json:"error,omitempty"`
}

type SSETokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
