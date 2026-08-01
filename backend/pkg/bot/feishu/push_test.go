package feishu

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignFeishuWebhook(t *testing.T) {
	got := SignFeishuWebhook(1599360473, "test_secret")
	require.Equal(t, "FW06sV98dJlmB07TC2kBBUSpkGrDmWP+mE2IRa7SQhA=", got)
}

func TestSendTextMessageWithSecretAttachesSign(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer server.Close()

	fixed := time.Unix(1599360473, 0)
	n := NewFeishuWebhookNotifier()
	n.httpClient = server.Client()
	n.now = func() time.Time { return fixed }

	err := n.SendTextMessage(t.Context(), server.URL+"|test_secret", "hello")
	require.NoError(t, err)
	require.Equal(t, "text", gotBody["msg_type"])
	require.Equal(t, "1599360473", gotBody["timestamp"])
	require.Equal(t, "FW06sV98dJlmB07TC2kBBUSpkGrDmWP+mE2IRa7SQhA=", gotBody["sign"])
}

func TestSendTextMessageWithoutSecretOmitsSign(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer server.Close()

	n := NewFeishuWebhookNotifier()
	n.httpClient = server.Client()

	err := n.SendTextMessage(t.Context(), server.URL, "hello")
	require.NoError(t, err)
	require.Equal(t, "text", gotBody["msg_type"])
	_, hasTS := gotBody["timestamp"]
	_, hasSign := gotBody["sign"]
	require.False(t, hasTS)
	require.False(t, hasSign)
}
