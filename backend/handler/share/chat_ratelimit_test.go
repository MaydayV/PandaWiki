package share

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/go-playground/validator"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/handler"
	"github.com/chaitin/panda-wiki/log"
	storecache "github.com/chaitin/panda-wiki/store/cache"
)

type mockRedisCounter struct {
	mu      sync.Mutex
	counts  map[string]int64
	ln      net.Listener
	closeCh chan struct{}
}

func startMockRedisCounter(t *testing.T) *mockRedisCounter {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	m := &mockRedisCounter{
		counts:  make(map[string]int64),
		ln:      ln,
		closeCh: make(chan struct{}),
	}
	go m.serve()
	t.Cleanup(func() {
		close(m.closeCh)
		_ = ln.Close()
	})
	return m
}

func (m *mockRedisCounter) serve() {
	for {
		conn, err := m.ln.Accept()
		if err != nil {
			return
		}
		go m.handleConn(conn)
	}
}

func (m *mockRedisCounter) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	inMulti := false
	var queued [][]string

	execCmd := func(args []string) string {
		if len(args) == 0 {
			return "-ERR empty\r\n"
		}
		switch strings.ToUpper(args[0]) {
		case "PING":
			return "+PONG\r\n"
		case "HELLO":
			return "-ERR unknown command 'HELLO'\r\n"
		case "CLIENT":
			return "+OK\r\n"
		case "INCR":
			if len(args) < 2 {
				return "-ERR args\r\n"
			}
			m.mu.Lock()
			m.counts[args[1]]++
			val := m.counts[args[1]]
			m.mu.Unlock()
			return fmt.Sprintf(":%d\r\n", val)
		case "EXPIRE":
			return ":1\r\n"
		default:
			return "+OK\r\n"
		}
	}

	for {
		args, err := readRESPArray(r)
		if err != nil {
			return
		}
		if len(args) == 0 {
			continue
		}
		cmd := strings.ToUpper(args[0])
		if cmd == "MULTI" {
			inMulti = true
			queued = nil
			_, _ = conn.Write([]byte("+OK\r\n"))
			continue
		}
		if cmd == "EXEC" {
			inMulti = false
			var buf strings.Builder
			fmt.Fprintf(&buf, "*%d\r\n", len(queued))
			for _, q := range queued {
				buf.WriteString(execCmd(q))
			}
			_, _ = conn.Write([]byte(buf.String()))
			continue
		}
		if inMulti {
			queued = append(queued, args)
			_, _ = conn.Write([]byte("+QUEUED\r\n"))
			continue
		}
		_, _ = conn.Write([]byte(execCmd(args)))
	}
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return strings.Fields(line), nil
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		hdr = strings.TrimRight(hdr, "\r\n")
		sz, err := strconv.Atoi(strings.TrimPrefix(hdr, "$"))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, sz+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:sz]))
	}
	return args, nil
}

type shareTestValidator struct {
	v *validator.Validate
}

func (sv *shareTestValidator) Validate(i any) error {
	return sv.v.Struct(i)
}

func TestWidgetRateLimitStopsRequestAndPreservesJSON(t *testing.T) {
	redisSrv := startMockRedisCounter(t)
	cfg := &config.Config{
		Log:   config.LogConfig{Level: 8},
		Redis: config.RedisConfig{Addr: redisSrv.ln.Addr().String()},
	}
	cacheClient, err := storecache.NewCache(cfg)
	require.NoError(t, err)
	defer cacheClient.Close()

	logger := log.NewLogger(cfg)
	e := echo.New()
	e.Validator = &shareTestValidator{v: validator.New()}
	base := handler.NewBaseHandler(e, logger, cfg, nil, nil, nil)
	h := NewShareChatHandler(e, base, logger, cacheClient, nil, nil, nil, nil, nil, nil, nil)

	// 前 widgetRateLimitPerMin 次应允许通过
	for i := 0; i < widgetRateLimitPerMin; i++ {
		require.NoError(t, h.enforceWidgetRateLimit(t.Context(), "10.0.0.1"))
	}

	// 第 widgetRateLimitPerMin+1 次必须返回 errWidgetRateLimited
	err = h.enforceWidgetRateLimit(t.Context(), "10.0.0.1")
	require.ErrorIs(t, err, errWidgetRateLimited)

	// WidgetSearch 超过上限时必须直接返回干净的 JSON 错误响应，
	// 既不继续执行搜索（appUsecase 为 nil 也不会 panic），也不混入 SSE 帧。
	// 默认/英文请求必须返回英文限频文案，中文请求必须返回中文限频文案。
	reqEN := httptest.NewRequest(http.MethodPost, "/share/v1/chat/widget/search", bytes.NewBufferString(`{"message":"hello"}`))
	reqEN.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	reqEN.Header.Set("X-KB-ID", "kb-1")
	reqEN.Header.Set("X-Lang", "en-US")
	reqEN.RemoteAddr = "10.0.0.1:12345"
	recEN := httptest.NewRecorder()
	e.ServeHTTP(recEN, reqEN)

	require.Equal(t, http.StatusOK, recEN.Code)
	require.NotContains(t, recEN.Body.String(), "data: ")
	var respEN domain.PWResponse
	require.NoError(t, json.Unmarshal(recEN.Body.Bytes(), &respEN))
	require.False(t, respEN.Success)
	require.Contains(t, respEN.Message, "Too many requests")

	reqZH := httptest.NewRequest(http.MethodPost, "/share/v1/chat/widget/search", bytes.NewBufferString(`{"message":"hello"}`))
	reqZH.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	reqZH.Header.Set("X-KB-ID", "kb-1")
	reqZH.Header.Set("X-Lang", "zh-CN")
	reqZH.RemoteAddr = "10.0.0.1:12345"
	recZH := httptest.NewRecorder()
	e.ServeHTTP(recZH, reqZH)

	require.Equal(t, http.StatusOK, recZH.Code)
	require.NotContains(t, recZH.Body.String(), "data: ")
	var respZH domain.PWResponse
	require.NoError(t, json.Unmarshal(recZH.Body.Bytes(), &respZH))
	require.False(t, respZH.Success)
	require.Contains(t, respZH.Message, "请求过于频繁")
}
