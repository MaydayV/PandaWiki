package wechat_service

import (
	"context"
	"sync"
	"time"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
)

type WechatServiceConfig struct {
	Ctx             context.Context
	CorpID          string
	Token           string
	EncodingAESKey  string
	kbID            string
	Secret          string
	logger          *log.Logger
	containKeywords []string
	equalKeywords   []string
	logoUrl         string
	// db
	WeRepo *pg.WechatRepository
}

// 存储ai知识库获取的cursor值以客服为标准，方便拉取用户的消息
var KfCursors = &sync.Map{}

// 微信客服发送的消息
type WeixinUserAskMsg struct {
	ToUserName string `xml:"ToUserName"`
	CreateTime int64  `xml:"CreateTime"`
	MsgType    string `xml:"MsgType"`
	Event      string `xml:"Event"`
	Token      string `xml:"Token"`
	OpenKfId   string `xml:"OpenKfId"`
}

type AccessToken struct {
	Errcode     int    `json:"errcode"`
	Errmsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type MsgRequest struct {
	Cursor      string `json:"cursor"`
	Token       string `json:"token"`
	Limit       int    `json:"limit"`
	VoiceFormat int    `json:"voice_format"`
	OpenKfid    string `json:"open_kfid"`
}

type MsgRet struct {
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
	NextCursor string `json:"next_cursor"` // 游标
	MsgList    []Msg  `json:"msg_list"`
	HasMore    int    `json:"has_more"`
}

type Msg struct {
	Msgid    string `json:"msgid"`
	SendTime int64  `json:"send_time"`
	Origin   int    `json:"origin"`
	Msgtype  string `json:"msgtype"`
	Event    struct {
		EventType      string `json:"event_type"`
		Scene          string `json:"scene"`
		OpenKfid       string `json:"open_kfid"`
		ExternalUserid string `json:"external_userid"`
		WelcomeCode    string `json:"welcome_code"`
	} `json:"event"`
	Text struct {
		Content string `json:"content"`
	} `json:"text"`
	OpenKfid       string `json:"open_kfid"`
	ExternalUserid string `json:"external_userid"`
}

// send msg to user with message
type ReplyMsg struct {
	Touser   string `json:"touser,omitempty"`
	OpenKfid string `json:"open_kfid,omitempty"`
	Msgid    string `json:"msgid,omitempty"`
	Msgtype  string `json:"msgtype,omitempty"`
	Text     struct {
		Content string `json:"content,omitempty"`
	} `json:"text,omitempty"`
}

// send msg to user with url
type ReplyMsgUrl struct {
	Touser   string `json:"touser,omitempty"`
	OpenKfid string `json:"open_kfid,omitempty"`
	Msgid    string `json:"msgid,omitempty"`
	Msgtype  string `json:"msgtype,omitempty"`
	Link     Link   `json:"link,omitempty"`
}

type Link struct {
	Title        string `json:"title,omitempty"`
	Desc         string `json:"desc,omitempty"`
	Url          string `json:"url,omitempty"`
	ThumbMediaID string `json:"thumb_media_id,omitempty"`
}

// Upload file response
type MediaUploadResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	MediaType string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt string `json:"created_at"`
}

// 获取用户消息应该得到的响应
type WechatCustomerResponse struct {
	ErrCode                int        `json:"errcode"`
	ErrMsg                 string     `json:"errmsg"`
	CustomerList           []Customer `json:"customer_list"`
	InvalidExternalUserIDs []string   `json:"invalid_external_userid"`
}

type Customer struct {
	ExternalUserID string `json:"external_userid"`
	Nickname       string `json:"nickname"`
	Avatar         string `json:"avatar"`
	Gender         int    `json:"gender"`
	UnionID        string `json:"unionid"`
}

type UerInfoRequest struct {
	UserID         []string `json:"external_userid_list"`
	SessionContext int      `json:"need_enter_session_context"`
}

// chat status
type Status struct {
	ErrCode       int    `json:"errcode"`
	ErrMsg        string `json:"errmsg"`
	ServiceState  int    `json:"service_state"`
	ServiceUserId string `json:"servicer_userid"`
}

type HumanList struct {
	ErrCode      int            `json:"errcode"`
	ErrMsg       string         `json:"errmsg"`
	ServicerList []ServicerList `json:"servicer_list"`
}

type ServicerList struct {
	UserID string `json:"userid"`
	Status int    `json:"status"`
}

type TokenCache struct {
	AccessToken string
	TokenExpire time.Time
	Mutex       sync.Mutex
}

// Map-based token cache keyed by app credentials
var tokenCacheMap = make(map[string]*TokenCache)
var tokenCacheMapMutex = sync.Mutex{}

// Generate a key for the token cache based on app credentials
func getTokenCacheKey(kbID, secret string) string {
	return kbID + ":" + secret
}

type UserImageCache struct {
	ImageID     string
	ImagePath   string
	ImageExpire time.Time
	Mutex       sync.Mutex
}

var UImageCache = &UserImageCache{}

type DefaultImageCache struct {
	ImageID     string
	ImageExpire time.Time
	Mutex       sync.Mutex
}

var DImageCache = &DefaultImageCache{}
