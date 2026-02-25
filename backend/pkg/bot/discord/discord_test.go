package discord

import (
	"context"
	"testing"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/stretchr/testify/require"
)

func TestNewDiscordClient(t *testing.T) {
	cfg, _ := config.NewConfig()
	log := log.NewLogger(cfg)
	token := "token"
	getQA := func(ctx context.Context, msg string, info domain.ConversationInfo, ConversationID string) (chan string, error) {
		contentCh := make(chan string, 10)
		go func() {
			defer close(contentCh)
			contentCh <- "hello " + msg
		}()
		return contentCh, nil
	}
	c, err := NewDiscordClient(log, token, getQA)
	require.NoError(t, err)
	require.NotNil(t, c)
}
