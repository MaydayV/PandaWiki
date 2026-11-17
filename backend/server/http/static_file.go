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

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
			if strings.HasPrefix(r.Out.URL.Path, staticFilePrefix) {
				r.Out.URL.Path = strings.TrimPrefix(r.Out.URL.Path, staticFilePrefix)
				if r.Out.URL.Path == "" {
					r.Out.URL.Path = "/"
				} else if r.Out.URL.Path[0] != '/' {
					r.Out.URL.Path = "/" + r.Out.URL.Path
				}
			}
			r.SetXForwarded()
		},
	}
	if target.Scheme == "https" {
		proxy.Transport = http.DefaultTransport.(*http.Transport).Clone()
	}
	proxy.ModifyResponse = func(resp *http.Response) error {
		if resp.Request != nil {
			resp.Header.Set("Content-Disposition", "attachment")
			resp.Header.Set("X-Content-Type-Options", "nosniff")
		}
		return nil
	}

	handler := echo.WrapHandler(proxy)
	e.GET("/static-file/*", handler)
	e.HEAD("/static-file/*", handler)
}
