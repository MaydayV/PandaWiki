package share

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	pgrepo "github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/usecase"
)

const openAICompletionsEndpoint = "/share/v1/chat/completions"

type ShareChatHandler struct {
	*handler.BaseHandler
	logger              *log.Logger
	appUsecase          *usecase.AppUsecase
	chatUsecase         *usecase.ChatUsecase
	authUsecase         *usecase.AuthUsecase
	conversationUsecase *usecase.ConversationUsecase
	modelUsecase        *usecase.ModelUsecase
	apiTokenRepo        *pgrepo.APITokenRepo
	apiCallAuditRepo    *pgrepo.APICallAuditRepo
}

func NewShareChatHandler(
	e *echo.Echo,
	baseHandler *handler.BaseHandler,
	logger *log.Logger,
	appUsecase *usecase.AppUsecase,
	chatUsecase *usecase.ChatUsecase,
	authUsecase *usecase.AuthUsecase,
	conversationUsecase *usecase.ConversationUsecase,
	modelUsecase *usecase.ModelUsecase,
	apiTokenRepo *pgrepo.APITokenRepo,
	apiCallAuditRepo *pgrepo.APICallAuditRepo,
) *ShareChatHandler {
	h := &ShareChatHandler{
		BaseHandler:         baseHandler,
		logger:              logger.WithModule("handler.share.chat"),
		appUsecase:          appUsecase,
		chatUsecase:         chatUsecase,
		authUsecase:         authUsecase,
		conversationUsecase: conversationUsecase,
		modelUsecase:        modelUsecase,
		apiTokenRepo:        apiTokenRepo,
		apiCallAuditRepo:    apiCallAuditRepo,
	}

	share := e.Group("share/v1/chat",
		func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Access-Control-Allow-Origin", "*")
				c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Origin, Accept")
				if c.Request().Method == "OPTIONS" {
					return c.NoContent(http.StatusOK)
				}
				return next(c)
			}
		})
	share.POST("/message", h.ChatMessage, h.ShareAuthMiddleware.Authorize)
	share.POST("/search", h.ChatSearch, h.ShareAuthMiddleware.Authorize)
	share.POST("/completions", h.ChatCompletions)
	share.POST("/widget", h.ChatWidget)
	share.POST("/widget/search", h.WidgetSearch)
	share.POST("/feedback", h.FeedBack)
	return h
}

// ChatMessage chat message
//
//	@Summary		ChatMessage
//	@Description	ChatMessage
//	@Tags			share_chat
//	@Accept			json
//	@Produce		json
//	@Param			app_type	query		string				true	"app type"
//	@Param			request		body		domain.ChatRequest	true	"request"
//	@Success		200			{object}	domain.Response
//	@Router			/share/v1/chat/message [post]
func (h *ShareChatHandler) ChatMessage(c echo.Context) error {
	var req domain.ChatRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("parse request failed", log.Error(err))
		return h.sendErrMsg(c, "parse request failed")
	}
	req.KBID = c.Request().Header.Get("X-KB-ID") // get from caddy header
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validate request failed", log.Error(err))
		return h.sendErrMsg(c, "validate request failed")
	}

	for _, path := range req.ImagePaths {
		if !strings.HasPrefix(path, "/static-file/") {
			return h.sendErrMsg(c, "invalid image path")
		}
	}

	if req.Message == "" && len(req.ImagePaths) == 0 {
		return h.sendErrMsg(c, "message is empty")
	}

	if req.AppType != domain.AppTypeWeb {
		return h.sendErrMsg(c, "invalid app type")
	}
	ctx := c.Request().Context()
	// validate captcha token
	if !h.Captcha.ValidateToken(ctx, req.CaptchaToken) {
		return h.sendErrMsg(c, "failed to validate captcha")
	}

	req.RemoteIP = c.RealIP()

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Transfer-Encoding", "chunked")

	// get user info --> no enterprise is nil
	userID := c.Get("user_id")
	h.logger.Debug("userid:", userID)
	if userID != nil { // find userinfo from auth
		userIDValue := userID.(uint)
		req.Info.UserInfo.AuthUserID = userIDValue
	}

	eventCh, err := h.chatUsecase.Chat(ctx, &req)
	if err != nil {
		return h.sendErrMsg(c, err.Error())
	}

	for event := range eventCh {
		if err := h.writeSSEEvent(c, event); err != nil {
			return err
		}
		if event.Type == "done" || event.Type == "error" {
			break
		}
	}
	return nil
}

// ChatWidget chat widget
//
//	@Summary		ChatWidget
//	@Description	ChatWidget
//	@Tags			Widget
//	@Accept			json
//	@Produce		json
//	@Param			app_type	query		string				true	"app type"
//	@Param			request		body		domain.ChatRequest	true	"request"
//	@Success		200			{object}	domain.Response
//	@Router			/share/v1/chat/widget [post]
func (h *ShareChatHandler) ChatWidget(c echo.Context) error {
	var req domain.ChatRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("parse request failed", log.Error(err))
		return h.sendErrMsg(c, "parse request failed")
	}
	req.KBID = c.Request().Header.Get("X-KB-ID") // get from caddy header
	if err := c.Validate(&req); err != nil {
		h.logger.Error("validate request failed", log.Error(err))
		return h.sendErrMsg(c, "validate request failed")
	}
	if req.AppType != domain.AppTypeWidget {
		return h.sendErrMsg(c, "invalid app type")
	}
	if req.Message == "" && len(req.ImagePaths) == 0 {
		return h.sendErrMsg(c, "message is empty")
	}
	for _, path := range req.ImagePaths {
		if !strings.HasPrefix(path, "/static-file/") {
			return h.sendErrMsg(c, "invalid image path")
		}
	}

	// get widget app info
	widgetAppInfo, err := h.appUsecase.GetWidgetAppInfo(c.Request().Context(), req.KBID)
	if err != nil {
		h.logger.Error("get widget app info failed", log.Error(err))
		return h.sendErrMsg(c, "get app info error")
	}
	if !widgetAppInfo.Settings.WidgetBotSettings.IsOpen {
		return h.sendErrMsg(c, "widget is not open")
	}

	req.RemoteIP = c.RealIP()

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().Header().Set("Transfer-Encoding", "chunked")

	eventCh, err := h.chatUsecase.Chat(c.Request().Context(), &req)
	if err != nil {
		return h.sendErrMsg(c, err.Error())
	}

	for event := range eventCh {
		if err := h.writeSSEEvent(c, event); err != nil {
			return err
		}
		if event.Type == "done" || event.Type == "error" {
			break
		}
	}
	return nil
}

func (h *ShareChatHandler) sendErrMsg(c echo.Context, errMsg string) error {
	return h.writeSSEEvent(c, domain.SSEEvent{Type: "error", Content: errMsg})
}

func (h *ShareChatHandler) writeSSEEvent(c echo.Context, data any) error {
	jsonContent, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sseMessage := fmt.Sprintf("data: %s\n\n", string(jsonContent))
	if _, err := c.Response().Write([]byte(sseMessage)); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

// FeedBack handle chat feedback
//
//	@Summary		Handle chat feedback
//	@Description	Process user feedback for chat conversations
//	@Tags			share_chat
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.FeedbackRequest	true	"feedback request"
//	@Success		200		{object}	domain.Response
//	@Router			/share/v1/chat/feedback [post]
func (h *ShareChatHandler) FeedBack(c echo.Context) error {
	// 前端传入对应的conversationId和feedback内容，后端处理并返回反馈结果
	var feedbackReq domain.FeedbackRequest
	if err := c.Bind(&feedbackReq); err != nil {
		return h.NewResponseWithError(c, "bind feedback request failed", err)
	}
	if err := c.Validate(&feedbackReq); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}
	h.logger.Debug("receive feedback request:", log.Any("feedback_request", feedbackReq))
	if err := h.conversationUsecase.FeedBack(c.Request().Context(), &feedbackReq); err != nil {
		return h.NewResponseWithError(c, "handle feedback failed", err)
	}
	return h.NewResponseWithData(c, "success")
}

// ChatCompletions OpenAI API compatible chat completions
//
//	@Summary		ChatCompletions
//	@Description	OpenAI API compatible chat completions endpoint
//	@Tags			share_chat
//	@Accept			json
//	@Produce		json
//	@Param			X-KB-ID	header		string							true	"Knowledge Base ID"
//	@Param			request	body		domain.OpenAICompletionsRequest	true	"OpenAI API request"
//	@Success		200		{object}	domain.OpenAICompletionsResponse
//	@Failure		400		{object}	domain.OpenAIErrorResponse
//	@Router			/share/v1/chat/completions [post]
func (h *ShareChatHandler) ChatCompletions(c echo.Context) error {
	startedAt := time.Now()
	auditMeta := openAIAuditMeta{
		KBID:      strings.TrimSpace(c.Request().Header.Get("X-KB-ID")),
		Endpoint:  openAICompletionsEndpoint,
		RemoteIP:  c.RealIP(),
		RequestID: h.resolveRequestID(c),
	}

	var req domain.OpenAICompletionsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("parse OpenAI request failed", log.Error(err))
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "parse request failed", "invalid_request_error", startedAt)
	}
	auditMeta.Model = strings.TrimSpace(req.Model)

	// get kb id from header
	kbID := auditMeta.KBID
	if kbID == "" {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "X-KB-ID header is required", "invalid_request_error", startedAt)
	}

	if err := c.Validate(&req); err != nil {
		h.logger.Error("validate OpenAI request failed", log.Error(err))
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "validate request failed", "invalid_request_error", startedAt)
	}

	// validate messages
	if len(req.Messages) == 0 {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "messages cannot be empty", "invalid_request_error", startedAt)
	}

	promptMessage, err := buildOpenAIChatMessage(req.Messages)
	if err != nil {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "no user message found", "invalid_request_error", startedAt)
	}

	// validate api bot settings
	appBot, err := h.appUsecase.GetOpenAIAPIAppInfo(c.Request().Context(), kbID)
	if err != nil {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, err.Error(), "internal_error", startedAt)
	}
	if !appBot.Settings.OpenAIAPIBotSettings.IsEnabled {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "API Bot is not enabled", "forbidden", startedAt)
	}

	secretKeyHeader := c.Request().Header.Get("Authorization")
	if secretKeyHeader == "" {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "Authorization header is required", "invalid_request_error", startedAt)
	}
	secretKey, found := strings.CutPrefix(secretKeyHeader, "Bearer ")
	if !found {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "Invalid Authorization key format", "invalid_request_error", startedAt)
	}
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "Invalid Authorization key format", "invalid_request_error", startedAt)
	}

	apiToken := h.resolveAPIToken(c.Request().Context(), secretKey, kbID)
	if apiToken != nil {
		tokenID := apiToken.ID
		auditMeta.APITokenID = &tokenID
	}

	authorizedByAppSecret := appBot.Settings.OpenAIAPIBotSettings.SecretKey == secretKey
	authorizedByAPIToken := apiToken != nil
	if !authorizedByAppSecret && !authorizedByAPIToken {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, "Invalid Authorization key", "unauthorized", startedAt)
	}

	if authorizedByAPIToken {
		if errorType, err := h.checkAPITokenGovernance(c.Request().Context(), apiToken, startedAt); err != nil {
			h.logger.Warn("check api token governance failed", log.Error(err))
		} else if errorType != "" {
			return h.sendOpenAIErrorWithAudit(c, auditMeta, openAIGovernanceErrorMessage(errorType), errorType, startedAt)
		}
	}

	chatReq := &domain.ChatRequest{
		Message:  promptMessage,
		KBID:     kbID,
		AppType:  domain.AppTypeOpenAIAPI,
		RemoteIP: c.RealIP(),
	}

	// set stream response header
	if req.Stream {
		c.Response().Header().Set("Content-Type", "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().Header().Set("Transfer-Encoding", "chunked")
	}

	eventCh, err := h.chatUsecase.Chat(c.Request().Context(), chatReq)
	if err != nil {
		return h.sendOpenAIErrorWithAudit(c, auditMeta, err.Error(), "internal_error", startedAt)
	}

	// handle stream response
	if req.Stream {
		return h.handleOpenAIStreamResponse(c, eventCh, auditMeta, req.StreamOptions, startedAt)
	} else {
		return h.handleOpenAINonStreamResponse(c, eventCh, auditMeta, startedAt)
	}
}

func buildOpenAIChatMessage(messages []domain.OpenAIMessage) (string, error) {
	systemMessages := make([]string, 0)
	historyMessages := make([]string, 0)
	lastUserMessage := ""

	for idx, message := range messages {
		content := ""
		if message.Content != nil {
			content = strings.TrimSpace(message.Content.StringWithImages())
		}
		if content == "" {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(message.Role)) {
		case "system":
			systemMessages = append(systemMessages, content)
		case "user":
			lastUserMessage = content
			if idx != len(messages)-1 {
				historyMessages = append(historyMessages, "user: "+content)
			}
		case "assistant", "tool":
			historyMessages = append(historyMessages, message.Role+": "+content)
		default:
			historyMessages = append(historyMessages, message.Role+": "+content)
		}
	}

	if lastUserMessage == "" {
		return "", fmt.Errorf("no user message")
	}

	var builder strings.Builder
	if len(systemMessages) > 0 {
		builder.WriteString("System Instructions:\n")
		builder.WriteString(strings.Join(systemMessages, "\n"))
		builder.WriteString("\n\n")
	}

	if len(historyMessages) > 0 {
		builder.WriteString("Conversation History:\n")
		builder.WriteString(strings.Join(historyMessages, "\n"))
		builder.WriteString("\n\n")
	}

	builder.WriteString("Current User Question:\n")
	builder.WriteString(lastUserMessage)
	return builder.String(), nil
}

func (h *ShareChatHandler) handleOpenAIStreamResponse(
	c echo.Context,
	eventCh <-chan domain.SSEEvent,
	auditMeta openAIAuditMeta,
	streamOptions *domain.OpenAIStreamOptions,
	startedAt time.Time,
) error {
	responseID := "chatcmpl-" + generateID()
	created := time.Now().Unix()
	var usage *domain.OpenAIUsage

	for event := range eventCh {
		switch event.Type {
		case "error":
			return h.sendOpenAIErrorWithAudit(c, auditMeta, event.Content, "internal_error", startedAt)
		case "data":
			// send stream response
			streamResp := domain.OpenAIStreamResponse{
				ID:      responseID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   auditMeta.Model,
				Choices: []domain.OpenAIStreamChoice{
					{
						Index: 0,
						Delta: domain.OpenAIMessage{
							Role:    "assistant",
							Content: domain.NewStringContent(event.Content),
						},
					},
				},
			}

			if err := h.writeOpenAIStreamEvent(c, streamResp); err != nil {
				h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", err.Error(), startedAt)
				return err
			}
		case "usage":
			if event.Usage != nil {
				usage = &domain.OpenAIUsage{
					PromptTokens:     event.Usage.PromptTokens,
					CompletionTokens: event.Usage.CompletionTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
			}
		case "done":
			// send done event
			streamResp := domain.OpenAIStreamResponse{
				ID:      responseID,
				Object:  "chat.completion.chunk",
				Created: created,
				Model:   auditMeta.Model,
				Choices: []domain.OpenAIStreamChoice{
					{
						Index:        0,
						Delta:        domain.OpenAIMessage{},
						FinishReason: stringPtr("stop"),
					},
				},
			}
			if err := h.writeOpenAIStreamEvent(c, streamResp); err != nil {
				h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", err.Error(), startedAt)
				return err
			}
			if streamOptions != nil && streamOptions.IncludeUsage && usage != nil {
				usageResp := domain.OpenAIStreamResponse{
					ID:      responseID,
					Object:  "chat.completion.chunk",
					Created: created,
					Model:   auditMeta.Model,
					Choices: []domain.OpenAIStreamChoice{},
					Usage:   usage,
				}
				if err := h.writeOpenAIStreamEvent(c, usageResp); err != nil {
					h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", err.Error(), startedAt)
					return err
				}
			}
			if err := h.writeOpenAIDoneEvent(c); err != nil {
				h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", err.Error(), startedAt)
				return err
			}
			h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusOK, usage, "", "", startedAt)
			return nil
		}
	}
	h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", "stream ended without done event", startedAt)
	return nil
}

func (h *ShareChatHandler) handleOpenAINonStreamResponse(
	c echo.Context,
	eventCh <-chan domain.SSEEvent,
	auditMeta openAIAuditMeta,
	startedAt time.Time,
) error {
	responseID := "chatcmpl-" + generateID()
	created := time.Now().Unix()

	var content string
	var usage *domain.OpenAIUsage
	for event := range eventCh {
		switch event.Type {
		case "error":
			return h.sendOpenAIErrorWithAudit(c, auditMeta, event.Content, "internal_error", startedAt)
		case "data":
			content += event.Content
		case "usage":
			if event.Usage != nil {
				usage = &domain.OpenAIUsage{
					PromptTokens:     event.Usage.PromptTokens,
					CompletionTokens: event.Usage.CompletionTokens,
					TotalTokens:      event.Usage.TotalTokens,
				}
			}
		case "done":
			// send complete response
			resp := domain.OpenAICompletionsResponse{
				ID:      responseID,
				Object:  "chat.completion",
				Created: created,
				Model:   auditMeta.Model,
				Choices: []domain.OpenAIChoice{
					{
						Index: 0,
						Message: domain.OpenAIMessage{
							Role:    "assistant",
							Content: domain.NewStringContent(content),
						},
						FinishReason: "stop",
					},
				},
				Usage: usage,
			}
			if err := c.JSON(http.StatusOK, resp); err != nil {
				h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", err.Error(), startedAt)
				return err
			}
			h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusOK, usage, "", "", startedAt)
			return nil
		}
	}
	h.recordOpenAIAudit(c.Request().Context(), auditMeta, http.StatusInternalServerError, usage, "internal_error", "stream ended without done event", startedAt)
	return nil
}

func (h *ShareChatHandler) sendOpenAIError(c echo.Context, message, errorType string) error {
	errResp := domain.OpenAIErrorResponse{
		Error: domain.OpenAIError{
			Message: message,
			Type:    errorType,
		},
	}
	return c.JSON(openAIErrorStatusCode(errorType), errResp)
}

func (h *ShareChatHandler) writeOpenAIStreamEvent(c echo.Context, data domain.OpenAIStreamResponse) error {
	jsonContent, err := json.Marshal(data)
	if err != nil {
		return err
	}

	sseMessage := fmt.Sprintf("data: %s\n\n", string(jsonContent))
	if _, err := c.Response().Write([]byte(sseMessage)); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

func (h *ShareChatHandler) writeOpenAIDoneEvent(c echo.Context) error {
	if _, err := c.Response().Write([]byte("data: [DONE]\n\n")); err != nil {
		return err
	}
	c.Response().Flush()
	return nil
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func stringPtr(s string) *string {
	return &s
}

type openAIAuditMeta struct {
	KBID       string
	APITokenID *string
	Endpoint   string
	Model      string
	RemoteIP   string
	RequestID  *string
}

func (h *ShareChatHandler) sendOpenAIErrorWithAudit(
	c echo.Context,
	auditMeta openAIAuditMeta,
	message, errorType string,
	startedAt time.Time,
) error {
	h.recordOpenAIAudit(
		c.Request().Context(),
		auditMeta,
		openAIErrorStatusCode(errorType),
		nil,
		errorType,
		message,
		startedAt,
	)
	return h.sendOpenAIError(c, message, errorType)
}

func (h *ShareChatHandler) checkAPITokenGovernance(ctx context.Context, apiToken *domain.APIToken, now time.Time) (string, error) {
	if apiToken == nil || h.apiCallAuditRepo == nil {
		return "", nil
	}

	var (
		minuteCount int64
		dailyCount  int64
	)

	if apiToken.RateLimitPerMinute > 0 {
		count, err := h.apiCallAuditRepo.CountByTokenSince(ctx, apiToken.ID, openAICompletionsEndpoint, now.Add(-time.Minute))
		if err != nil {
			return "", err
		}
		minuteCount = count
	}

	if apiToken.DailyQuota > 0 {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		count, err := h.apiCallAuditRepo.CountByTokenSince(ctx, apiToken.ID, openAICompletionsEndpoint, dayStart)
		if err != nil {
			return "", err
		}
		dailyCount = count
	}

	return evaluateAPITokenGovernanceViolation(apiToken, minuteCount, dailyCount), nil
}

func evaluateAPITokenGovernanceViolation(apiToken *domain.APIToken, minuteCount, dailyCount int64) string {
	if apiToken == nil {
		return ""
	}
	if apiToken.RateLimitPerMinute > 0 && minuteCount >= int64(apiToken.RateLimitPerMinute) {
		return "rate_limit_error"
	}
	if apiToken.DailyQuota > 0 && dailyCount >= int64(apiToken.DailyQuota) {
		return "insufficient_quota"
	}
	return ""
}

func openAIGovernanceErrorMessage(errorType string) string {
	switch errorType {
	case "rate_limit_error":
		return "Rate limit exceeded for this API token"
	case "insufficient_quota":
		return "Daily quota exceeded for this API token"
	default:
		return "API token governance check failed"
	}
}

func (h *ShareChatHandler) recordOpenAIAudit(
	ctx context.Context,
	auditMeta openAIAuditMeta,
	statusCode int,
	usage *domain.OpenAIUsage,
	errorType, errorMessage string,
	startedAt time.Time,
) {
	if h.apiCallAuditRepo == nil {
		return
	}

	latencyMS := time.Since(startedAt).Milliseconds()
	if latencyMS < 0 {
		latencyMS = 0
	}

	audit := &domain.APICallAudit{
		KBID:         auditMeta.KBID,
		APITokenID:   auditMeta.APITokenID,
		Endpoint:     auditMeta.Endpoint,
		Model:        auditMeta.Model,
		StatusCode:   statusCode,
		ErrorType:    errorType,
		ErrorMessage: errorMessage,
		LatencyMS:    latencyMS,
		RemoteIP:     auditMeta.RemoteIP,
		RequestID:    auditMeta.RequestID,
	}
	if usage != nil {
		audit.PromptTokens = usage.PromptTokens
		audit.CompletionTokens = usage.CompletionTokens
		audit.TotalTokens = usage.TotalTokens
	}
	if err := h.apiCallAuditRepo.Create(ctx, audit); err != nil {
		h.logger.Warn("create api call audit failed",
			log.Error(err),
			log.String("kb_id", auditMeta.KBID),
			log.String("endpoint", auditMeta.Endpoint),
			log.Int("status_code", statusCode))
	}
}

func (h *ShareChatHandler) resolveAPIToken(ctx context.Context, token, kbID string) *domain.APIToken {
	if h.apiTokenRepo == nil || token == "" {
		return nil
	}
	apiToken, err := h.apiTokenRepo.GetByTokenWithCache(ctx, token)
	if err != nil {
		h.logger.Warn("get api token for audit failed", log.Error(err))
		return nil
	}
	if apiToken == nil || apiToken.ID == "" {
		return nil
	}
	if kbID != "" && apiToken.KbId != "" && apiToken.KbId != kbID {
		return nil
	}
	return apiToken
}

func (h *ShareChatHandler) resolveRequestID(c echo.Context) *string {
	candidates := []string{
		strings.TrimSpace(c.Request().Header.Get(echo.HeaderXRequestID)),
		strings.TrimSpace(c.Request().Header.Get("X-Request-ID")),
		strings.TrimSpace(c.Response().Header().Get(echo.HeaderXRequestID)),
	}
	for _, candidate := range candidates {
		if candidate != "" {
			requestID := candidate
			return &requestID
		}
	}
	return nil
}

func openAIErrorStatusCode(errorType string) int {
	switch errorType {
	case "unauthorized":
		return http.StatusUnauthorized
	case "forbidden":
		return http.StatusForbidden
	case "rate_limit_error", "insufficient_quota":
		return http.StatusTooManyRequests
	case "internal_error":
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

// ChatSearch searches chat messages in shared knowledge base
//
//	@Summary		ChatSearch
//	@Description	ChatSearch
//	@Tags			share_chat_search
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.ChatSearchReq	true	"request"
//	@Success		200		{object}	domain.Response{data=domain.ChatSearchResp}
//	@Router			/share/v1/chat/search [post]
func (h *ShareChatHandler) ChatSearch(c echo.Context) error {
	var req domain.ChatSearchReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "parse request failed", err)
	}
	req.KBID = c.Request().Header.Get("X-KB-ID") // get from caddy header
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}
	ctx := c.Request().Context()
	// validate captcha token
	if !h.Captcha.ValidateToken(ctx, req.CaptchaToken) {
		return h.NewResponseWithError(c, "invalid captcha token", nil)
	}

	req.RemoteIP = c.RealIP()

	// get user info --> no enterprise is nil
	userID := c.Get("user_id")
	if userID != nil {
		if userIDValue, ok := userID.(uint); ok {
			req.AuthUserID = userIDValue
		} else {
			return h.NewResponseWithError(c, "invalid user id type", nil)
		}
	}

	resp, err := h.chatUsecase.Search(ctx, &req)
	if err != nil {
		return h.NewResponseWithError(c, "failed to search docs", err)
	}
	return h.NewResponseWithData(c, resp)
}

// WidgetSearch
//
//	@Summary		WidgetSearch
//	@Description	WidgetSearch
//	@Tags			Widget
//	@Accept			json
//	@Produce		json
//	@Param			request	body		domain.ChatSearchReq	true	"Comment"
//	@Success		200		{object}	domain.Response{data=domain.ChatSearchResp}
//	@Router			/share/v1/chat/widget/search [post]
func (h *ShareChatHandler) WidgetSearch(c echo.Context) error {
	var req domain.ChatSearchReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "parse request failed", err)
	}
	req.KBID = c.Request().Header.Get("X-KB-ID")
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}
	ctx := c.Request().Context()

	// validate widget info
	widgetAppInfo, err := h.appUsecase.GetWidgetAppInfo(c.Request().Context(), req.KBID)
	if err != nil {
		h.logger.Error("get widget app info failed", log.Error(err))
		return h.sendErrMsg(c, "get app info error")
	}
	if !widgetAppInfo.Settings.WidgetBotSettings.IsOpen {
		return h.sendErrMsg(c, "widget is not open")
	}

	req.RemoteIP = c.RealIP()

	resp, err := h.chatUsecase.Search(ctx, &req)
	if err != nil {
		return h.NewResponseWithError(c, "failed to search docs", err)
	}
	return h.NewResponseWithData(c, resp)
}
