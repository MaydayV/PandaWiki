package wecom

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	pwlog "github.com/chaitin/panda-wiki/log"
)

func TestWecomSendTextMessage(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	n := NewWecomWebhookNotifier(&pwlog.Logger{Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	n.httpClient = server.Client()

	err := n.SendTextMessage(t.Context(), server.URL, "**hello**")
	require.NoError(t, err)
	require.Equal(t, "markdown", gotBody["msgtype"])
	md, ok := gotBody["markdown"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "**hello**", md["content"])
}
