package http

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/chaitin/panda-wiki/config"
)

const staticFilePrefix = "/static-file/"

// RegisterStaticFileRoutes proxies or redirects /static-file/* to configured object storage.
func RegisterStaticFileRoutes(e *echo.Echo, cfg *config.Config) {
	target, err := cfg.S3.StaticFileProxyTarget()
	if err != nil || target == nil {
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	if target.Scheme == "https" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		proxy.Transport = transport
	}
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		if strings.HasPrefix(req.URL.Path, staticFilePrefix) {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, staticFilePrefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			} else if req.URL.Path[0] != '/' {
				req.URL.Path = "/" + req.URL.Path
			}
		}
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Request != nil && !nonPDFStaticFile.MatchString(resp.Request.URL.Path) {
			resp.Header.Set("Content-Disposition", "attachment")
		}
		return nil
	}

	handler := echo.WrapHandler(proxy)
	e.GET("/static-file/*", handler)
	e.HEAD("/static-file/*", handler)
}
