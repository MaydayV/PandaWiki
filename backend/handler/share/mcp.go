package share

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/usecase"
)

const (
	defaultMCPToolName = "get_docs"
	defaultMCPToolDesc = "为解决用户的问题从知识库中检索文档"
	mcpProtocolVersion = "2024-11-05"
	mcpResultLimit     = 5
)

type ShareMCPHandler struct {
	*handler.BaseHandler
	logger     *log.Logger
	appUsecase *usecase.AppUsecase
	mcpRepo    *pg.MCPRepository
}

func NewShareMCPHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	appUsecase *usecase.AppUsecase,
	mcpRepo *pg.MCPRepository,
) *ShareMCPHandler {
	h := &ShareMCPHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.share.mcp"),
		appUsecase:  appUsecase,
		mcpRepo:     mcpRepo,
	}
	e.POST("/mcp", h.HandleMCP)
	return h
}

type mcpJSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpJSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      any              `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *mcpJSONRPCError `json:"error,omitempty"`
}

type mcpJSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpInitializeResult struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	Capabilities    map[string]any         `json:"capabilities"`
	ServerInfo      mcpInitializeServerRef `json:"serverInfo"`
}

type mcpInitializeServerRef struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpToolsCallResult struct {
	Content []mcpTextContent `json:"content"`
	IsError bool             `json:"isError,omitempty"`
}

type mcpTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (h *ShareMCPHandler) HandleMCP(c echo.Context) error {
	kbID := strings.TrimSpace(c.Request().Header.Get("X-KB-ID"))
	if kbID == "" {
		return c.JSON(http.StatusBadRequest, h.newJSONRPCError(nil, -32600, "X-KB-ID header is required"))
	}

	var req mcpJSONRPCRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, h.newJSONRPCError(nil, -32700, "invalid JSON-RPC request"))
	}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		return c.JSON(http.StatusBadRequest, h.newJSONRPCError(req.ID, -32600, "invalid JSON-RPC request"))
	}

	appInfo, err := h.appUsecase.GetMCPServerAppInfo(c.Request().Context(), kbID)
	if err != nil {
		h.logger.Error("get mcp app info failed", log.Error(err), log.String("kb_id", kbID))
		return c.JSON(http.StatusOK, h.newJSONRPCError(req.ID, -32000, "failed to load mcp settings"))
	}
	settings := appInfo.Settings.MCPServerSettings
	if !settings.IsEnabled {
		return c.JSON(http.StatusForbidden, h.newJSONRPCError(req.ID, -32001, "mcp server is disabled"))
	}
	if settings.SampleAuth.Enabled && !h.validateSampleAuth(c, settings.SampleAuth.Password) {
		return c.JSON(http.StatusUnauthorized, h.newJSONRPCError(req.ID, -32001, "unauthorized"))
	}

	sessionID := h.resolveSessionID(c)
	c.Response().Header().Set("Mcp-Session-Id", sessionID)

	toolName, toolDesc := h.getToolSettings(settings.DocsToolSettings.Name, settings.DocsToolSettings.Desc)
	switch req.Method {
	case "initialize":
		resp := mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcpInitializeResult{
				ProtocolVersion: mcpProtocolVersion,
				Capabilities: map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
				ServerInfo: mcpInitializeServerRef{
					Name:    "PandaWiki MCP Server",
					Version: "1.0.0",
				},
			},
		}
		if err := h.mcpRepo.LogInitializeCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); err != nil {
			h.logger.Error("log mcp initialize call failed", log.Error(err), log.String("kb_id", kbID))
		}
		return c.JSON(http.StatusOK, resp)
	case "tools/list":
		resp := mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcpToolsListResult{
				Tools: []mcpTool{
					{
						Name:        toolName,
						Description: toolDesc,
						InputSchema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"query": map[string]any{
									"type":        "string",
									"description": "keywords to search released docs",
								},
							},
							"required": []string{"query"},
						},
					},
				},
			},
		}
		return c.JSON(http.StatusOK, resp)
	case "tools/call":
		var params mcpToolsCallParams
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				resp := h.newJSONRPCError(req.ID, -32602, "invalid tools/call params")
				if logErr := h.mcpRepo.LogToolCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); logErr != nil {
					h.logger.Error("log mcp tool call failed", log.Error(logErr), log.String("kb_id", kbID))
				}
				return c.JSON(http.StatusOK, resp)
			}
		}

		if params.Name != toolName {
			resp := h.newJSONRPCError(req.ID, -32602, fmt.Sprintf("unsupported tool: %s", params.Name))
			if logErr := h.mcpRepo.LogToolCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); logErr != nil {
				h.logger.Error("log mcp tool call failed", log.Error(logErr), log.String("kb_id", kbID))
			}
			return c.JSON(http.StatusOK, resp)
		}

		query := strings.TrimSpace(anyToString(params.Arguments["query"]))
		if query == "" {
			resp := h.newJSONRPCError(req.ID, -32602, "tools/call.arguments.query is required")
			if logErr := h.mcpRepo.LogToolCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); logErr != nil {
				h.logger.Error("log mcp tool call failed", log.Error(logErr), log.String("kb_id", kbID))
			}
			return c.JSON(http.StatusOK, resp)
		}

		docs, err := h.mcpRepo.SearchReleasedDocs(c.Request().Context(), kbID, query, mcpResultLimit)
		if err != nil {
			h.logger.Error("search released docs failed", log.Error(err), log.String("kb_id", kbID), log.String("query", query))
			resp := h.newJSONRPCError(req.ID, -32000, "failed to search docs")
			if logErr := h.mcpRepo.LogToolCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); logErr != nil {
				h.logger.Error("log mcp tool call failed", log.Error(logErr), log.String("kb_id", kbID))
			}
			return c.JSON(http.StatusOK, resp)
		}

		resp := mcpJSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: mcpToolsCallResult{
				Content: []mcpTextContent{
					{
						Type: "text",
						Text: renderDocSearchResult(query, docs),
					},
				},
			},
		}
		if err := h.mcpRepo.LogToolCall(c.Request().Context(), sessionID, kbID, c.RealIP(), req, resp); err != nil {
			h.logger.Error("log mcp tool call failed", log.Error(err), log.String("kb_id", kbID))
		}
		return c.JSON(http.StatusOK, resp)
	default:
		return c.JSON(http.StatusOK, h.newJSONRPCError(req.ID, -32601, "method not found"))
	}
}

func (h *ShareMCPHandler) newJSONRPCError(id any, code int, message string) mcpJSONRPCResponse {
	return mcpJSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcpJSONRPCError{
			Code:    code,
			Message: message,
		},
	}
}

func (h *ShareMCPHandler) getToolSettings(name, desc string) (string, string) {
	toolName := strings.TrimSpace(name)
	if toolName == "" {
		toolName = defaultMCPToolName
	}
	toolDesc := strings.TrimSpace(desc)
	if toolDesc == "" {
		toolDesc = defaultMCPToolDesc
	}
	return toolName, toolDesc
}

func (h *ShareMCPHandler) resolveSessionID(c echo.Context) string {
	sessionID := strings.TrimSpace(c.Request().Header.Get("Mcp-Session-Id"))
	if sessionID != "" {
		return sessionID
	}
	sessionID = strings.TrimSpace(c.Request().Header.Get("X-MCP-Session-ID"))
	if sessionID != "" {
		return sessionID
	}
	return uuid.NewString()
}

func (h *ShareMCPHandler) validateSampleAuth(c echo.Context, expectedPassword string) bool {
	expectedPassword = strings.TrimSpace(expectedPassword)
	if expectedPassword == "" {
		return false
	}

	authHeader := strings.TrimSpace(c.Request().Header.Get("Authorization"))
	if token, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return strings.TrimSpace(token) == expectedPassword
	}

	rawToken := strings.TrimSpace(c.Request().Header.Get("X-MCP-Token"))
	if rawToken != "" {
		return rawToken == expectedPassword
	}

	queryToken := strings.TrimSpace(c.QueryParam("token"))
	if queryToken != "" {
		return queryToken == expectedPassword
	}
	return false
}

func anyToString(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func renderDocSearchResult(query string, docs []*pg.MCPDocSearchResult) string {
	if len(docs) == 0 {
		return fmt.Sprintf("No released docs found for query: %s", query)
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Found %d released docs for query: %s\n\n", len(docs), query))
	for i, doc := range docs {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, strings.TrimSpace(doc.Name)))
		b.WriteString(fmt.Sprintf("node_id: %s\n", strings.TrimSpace(doc.NodeID)))
		b.WriteString(fmt.Sprintf("url: /node/%s\n", strings.TrimSpace(doc.NodeID)))
		if summary := compactAndTruncate(doc.Summary, 160); summary != "" {
			b.WriteString(fmt.Sprintf("summary: %s\n", summary))
		}
		if snippet := compactAndTruncate(doc.Content, 220); snippet != "" {
			b.WriteString(fmt.Sprintf("snippet: %s\n", snippet))
		}
		if i != len(docs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func compactAndTruncate(s string, limit int) string {
	compacted := strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if limit <= 0 {
		return compacted
	}
	runes := []rune(compacted)
	if len(runes) <= limit {
		return compacted
	}
	return string(runes[:limit]) + "..."
}
