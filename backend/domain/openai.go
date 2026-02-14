package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAI API 请求结构体
type OpenAICompletionsRequest struct {
	Model            string                `json:"model" validate:"required"`
	Messages         []OpenAIMessage       `json:"messages" validate:"required"`
	Stream           bool                  `json:"stream,omitempty"`
	StreamOptions    *OpenAIStreamOptions  `json:"stream_options,omitempty"`
	Temperature      *float64              `json:"temperature,omitempty"`
	MaxTokens        *int                  `json:"max_tokens,omitempty"`
	TopP             *float64              `json:"top_p,omitempty"`
	FrequencyPenalty *float64              `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64              `json:"presence_penalty,omitempty"`
	Stop             []string              `json:"stop,omitempty"`
	User             string                `json:"user,omitempty"`
	Tools            []OpenAITool          `json:"tools,omitempty"`
	ToolChoice       *OpenAIToolChoice     `json:"tool_choice,omitempty"`
	ResponseFormat   *OpenAIResponseFormat `json:"response_format,omitempty"`
}

type OpenAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// MessageContent 支持字符串或内容数组
type MessageContent struct {
	isString bool
	strValue string
	arrValue []OpenAIContentPart
}

// OpenAIContentPart 表示内容数组中的单个元素
type OpenAIContentPart struct {
	Type     string                `json:"type"`
	Text     string                `json:"text,omitempty"`
	ImageURL *OpenAIContentPartURL `json:"image_url,omitempty"`
}

// OpenAIContentPartURL represents the image_url field in content parts
type OpenAIContentPartURL struct {
	URL string `json:"url"`
}

// UnmarshalJSON 自定义解析，支持 string 或 array 格式
func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	// 尝试解析为字符串
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		mc.isString = true
		mc.strValue = str
		return nil
	}

	// 尝试解析为数组
	var arr []OpenAIContentPart
	if err := json.Unmarshal(data, &arr); err == nil {
		mc.isString = false
		mc.arrValue = arr
		return nil
	}

	return fmt.Errorf("content must be string or array")
}

// MarshalJSON 自定义序列化
func (mc *MessageContent) MarshalJSON() ([]byte, error) {
	if mc.isString {
		return json.Marshal(mc.strValue)
	}
	return json.Marshal(mc.arrValue)
}

// NewStringContent 创建字符串类型的 MessageContent
func NewStringContent(s string) *MessageContent {
	return &MessageContent{
		isString: true,
		strValue: s,
	}
}

// NewArrayContent 创建数组类型的 MessageContent
func NewArrayContent(parts []OpenAIContentPart) *MessageContent {
	return &MessageContent{
		isString: false,
		arrValue: parts,
	}
}

// String 获取文本内容
func (mc *MessageContent) String() string {
	if mc.isString {
		return mc.strValue
	}
	// 从数组中提取文本
	var builder strings.Builder
	for _, part := range mc.arrValue {
		if part.Type == "text" {
			if builder.Len() > 0 && part.Text != "" {
				builder.WriteString(" ")
			}
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

// StringWithImages returns content text plus lightweight image_url markers.
func (mc *MessageContent) StringWithImages() string {
	if mc.isString {
		return mc.strValue
	}

	var builder strings.Builder
	for _, part := range mc.arrValue {
		switch part.Type {
		case "text":
			if builder.Len() > 0 && part.Text != "" {
				builder.WriteString(" ")
			}
			builder.WriteString(part.Text)
		case "image_url":
			if part.ImageURL == nil || strings.TrimSpace(part.ImageURL.URL) == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString(" ")
			}
			builder.WriteString("[image_url:")
			builder.WriteString(strings.TrimSpace(part.ImageURL.URL))
			builder.WriteString("]")
		}
	}
	return builder.String()
}

type OpenAIMessage struct {
	Role       string           `json:"role" validate:"required"`
	Content    *MessageContent  `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type OpenAITool struct {
	Type     string          `json:"type" validate:"required"`
	Function *OpenAIFunction `json:"function,omitempty"`
}

type OpenAIFunction struct {
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type OpenAIToolCall struct {
	ID       string             `json:"id" validate:"required"`
	Type     string             `json:"type" validate:"required"`
	Function OpenAIFunctionCall `json:"function" validate:"required"`
}

type OpenAIFunctionCall struct {
	Name      string `json:"name" validate:"required"`
	Arguments string `json:"arguments" validate:"required"`
}

type OpenAIToolChoice struct {
	Type     string                `json:"type,omitempty"`
	Function *OpenAIFunctionChoice `json:"function,omitempty"`
}

type OpenAIFunctionChoice struct {
	Name string `json:"name" validate:"required"`
}

type OpenAIResponseFormat struct {
	Type string `json:"type" validate:"required"`
}

// OpenAI API 响应结构体
type OpenAICompletionsResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   *OpenAIUsage   `json:"usage,omitempty"`
}

type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      OpenAIMessage  `json:"message"`
	FinishReason string         `json:"finish_reason"`
	Delta        *OpenAIMessage `json:"delta,omitempty"` // for streaming
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAI 流式响应结构体
type OpenAIStreamResponse struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

type OpenAIStreamChoice struct {
	Index        int           `json:"index"`
	Delta        OpenAIMessage `json:"delta"`
	FinishReason *string       `json:"finish_reason,omitempty"`
}

// APICallAudit records OpenAI-compatible API call audit info.
type APICallAudit struct {
	KBID             string  `json:"kb_id"`
	APITokenID       *string `json:"api_token_id,omitempty"`
	Endpoint         string  `json:"endpoint"`
	Model            string  `json:"model"`
	StatusCode       int     `json:"status_code"`
	ErrorType        string  `json:"error_type,omitempty"`
	ErrorMessage     string  `json:"error_message,omitempty"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	LatencyMS        int64   `json:"latency_ms"`
	RemoteIP         string  `json:"remote_ip"`
	RequestID        *string `json:"request_id,omitempty"`
}

func (APICallAudit) TableName() string {
	return "api_call_audits"
}

// OpenAI 错误响应结构体
type OpenAIErrorResponse struct {
	Error OpenAIError `json:"error"`
}

type OpenAIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
	Param   string `json:"param,omitempty"`
}
