package v1

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/middleware"
	"github.com/chaitin/panda-wiki/store/s3"
	"github.com/chaitin/panda-wiki/usecase"
)

type FileHandler struct {
	*handler.BaseHandler
	logger      *log.Logger
	auth        middleware.AuthMiddleware
	fileUsecase *usecase.FileUsecase
}

func NewFileHandler(echo *echo.Echo, baseHandler *handler.BaseHandler, logger *log.Logger, auth middleware.AuthMiddleware, minioClient *s3.MinioClient, fileUsecase *usecase.FileUsecase) *FileHandler {
	h := &FileHandler{
		BaseHandler: baseHandler,
		logger:      logger.WithModule("handler.v1.file"),
		auth:        auth,
		fileUsecase: fileUsecase,
	}
	group := echo.Group("/api/v1/file", h.auth.Authorize, h.requireKBPermIfPresent)
	group.POST("/upload", h.Upload)
	group.POST("/upload/url", h.UploadByUrl)
	return h
}

// requireKBPermIfPresent 对显式指定 kb_id 的上传校验 KB 文档管理权限。
// 未指定 kb_id 的上传写入随机 uuid 前缀（与任何 KB 无关），放行——
// 编辑器插图等场景依赖该路径，而显式 kb_id 若不校验则任何登录用户可向任意 KB 前缀写文件。
func (h *FileHandler) requireKBPermIfPresent(next echo.HandlerFunc) echo.HandlerFunc {
	check := h.auth.ValidateKBUserPerm(consts.UserKBPermissionDocManage)(next)
	return func(c echo.Context) error {
		kbID, err := middleware.GetKbID(c)
		if err != nil {
			return h.NewResponseWithError(c, "invalid request parameters", err)
		}
		if kbID == "" {
			return next(c)
		}
		return check(c)
	}
}

// Upload
//
//	@Summary		Upload File
//	@Description	Upload File
//	@Tags			file
//	@Accept			multipart/form-data
//	@Param			file	formData	file	true	"File"
//	@Param			kb_id	formData	string	false	"Knowledge Base ID"
//	@Success		200		{object}	domain.ObjectUploadResp
//	@Router			/api/v1/file/upload [post]
func (h *FileHandler) Upload(c echo.Context) error {
	cxt := c.Request().Context()
	kbID := c.FormValue("kb_id")
	if kbID == "" {
		kbID = uuid.New().String()
	}
	file, err := c.FormFile("file")
	if err != nil {
		return h.NewResponseWithError(c, "failed to get file", err)
	}
	key, err := h.fileUsecase.UploadFile(cxt, kbID, file)
	if err != nil {
		return h.NewResponseWithError(c, "upload failed", err)
	}

	return h.NewResponseWithData(c, domain.ObjectUploadResp{
		Key:      key,
		Filename: file.Filename,
	})
}

// UploadByUrl
//
//	@Summary		Upload File By Url
//	@Description	Upload File By Url
//	@Tags			file
//	@Accept			json
//	@Produce		json
//	@Param			body	body		domain.UploadByUrlReq	true	"Request Body"
//	@Success		200		{object}	domain.Response{data=domain.ObjectUploadResp}
//	@Router			/api/v1/file/upload/url [post]
func (h *FileHandler) UploadByUrl(c echo.Context) error {
	ctx := c.Request().Context()

	var req domain.UploadByUrlReq
	if err := c.Bind(&req); err != nil {
		return h.NewResponseWithError(c, "invalid request parameters", err)
	}

	if err := c.Validate(req); err != nil {
		return h.NewResponseWithError(c, "validate request body failed", err)
	}

	kbID := req.KbId
	if kbID == "" {
		kbID = uuid.New().String()
	}

	key, err := h.fileUsecase.UploadFileByUrl(ctx, kbID, req.Url)
	if err != nil {
		return h.NewResponseWithError(c, "upload failed", err)
	}

	return h.NewResponseWithData(c, domain.ObjectUploadResp{
		Key: key,
	})
}
