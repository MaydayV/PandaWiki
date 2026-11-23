package v1

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
)

type stubFileAuthMiddleware struct {
	denyKBPerm bool
}

func (s *stubFileAuthMiddleware) Authorize(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return next(c)
	}
}

func (s *stubFileAuthMiddleware) ValidateUserRole(_ consts.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}

func (s *stubFileAuthMiddleware) ValidateKBUserPerm(_ consts.UserKBPermission) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if s.denyKBPerm {
				return c.JSON(http.StatusForbidden, domain.PWResponse{
					Success: false,
					Message: "Forbidden",
				})
			}
			return next(c)
		}
	}
}

func (s *stubFileAuthMiddleware) ValidateLicenseEdition(_ ...consts.LicenseEdition) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return next
	}
}

func (s *stubFileAuthMiddleware) MustGetUserID(_ echo.Context) (string, bool) {
	return "user-1", true
}

type fileTestValidator struct {
	v *validator.Validate
}

func (fv *fileTestValidator) Validate(i any) error {
	return fv.v.Struct(i)
}

func newTestFileEcho(denyKBPerm bool) *echo.Echo {
	cfg := &config.Config{Log: config.LogConfig{Level: 8}}
	logger := log.NewLogger(cfg)
	e := echo.New()
	e.Validator = &fileTestValidator{v: validator.New()}
	auth := &stubFileAuthMiddleware{denyKBPerm: denyKBPerm}
	base := handler.NewBaseHandler(e, logger, cfg, auth, nil, nil)
	NewFileHandler(e, base, logger, auth, nil, nil)
	return e
}

func TestFileUploadBlocksUnauthorizedKB(t *testing.T) {
	e := newTestFileEcho(true)

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	require.NoError(t, mw.WriteField("kb_id", "kb-target"))
	part, err := mw.CreateFormFile("file", "note.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("hello"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/file/upload", &body)
	req.Header.Set(echo.HeaderContentType, mw.FormDataContentType())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// 必须被中间件截断在 403 且不会进入 Upload handler（若进入会因 nil fileUsecase panic）
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestFileUploadByUrlBlocksUnauthorizedAndPreservesBodyWhenAllowed(t *testing.T) {
	t.Run("forbidden kb_id stops before handler", func(t *testing.T) {
		e := newTestFileEcho(true)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/file/upload/url", bytes.NewBufferString(`{"kb_id":"kb-target","url":"https://example.com/a.png"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("allowed kb_id preserves body for UploadByUrl bind and validate", func(t *testing.T) {
		e := newTestFileEcho(false)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/file/upload/url", bytes.NewBufferString(`{"kb_id":"kb-target","url":"not-a-url"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp domain.PWResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.False(t, resp.Success)
		require.Contains(t, resp.Message, "validate request body failed")
	})
}
