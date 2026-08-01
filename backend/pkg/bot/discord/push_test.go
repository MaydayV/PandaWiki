package discord

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	pwlog "github.com/chaitin/panda-wiki/log"
)

func TestDiscordSendTextMessage(t *testing.T) {
	var gotAuth string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		require.True(t, strings.HasSuffix(r.URL.Path, "/channels/123456/messages"))
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer server.Close()

	n := NewDiscordPushNotifier(&pwlog.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}, "test-token")
	n.httpClient = server.Client()
	n.apiBase = server.URL

	err := n.SendTextMessage(t.Context(), "123456", "hello discord")
	require.NoError(t, err)
	require.Equal(t, "Bot test-token", gotAuth)
	require.Equal(t, "hello discord", gotBody["content"])
}

func TestDiscordSendTextMessageTruncates(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer server.Close()

	n := NewDiscordPushNotifier(&pwlog.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}, "token")
	n.httpClient = server.Client()
	n.apiBase = server.URL

	long := strings.Repeat("a", discordContentLimit+50)
	err := n.SendTextMessage(t.Context(), "123", long)
	require.NoError(t, err)
	content, _ := gotBody["content"].(string)
	require.Len(t, content, discordContentLimit)
	require.True(t, strings.HasSuffix(content, "..."))
}
