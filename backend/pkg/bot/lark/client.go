package lark

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
)

// LarkBotLogger implements Lark SDK logger interface
type LarkBotLogger struct {
	logger *log.Logger
}

func (l *LarkBotLogger) Info(ctx context.Context, args ...interface{}) {
	l.logger.Info("lark bot", log.Any("args", args))
}

func (l *LarkBotLogger) Error(ctx context.Context, args ...interface{}) {
	l.logger.Error("lark bot", log.Any("args", args))
}

func (l *LarkBotLogger) Debug(ctx context.Context, args ...interface{}) {
	l.logger.Debug("lark bot", log.Any("args", args))
}

func (l *LarkBotLogger) Warn(ctx context.Context, args ...interface{}) {
	l.logger.Warn("lark bot", log.Any("args", args))
}

// LarkClient is a Lark bot client using larksuite SDK (configured for Lark international endpoints)
// Note: Lark uses HTTP callbacks instead of WebSocket for event handling
type LarkClient struct {
	ctx          context.Context
	cancel       context.CancelFunc
	done         chan struct{}
	clientID     string
	clientSecret string
	logger       *log.Logger
	client       *lark.Client
	msgMap       sync.Map
	getQA        bot.GetQAFun
	eventHandler *dispatcher.EventDispatcher
	verifyToken  string
	encryptKey   string
}

// NewLarkClient creates a new Lark bot client
// Lark is the international version of Feishu, using different API endpoints
// Unlike Feishu (China), Lark (International) uses HTTP callbacks instead of WebSocket
func NewLarkClient(ctx context.Context, cancel context.CancelFunc, clientID, clientSecret, verifyToken, encryptKey string, logger *log.Logger, getQA bot.GetQAFun) (*LarkClient, error) {
	// Create client with Lark (international) domain
	client := lark.NewClient(clientID, clientSecret,
		lark.WithLogger(&LarkBotLogger{logger: logger}),
		lark.WithOpenBaseUrl("https://open.larksuite.com"), // Lark international endpoint
	)

	c := &LarkClient{
		ctx:          ctx,
		cancel:       cancel,
		done:         make(chan struct{}),
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       client,
		logger:       logger,
		getQA:        getQA,
		verifyToken:  verifyToken,
		encryptKey:   encryptKey,
	}

	// Setup event handler for HTTP callbacks
	c.setupEventHandler()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.msgMap.Range(func(key, value any) bool {
					// remove messageId if it is older than 5 minutes
					if time.Now().Unix()-value.(int64) > 5*60 {
						c.msgMap.Delete(key)
					}
					return true
				})
			}
		}
	}()
	return c, nil
}

// setupEventHandler configures the event dispatcher for handling HTTP callbacks
func (c *LarkClient) setupEventHandler() {
	c.eventHandler = dispatcher.NewEventDispatcher(c.verifyToken, c.encryptKey).
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			if *event.Event.Message.MessageId == "" {
				return nil
			}
			messageId := *event.Event.Message.MessageId
			if _, ok := c.msgMap.Load(messageId); ok {
				return nil
			}
			c.msgMap.Store(messageId, time.Now().Unix())
			c.logger.Info("received message from lark bot", log.String("message_id", messageId))
			if *event.Event.Message.MessageType != "text" {
				return nil
			}
			switch *event.Event.Message.ChatType {
			case "group":
				var message Message
				if err := json.Unmarshal([]byte(*event.Event.Message.Content), &message); err != nil {
					c.logger.Error("failed to unmarshal message", log.Error(err))
					return nil
				}
				// Replace mention placeholders with actual user names
				questionText := c.replaceMentions(message.Text, event.Event.Message.Mentions)
				go c.sendQACard(c.ctx, "chat_id", *event.Event.Message.ChatId, questionText, *event.Event.Sender.SenderId.OpenId)
			case "p2p":
				var message Message
				if err := json.Unmarshal([]byte(*event.Event.Message.Content), &message); err != nil {
					c.logger.Error("failed to unmarshal message", log.Error(err))
					return nil
				}
				go c.sendQACard(c.ctx, "open_id", *event.Event.Sender.SenderId.OpenId, message.Text, *event.Event.Message.ChatId)
			default:
				c.logger.Warn("unsupported chat type", log.String("chat_type", *event.Event.Message.ChatType))
			}
			return nil
		})
}

// GetEventHandler returns the event dispatcher for HTTP callback handling
// This should be registered with the HTTP server to handle Lark callbacks
func (c *LarkClient) GetEventHandler() *dispatcher.EventDispatcher {
	return c.eventHandler
}

var cardDataTemplate = `{"schema":"2.0","header":{"title":{"content":"%s","tag":"plain_text"}},"config":{"streaming_mode":true,"summary":{"content":""}},"body":{"elements":[{"tag":"markdown","content":"%s","element_id":"markdown_1"}]}}`

func (c *LarkClient) sendQACard(ctx context.Context, receiveIdType string, receiveId string, question string, additionalInfo string) {
	// create card
	cardData := fmt.Sprintf(cardDataTemplate, question, "稍等，让我想一想...")
	req := larkcardkit.NewCreateCardReqBuilder().
		Body(larkcardkit.NewCreateCardReqBodyBuilder().
			Type(`card_json`).
			Data(cardData).
			Build()).
		Build()
	resp, err := c.client.Cardkit.V1.Card.Create(ctx, req)
	if err != nil {
		c.logger.Error("failed to create card", log.Error(err))
		return
	}
	if !resp.Success() {
		c.logger.Error("failed to create card", log.String("request_id", resp.RequestId()), log.Any("code_error", resp.CodeError))
		return
	}
	content, err := json.Marshal(map[string]any{
		"type": "card",
		"data": map[string]string{
			"card_id": *resp.Data.CardId,
		},
	})
	if err != nil {
		c.logger.Error("failed to marshal alarm card", log.Error(err))
		return
	}
	// send card to user or group
	res, err := c.client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIdType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType("interactive").
			ReceiveId(receiveId).
			Content(string(content)).
			Build()).
		Build())
	if err != nil {
		c.logger.Error("failed to create message", log.Error(err))
		return
	}
	if !res.Success() {
		c.logger.Error("failed to create message", log.Int("code", res.Code), log.String("msg", res.Msg), log.String("request_id", res.RequestId()))
		return
	}
	c.logger.Info("send QA card to user or group", log.String("receive_id_type", receiveIdType), log.String("receive_id", receiveId), log.String("question", question), log.String("additional_info", additionalInfo))

	// start processing QA
	convInfo := domain.ConversationInfo{
		UserInfo: domain.UserInfo{
			From: domain.MessageFromPrivate,
		},
	}
	if receiveIdType == "open_id" {
		userinfo, err := c.GetUserInfo(receiveId)
		if err != nil {
			c.logger.Error("get user info failed", log.Error(err))
		} else {
			if userinfo.UserId != nil {
				convInfo.UserInfo.UserID = *userinfo.UserId
			}
			if userinfo.Name != nil {
				convInfo.UserInfo.NickName = *userinfo.Name
			}
			if userinfo.Avatar != nil && userinfo.Avatar.AvatarOrigin != nil {
				convInfo.UserInfo.Avatar = *userinfo.Avatar.AvatarOrigin
			}
			c.logger.Info("get user info success", log.Any("user_info", userinfo))
		}
		convInfo.UserInfo.From = domain.MessageFromPrivate
	} else {
		userinfo, err := c.GetUserInfo(additionalInfo)
		if err != nil {
			c.logger.Error("get chat info failed", log.Error(err))
		} else {
			if userinfo.UserId != nil {
				convInfo.UserInfo.UserID = *userinfo.UserId
			}
			if userinfo.Name != nil {
				convInfo.UserInfo.NickName = *userinfo.Name
			}
			if userinfo.Avatar != nil && userinfo.Avatar.AvatarOrigin != nil {
				convInfo.UserInfo.Avatar = *userinfo.Avatar.AvatarOrigin
			}
			c.logger.Info("get chat user info success", log.Any("user_info", userinfo))
		}
		convInfo.UserInfo.From = domain.MessageFromGroup
	}

	answerCh, err := c.getQA(ctx, question, convInfo, "")
	if err != nil {
		c.logger.Error("lark client failed to get answer", log.Error(err))
		return
	}

	var buf strings.Builder
	seq := 0
	imageRegex := regexp.MustCompile(`!\[[^\]]*\]\([^)]+\)`)
	sendUpdate := func() error {
		seq++
		answer := imageRegex.ReplaceAllString(buf.String(), "")
		updateReq := larkcardkit.NewContentCardElementReqBuilder().
			CardId(*resp.Data.CardId).
			ElementId(`markdown_1`).
			Body(larkcardkit.NewContentCardElementReqBodyBuilder().
				Uuid(uuid.New().String()).
				Content(answer).
				Sequence(seq).
				Build()).
			Build()
		updateResp, err := c.client.Cardkit.V1.CardElement.Content(ctx, updateReq)
		if err != nil {
			c.logger.Error("failed to update card", log.Error(err))
			return err
		}
		if !updateResp.Success() {
			c.logger.Error("failed to update card", log.String("request_id", updateResp.RequestId()), log.Any("code_error", updateResp.CodeError))
			return fmt.Errorf("update card failed: %v", updateResp.CodeError)
		}
		return nil
	}

	for chunk := range answerCh {
		buf.WriteString(chunk)
		// drain all currently available chunks
		for len(answerCh) > 0 {
			buf.WriteString(<-answerCh)
		}
		if err := sendUpdate(); err != nil {
			c.logger.Error("lark client failed to send QA update", log.Error(err), log.Int("sequence", seq))
			return
		}
	}
	c.logger.Info("start processing QA", log.String("message_id", *res.Data.MessageId))
}

type Message struct {
	Text string `json:"text"`
}

// replaceMentions replaces mention placeholders like @_user_1 with actual user names
func (c *LarkClient) replaceMentions(text string, mentions []*larkim.MentionEvent) string {
	if len(mentions) == 0 {
		return text
	}

	result := text
	for _, mention := range mentions {
		if mention.Key != nil && mention.Name != nil {
			// Replace @_user_1, @_user_2, etc. with @ActualUserName
			result = strings.ReplaceAll(result, *mention.Key, "@"+*mention.Name)
		}
	}
	return result
}

// Start initializes the Lark bot client
// Note: Unlike Feishu, Lark doesn't use WebSocket. Events are handled via HTTP callbacks.
// The actual HTTP endpoint needs to be registered separately in the HTTP router.
func (c *LarkClient) Start() error {
	defer close(c.done)
	c.logger.Info("lark bot client initialized (HTTP callback mode)",
		log.String("app_id", c.clientID),
		log.String("note", "Register HTTP callback endpoint to receive events"))

	// For Lark, we don't start a WebSocket connection
	// Events will be received via HTTP callbacks handled by GetEventHandler()
	// Just keep the context alive
	<-c.ctx.Done()
	c.logger.Info("lark bot client stopped")
	return nil
}

func (c *LarkClient) GetUserInfo(UserOpenId string) (*larkcontact.User, error) {
	req := larkcontact.NewGetUserReqBuilder().UserId(UserOpenId).
		UserIdType(`open_id`).DepartmentIdType(`open_department_id`).Build()
	resp, err := c.client.Contact.User.Get(context.Background(), req)
	if err != nil {
		c.logger.Error("failed to get user info", log.Error(err))
		return nil, err
	}

	if !resp.Success() {
		c.logger.Error("failed to get user info, response status not success", log.Any("errcode:", resp.Code))
		return nil, fmt.Errorf("failed to get user info, response data not success")
	}

	return resp.Data.User, nil
}

func (c *LarkClient) Stop() {
	c.cancel()
}

// Done returns a channel that is closed when Start() has fully returned.
func (c *LarkClient) Done() <-chan struct{} {
	return c.done
}
