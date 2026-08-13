package http

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/setup"
)

var nonPDFStaticFile = regexp.MustCompile(`(?i)\.pdf($|\?)`)

// StartAdminServer serves the admin SPA on HTTPS when static assets are present.
func StartAdminServer(cfg *config.Config, logger *log.Logger) error {
	if !cfg.Admin.Enabled {
		logger.Info("admin server disabled (ADMIN_ENABLED=0)")
		return nil
	}

	distDir := cfg.Admin.DistDir
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		logger.Info("admin static assets not found, admin server skipped", log.String("dir", distDir))
		return nil
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("1000M"))

	apiTarget, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", cfg.HTTP.Port))
	if err != nil {
		return fmt.Errorf("parse api target: %w", err)
	}

	apiProxy := newReverseProxy(apiTarget, false)
	streamProxy := newReverseProxy(apiTarget, true)

	e.GET("/503", func(c echo.Context) error {
		return c.NoContent(http.StatusServiceUnavailable)
	})

	streamPaths := []string{"/share/v1/chat/message", "/api/v1/creation/text"}
	for _, p := range streamPaths {
		path := p
		e.Any(path, echo.WrapHandler(streamProxy))
	}

	e.Any("/api/*", echo.WrapHandler(apiProxy))
	e.Any("/share/*", echo.WrapHandler(apiProxy))
	RegisterStaticFileRoutes(e, cfg)

	e.GET("/*", func(c echo.Context) error {
		reqPath := c.Request().URL.Path
		if reqPath == "/" {
			reqPath = "/index.html"
		}
		clean := path.Clean(reqPath)
		if strings.HasPrefix(clean, "../") {
			return c.NoContent(http.StatusBadRequest)
		}
		filePath := filepath.Join(distDir, strings.TrimPrefix(clean, "/"))
		if info, statErr := os.Stat(filePath); statErr == nil && !info.IsDir() {
			if strings.HasSuffix(strings.ToLower(filePath), ".html") {
				c.Response().Header().Set("Cache-Control", "no-cache")
			}
			return c.File(filePath)
		}
		c.Response().Header().Set("Cache-Control", "no-cache")
		return c.File(filepath.Join(distDir, "index.html"))
	})

	addr := fmt.Sprintf(":%d", cfg.Admin.Port)
	logger.Info("starting admin server", log.String("addr", addr), log.String("dist", distDir))
	return e.StartTLS(addr, setup.CertFile, setup.KeyFile)
}

func newReverseProxy(target *url.URL, streaming bool) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if streaming {
		transport.ResponseHeaderTimeout = 0
	}
	proxy.Transport = transport
	if streaming {
		proxy.FlushInterval = -1
	}
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
	}
	return proxy
}
