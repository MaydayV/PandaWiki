package dingtalk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignDingTalkWebhook(t *testing.T) {
	got, err := SignDingTalkWebhook(1599360473, "test_secret")
	require.NoError(t, err)
	require.Equal(t, "ShYqxyPXUhH1FX+nHDYMnwYB1dIosS5k8uqgzXmyjUw=", got)
}

func TestAppendDingTalkWebhookSign(t *testing.T) {
	got, err := AppendDingTalkWebhookSign(
		"https://oapi.dingtalk.com/robot/send?access_token=abc",
		1599360473,
		"test_secret",
	)
	require.NoError(t, err)
	require.Contains(t, got, "access_token=abc")
	require.Contains(t, got, "timestamp=1599360473")
	require.Contains(t, got, "sign=ShYqxyPXUhH1FX%2BnHDYMnwYB1dIosS5k8uqgzXmyjUw%3D")
}

func TestSendTextMessageWithSecretSignsURL(t *testing.T) {
	var gotURL string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &gotBody))
		_, _ = w.Write([]byte(`{"errcode":0,"errmsg":"ok"}`))
	}))
	defer server.Close()

	fixed := time.Unix(1599360473, 0)
	n := NewDingTalkPushNotifier(newTestLogger())
	n.httpClient = server.Client()
	n.now = func() time.Time { return fixed }

	err := n.SendTextMessage(t.Context(), server.URL+"|test_secret", "hello")
	require.NoError(t, err)
	require.Equal(t, "markdown", gotBody["msgtype"])
	require.Contains(t, gotURL, "timestamp=1599360473")
	require.Contains(t, gotURL, "sign=")
}
