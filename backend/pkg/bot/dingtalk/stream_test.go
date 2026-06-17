package dingtalk

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pwlog "github.com/chaitin/panda-wiki/log"
)

func newTestLogger() *pwlog.Logger {
	return &pwlog.Logger{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func newTestDingTalkClient(t *testing.T) *DingTalkClient {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	client, err := NewDingTalkClient(
		ctx,
		cancel,
		"client-id",
		"client-secret",
		"template-id",
		newTestLogger(),
		nil,
	)
	require.NoError(t, err)

	client.messageTTL = time.Minute

	return client
}

func TestTryMarkMessageDeduplicatesWithinTTL(t *testing.T) {
	client := newTestDingTalkClient(t)

	now := time.Now()
	client.nowFunc = func() time.Time {
		return now
	}

	require.True(t, client.tryMarkMessage("msg-1"))
	require.False(t, client.tryMarkMessage("msg-1"))

	client.markMessageCompleted("msg-1")
	require.False(t, client.tryMarkMessage("msg-1"))

	now = now.Add(client.messageTTL + time.Second)

	require.True(t, client.tryMarkMessage("msg-1"))
}

func TestOnChatBotMessageReceivedIgnoresDuplicateMsgID(t *testing.T) {
	client := newTestDingTalkClient(t)

	processed := make(chan struct{}, 2)
	client.processMessageFn = func(context.Context, *chatbot.BotCallbackDataModel) error {
		processed <- struct{}{}
		return nil
	}

	data := &chatbot.BotCallbackDataModel{
		MsgId: "msg-1",
		Text: chatbot.BotCallbackDataTextModel{
			Content: "hello",
		},
	}

	resp, err := client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	resp, err = client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("expected first message to be processed")
	}

	select {
	case <-processed:
		t.Fatal("expected duplicate message to be ignored")
	case <-time.After(300 * time.Millisecond):
	}
}

func TestOnChatBotMessageReceivedReturnsBeforeProcessingCompletes(t *testing.T) {
	client := newTestDingTalkClient(t)

	started := make(chan struct{})
	unblock := make(chan struct{})
	client.processMessageFn = func(context.Context, *chatbot.BotCallbackDataModel) error {
		close(started)
		<-unblock
		return nil
	}

	done := make(chan struct{})
	go func() {
		_, _ = client.OnChatBotMessageReceived(context.Background(), &chatbot.BotCallbackDataModel{
			MsgId: "msg-2",
			Text: chatbot.BotCallbackDataTextModel{
				Content: "slow question",
			},
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected callback to return before background processing finishes")
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("expected background processing to start")
	}

	close(unblock)
}

func TestOnChatBotMessageReceivedAllowsRetryAfterProcessingError(t *testing.T) {
	client := newTestDingTalkClient(t)

	attempts := make(chan struct{}, 4)
	client.processMessageFn = func(context.Context, *chatbot.BotCallbackDataModel) error {
		attempts <- struct{}{}
		return assert.AnError
	}

	data := &chatbot.BotCallbackDataModel{
		MsgId: "msg-retry",
		Text: chatbot.BotCallbackDataTextModel{
			Content: "retry please",
		},
	}

	resp, err := client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	select {
	case <-attempts:
	case <-time.After(time.Second):
		t.Fatal("expected first message to be processed")
	}

	require.Eventually(t, func() bool {
		_, callErr := client.OnChatBotMessageReceived(context.Background(), data)
		require.NoError(t, callErr)

		select {
		case <-attempts:
			return true
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
}

func TestOnChatBotMessageReceivedRecoversBackgroundPanic(t *testing.T) {
	client := newTestDingTalkClient(t)

	attempts := make(chan struct{}, 4)
	client.processMessageFn = func(context.Context, *chatbot.BotCallbackDataModel) error {
		attempts <- struct{}{}
		panic("boom")
	}

	data := &chatbot.BotCallbackDataModel{
		MsgId: "msg-panic",
		Text: chatbot.BotCallbackDataTextModel{
			Content: "panic please",
		},
	}

	resp, err := client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	select {
	case <-attempts:
	case <-time.After(time.Second):
		t.Fatal("expected background processing to start")
	}

	require.Eventually(t, func() bool {
		_, callErr := client.OnChatBotMessageReceived(context.Background(), data)
		require.NoError(t, callErr)

		select {
		case <-attempts:
			return true
		default:
			return false
		}
	}, time.Second, 20*time.Millisecond)
}

func TestOnChatBotMessageReceivedKeepsInFlightMessageMarkedPastTTL(t *testing.T) {
	client := newTestDingTalkClient(t)

	now := time.Now()
	client.nowFunc = func() time.Time {
		return now
	}

	processed := make(chan struct{}, 2)
	unblock := make(chan struct{})
	client.processMessageFn = func(context.Context, *chatbot.BotCallbackDataModel) error {
		processed <- struct{}{}
		<-unblock
		return nil
	}

	data := &chatbot.BotCallbackDataModel{
		MsgId: "msg-inflight",
		Text: chatbot.BotCallbackDataTextModel{
			Content: "long running question",
		},
	}

	resp, err := client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	select {
	case <-processed:
	case <-time.After(time.Second):
		t.Fatal("expected first message to be processed")
	}

	now = now.Add(client.messageTTL + time.Second)
	client.cleanupExpiredMessages()

	resp, err = client.OnChatBotMessageReceived(context.Background(), data)
	require.NoError(t, err)
	assert.Equal(t, []byte(""), resp)

	select {
	case <-processed:
		t.Fatal("expected in-flight duplicate message to be ignored after ttl cleanup")
	case <-time.After(300 * time.Millisecond):
	}

	close(unblock)
}
