package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	lark "github.com/larksuite/oapi-sdk-go/v3"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkcardkit "github.com/larksuite/oapi-sdk-go/v3/service/cardkit/v1"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"

	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/pkg/bot"
)

type FeishuBotLogger struct {
	logger *log.Logger
}

func (l *FeishuBotLogger) Info(ctx context.Context, args ...interface{}) {
	l.logger.Info("feishu bot", log.Any("args", args))
}

func (l *FeishuBotLogger) Error(ctx context.Context, args ...interface{}) {
	l.logger.Error("feishu bot", log.Any("args", args))
}

func (l *FeishuBotLogger) Debug(ctx context.Context, args ...interface{}) {
	l.logger.Debug("feishu bot", log.Any("args", args))
}

func (l *FeishuBotLogger) Warn(ctx context.Context, args ...interface{}) {
	l.logger.Warn("feishu bot", log.Any("args", args))
}

type FeishuClient struct {
	ctx          context.Context
	cancel       context.CancelFunc
	clientID     string
	clientSecret string
	logger       *log.Logger
	client       *lark.Client
	msgMap       sync.Map
	getQA        bot.GetQAFun
}

func NewFeishuClient(ctx context.Context, cancel context.CancelFunc, clientID, clientSecret string, logger *log.Logger, getQA bot.GetQAFun) *FeishuClient {
	client := lark.NewClient(clientID, clientSecret, lark.WithLogger(&FeishuBotLogger{logger: logger}))

	c := &FeishuClient{
		ctx:          ctx,
		cancel:       cancel,
		clientID:     clientID,
		clientSecret: clientSecret,
		client:       client,
		logger:       logger,
		getQA:        getQA,
	}
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
	return c
}

var cardDataTemplate = `{"schema":"2.0","header":{"title":{"content":"%s","tag":"plain_text"}},"config":{"streaming_mode":true,"summary":{"content":""}},"body":{"elements":[{"tag":"markdown","content":"%s","element_id":"markdown_1"}]}}`

func (c *FeishuClient) sendTextMessage(ctx context.Context, receiveIdType string, receiveId string, text string) {
	msgContent, _ := json.Marshal(map[string]string{"text": text})
	resp, err := c.client.Im.Message.Create(ctx, larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIdType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			MsgType("text").
			ReceiveId(receiveId).
			Content(string(msgContent)).
			Build()).
		Build())
	if err != nil {
		c.logger.Error("failed to send fallback text message", log.Error(err))
		return
	}
	if !resp.Success() {
		c.logger.Error("failed to send fallback text message", log.Int("code", resp.Code), log.String("msg", resp.Msg))
	}
}

func (c *FeishuClient) sendQACard(ctx context.Context, receiveIdType string, receiveId string, question string, additionalInfo string) {
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
	c.logger.Info("send QA card to user or group", log.String("receive_id_type", receiveIdType), log.String("receive_id", receiveId), log.String("question", question), log.String("additional_info(chat:user_openid/p2p:chat_id)", additionalInfo))

	// start processing QA
	convInfo := domain.ConversationInfo{
		UserInfo: domain.UserInfo{
			From: domain.MessageFromPrivate, // 默认是私聊
		},
	}
	var userOpenId string
	if receiveIdType == "open_id" {
		userOpenId = receiveId
		convInfo.UserInfo.From = domain.MessageFromPrivate
	} else {
		userOpenId = additionalInfo
		convInfo.UserInfo.From = domain.MessageFromGroup
	}

	userinfo, err := c.GetUserInfo(userOpenId)
	if err != nil {
		c.logger.Error("get user info failed", log.Error(err), log.String("open_id", userOpenId))
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
		if userinfo.Mobile != nil {
			convInfo.UserInfo.Mobile = *userinfo.Mobile
		}
		c.logger.Info("get user info success", log.Any("user_info", userinfo))
	}

	answerCh, err := c.getQA(ctx, question, convInfo, "")
	if err != nil {
		c.logger.Error("get QA failed", log.Error(err))
		c.sendTextMessage(ctx, receiveIdType, receiveId, "出错了，请稍后再试")
		return
	}

	answer := ""
	seq := 1
	for chunk := range answerCh {
		seq += 1
		answer += chunk
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		// update card content streaming
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
			c.logger.Error("failed to update card (will retry next chunk)", log.Error(err))
			continue
		}
		if !updateResp.Success() {
			c.logger.Error("failed to update card (will retry next chunk)", log.String("request_id", updateResp.RequestId()), log.Any("code_error", updateResp.CodeError))
			continue
		}
	}
	// finalize card with full content
	finalReq := larkcardkit.NewContentCardElementReqBuilder().
		CardId(*resp.Data.CardId).
		ElementId(`markdown_1`).
		Body(larkcardkit.NewContentCardElementReqBodyBuilder().
			Uuid(uuid.New().String()).
			Content(answer).
			Sequence(seq + 1).
			Build()).
		Build()
	finalResp, err := c.client.Cardkit.V1.CardElement.Content(ctx, finalReq)
	if err != nil {
		c.logger.Error("failed to finalize card", log.Error(err))
	} else if !finalResp.Success() {
		c.logger.Error("failed to finalize card", log.String("request_id", finalResp.RequestId()), log.Any("code_error", finalResp.CodeError))
	}
	c.logger.Info("QA completed", log.String("message_id", *res.Data.MessageId))
}

type Message struct {
	Text string `json:"text"`
}

// replaceMentions removes @_user_N placeholders from group chat messages.
func (c *FeishuClient) replaceMentions(text string, mentions []*larkim.MentionEvent) string {
	if len(mentions) == 0 {
		return text
	}
	result := text
	for _, mention := range mentions {
		if mention.Key != nil && mention.Name != nil {
			result = strings.ReplaceAll(result, *mention.Key, "@"+*mention.Name)
		}
	}
	return result
}

func (c *FeishuClient) Start() error {
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2MessageReceiveV1(func(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
			// ignore duplicate message
			if *event.Event.Message.MessageId == "" {
				return nil
			}
			messageId := *event.Event.Message.MessageId
			if _, ok := c.msgMap.Load(messageId); ok {
				return nil
			}
			c.msgMap.Store(messageId, time.Now().Unix())
			c.logger.Info("received message from feishu bot", log.String("message_id", messageId))
			// only handle text type
			if *event.Event.Message.MessageType != "text" {
				return nil
			}
			var message Message
			if err := json.Unmarshal([]byte(*event.Event.Message.Content), &message); err != nil {
				c.logger.Error("failed to unmarshal message", log.Error(err))
				return nil
			}
			questionText := c.replaceMentions(message.Text, event.Event.Message.Mentions)

			// async processing with panic recovery
			go func() {
				defer func() {
					if r := recover(); r != nil {
						c.logger.Error("process feishu message panicked", log.String("msg_id", messageId), log.Any("panic", r))
						c.msgMap.Delete(messageId)
					}
				}()
				switch *event.Event.Message.ChatType {
				case "group":
					c.sendQACard(ctx, "chat_id", *event.Event.Message.ChatId, questionText, *event.Event.Sender.SenderId.OpenId)
				case "p2p":
					c.sendQACard(ctx, "open_id", *event.Event.Sender.SenderId.OpenId, questionText, *event.Event.Message.ChatId)
				default:
					c.logger.Warn("unsupported chat type", log.String("chat_type", *event.Event.Message.ChatType))
				}
			}()
			return nil
		})

	cli := larkws.NewClient(c.clientID, c.clientSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogger(&FeishuBotLogger{logger: c.logger}),
	)
	// FIXME: goroutine leak in larkws.Start
	err := cli.Start(c.ctx)
	if err != nil {
		return fmt.Errorf("failed to start feishu client: %w", err)
	}
	return nil
}

// 下面功能都是需要开启飞书对应的权限才可以获取到用户信息 -- 应用权限(否则获取不到对话用户的信息)

// 飞书机器人获取用户信息，只是适用于单个用户
func (c *FeishuClient) GetUserInfo(UserOpenId string) (*larkcontact.User, error) {
	// 获取用户信息，根据用户的id
	req := larkcontact.NewGetUserReqBuilder().UserId(UserOpenId).
		UserIdType(`open_id`).DepartmentIdType(`open_department_id`).Build()
	// 发起请求，获取用户消息
	resp, err := c.client.Contact.User.Get(context.Background(), req)
	if err != nil {
		c.logger.Error("failed to get user info", log.Error(err))
		return nil, err
	}

	// 失败
	if !resp.Success() {
		c.logger.Error("failed to get user info, response status not success", log.Any("errcode:", resp.Code))
		return nil, fmt.Errorf("failed to get user info, response data not success")
	}

	return resp.Data.User, nil
}

func (c *FeishuClient) Stop() {
	c.cancel()
}
