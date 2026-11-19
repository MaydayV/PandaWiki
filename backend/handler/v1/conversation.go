package v1

import (
	"github.com/labstack/echo/v4"

	v1 "github.com/chaitin/panda-wiki/api/conversation/v1"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/usecase"
)

type ConversationHandler struct {
	*handler.BaseHandler
	logger  *log.Logger
	auth    middleware.AuthMiddleware
	usecase *usecase.ConversationUsecase
}

func NewConversationHandler(echo *echo.Echo, baseHandler *handler.BaseHandler, logger *log.Logger, auth middleware.AuthMiddleware, usecase *usecase.ConversationUsecase) *ConversationHandler {
	handler := &ConversationHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler_conversation"),
		auth:        auth,
		usecase:     usecase,
	}
	group := echo.Group("/api/v1/conversation", handler.auth.Authorize, handler.auth.ValidateKBUserPerm(consts.UserKBPermissionDataOperate))
	group.GET("", handler.GetConversationList)
	group.GET("/detail", handler.GetConversationDetail)
	group.GET("/message/list", handler.GetMessageFeedBackList)
	group.GET("/message/detail", handler.GetMessageDetail)

	return handler
}

type ConversationListItems = domain.PaginatedResult[[]domain.ConversationListItem]

// GetConversationList
//
//	@Summary		get conversation list
//	@Description	get conversation list
//	@Tags			conversation
//	@Accept			json
//	@Produce		json
//	@Param			req	query		domain.ConversationListReq	true	"conversation list request"
//	@Success		200	{object}	domain.PWResponse{data=ConversationListItems}
//	@Router			/api/v1/conversation [get]
func (h *ConversationHandler) GetConversationList(c echo.Context) error {
	var request domain.ConversationListReq
	if err := c.Bind(&request); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}

	ctx := c.Request().Context()

	conversationList, err := h.usecase.GetConversationList(ctx, &request)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get conversation list", err)
	}

	return h.NewResponseWithData(c, conversationList)
}

// GetConversationDetail
//
//	@Summary		get conversation detail
//	@Description	get conversation detail
//	@Tags			conversation
//	@Accept			json
//	@Produce		json
//	@Param			param	query		v1.GetConversationDetailReq	true	"conversation id"
//	@Success		200		{object}	domain.PWResponse{data=domain.ConversationDetailResp}
//	@Router			/api/v1/conversation/detail [get]
func (h *ConversationHandler) GetConversationDetail(c echo.Context) error {

	var req v1.GetConversationDetailReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	conversation, err := h.usecase.GetConversationDetail(c.Request().Context(), req.KbId, req.ID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get conversation detail", err)
	}

	return h.NewResponseWithData(c, conversation)
}

// GetMessageFeedBackList
//
//	@Summary		GetMessageFeedBackList
//	@Description	GetMessageFeedBackList
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			req	query		domain.MessageListReq																	true	"message list request"
//
//	@Success		200	{object}	domain.PWResponse{data=domain.PaginatedResult[[]domain.ConversationMessageListItem]}	"MessageList"
//	@Router			/api/v1/conversation/message/list [get]
func (h *ConversationHandler) GetMessageFeedBackList(c echo.Context) error {
	var request domain.MessageListReq
	if err := c.Bind(&request); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	h.logger.Debug("GetMessageFeedBackList request", log.Any("request", request))
	ctx := c.Request().Context()
	messages, err := h.usecase.GetMessageList(ctx, &request)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get message list", err)
	}
	return h.NewResponseWithData(c, messages)
}

// GetMessageDetail
//
//	@Summary		Get message detail
//	@Description	Get message detail
//	@Tags			Message
//	@Accept			json
//	@Produce		json
//	@Param			id	query		v1.GetMessageDetailReq	true	"message id"
//	@Success		200	{object}	domain.PWResponse{data=domain.ConversationMessage}
//	@Router			/api/v1/conversation/message/detail [get]
func (h *ConversationHandler) GetMessageDetail(c echo.Context) error {
	var req v1.GetMessageDetailReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request", err)
	}
	if err := c.Validate(&req); err != nil {
		return h.NewResponseWithError(c, "validate request failed", err)
	}

	message, err := h.usecase.GetMessageDetail(c.Request().Context(), req.KbId, req.ID)
	if err != nil {
		return h.NewResponseWithError(c, "failed to get message detail", err)
	}

	return h.NewResponseWithData(c, message)
}
