package wecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
)

const (
	// AuthURL api doc https://developer.work.weixin.qq.com/document/path/98152
	AuthWebURL    = "https://login.work.weixin.qq.com/wwlogin/sso/login"
	AuthAPPURL    = "https://open.weixin.qq.com/connect/oauth2/authorize"
	TokenURL      = "https://qyapi.weixin.qq.com/cgi-bin/gettoken"
	UserInfoURL   = "https://qyapi.weixin.qq.com/cgi-bin/auth/getuserinfo"
	UserDetailURL = "https://qyapi.weixin.qq.com/cgi-bin/user/get"
	// DepartmentListURL https://developer.work.weixin.qq.com/document/path/90344
	DepartmentListURL = "https://qyapi.weixin.qq.com/cgi-bin/department/list"
	// UserListUrl https://developer.work.weixin.qq.com/document/path/90337
	UserListUrl  = "https://qyapi.weixin.qq.com/cgi-bin/user/list"
	callbackPath = "/share/pro/v1/openapi/wecom/callback"
)

// Client 企业微信客户端
type Client struct {
	context     context.Context
	cache       *cache.Cache
	httpClient  *http.Client
	oauthConfig *oauth2.Config
	logger      *log.Logger
	corpID      string
	agentID     string
}

type TokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type UserInfoResponse struct {
	ErrCode        int    `json:"errcode"`
	ErrMsg         string `json:"errmsg"`
	UserID         string `json:"userid"`
	UserTicket     string `json:"user_ticket"`
	OpenID         string `json:"openid"`
	ExternalUserid string `json:"external_userid"`
}

type UserDetailResponse struct {
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
	Userid     string `json:"userid"`
	Name       string `json:"name"`
	Mobile     string `json:"mobile"`
	Gender     string `json:"gender"`
	Email      string `json:"email"`
	Avatar     string `json:"avatar"`
	OpenUserid string `json:"open_userid"`
}
type DepartmentListResponse struct {
	Errcode    int    `json:"errcode"`
	Errmsg     string `json:"errmsg"`
	Department []struct {
		Id               int      `json:"id"`
		Name             string   `json:"name"`
		NameEn           string   `json:"name_en"`
		DepartmentLeader []string `json:"department_leader"`
		Parentid         int      `json:"parentid"`
		Order            int      `json:"order"`
	} `json:"department"`
}

type UserListResponse struct {
	Errcode  int    `json:"errcode"`
	Errmsg   string `json:"errmsg"`
	Userlist []struct {
		Name       string `json:"name"`
		Department []int  `json:"department"`
		Position   string `json:"position"`
		Status     int    `json:"status"`
		Email      string `json:"email"`
		Avatar     string `json:"avatar"`
		Enable     int    `json:"enable"`
		Isleader   int    `json:"isleader"`
		Extattr    struct {
			Attrs []interface{} `json:"attrs"`
		} `json:"extattr"`
		HideMobile      int    `json:"hide_mobile"`
		Telephone       string `json:"telephone"`
		Order           []int  `json:"order"`
		ExternalProfile struct {
			ExternalAttr     []interface{} `json:"external_attr"`
			ExternalCorpName string        `json:"external_corp_name"`
		} `json:"external_profile"`
		MainDepartment int           `json:"main_department"`
		Alias          string        `json:"alias"`
		IsLeaderInDept []int         `json:"is_leader_in_dept"`
		Userid         string        `json:"userid"`
		DirectLeader   []interface{} `json:"direct_leader"`
	} `json:"userlist"`
}

func NewClient(ctx context.Context, logger *log.Logger, corpID, corpSecret, agentID, baseUrl string, cache *cache.Cache, isApp bool) (*Client, error) {
	redirectURI, err := url.JoinPath(baseUrl, callbackPath)
	if err != nil {
		return nil, err
	}

	authUrl := AuthWebURL
	if isApp {
		authUrl = AuthAPPURL
	}

	oauthConfig := &oauth2.Config{
		ClientID:     fmt.Sprintf("%s-%s", corpID, agentID),
		ClientSecret: corpSecret,
		RedirectURL:  redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: TokenURL,
		},
		Scopes: []string{"snsapi_privateinfo"},
	}

	return &Client{
		context:     ctx,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		cache:       cache,
		logger:      logger.WithModule("wecom.client"),
		oauthConfig: oauthConfig,
		corpID:      corpID,
		agentID:     agentID,
	}, nil
}

// GenerateAuthURL 生成授权 URL
func (c *Client) GenerateAuthURL(state string) string {
	params := url.Values{}
	params.Set("appid", c.corpID)
	params.Set("redirect_uri", c.oauthConfig.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "snsapi_privateinfo")
	params.Set("login_type", "CorpApp")
	params.Set("agentid", c.agentID)
	params.Set("state", state)

	authUrl := fmt.Sprintf("%s?%s", c.oauthConfig.Endpoint.AuthURL, params.Encode())
	if c.oauthConfig.Endpoint.AuthURL == AuthAPPURL {
		authUrl += "#wechat_redirect"
	}
	return authUrl
}

// GetAccessToken 获取企业微信访问令牌
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {

	cacheKey := fmt.Sprintf("wecom-access-token:%s", c.oauthConfig.ClientID)
	cachedData, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		return cachedData, nil
	}

	params := url.Values{}
	params.Set("corpid", c.corpID)
	params.Set("corpsecret", c.oauthConfig.ClientSecret)

	resp, err := c.httpClient.Get(fmt.Sprintf("%s?%s", TokenURL, params.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.ErrCode != 0 {
		return "", fmt.Errorf("failed to get access token: %s", tokenResp.ErrMsg)
	}

	if err := c.cache.Set(ctx, cacheKey, tokenResp.AccessToken, time.Duration(tokenResp.ExpiresIn-300)*time.Second).Err(); err != nil {
		c.logger.Warn("failed to set cache", log.Error(err))
	}

	return tokenResp.AccessToken, nil
}

// GetUserInfoByCode 通过授权码获取用户信息
func (c *Client) GetUserInfoByCode(ctx context.Context, code string) (*UserDetailResponse, error) {

	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("code", code)

	userInfoURL := fmt.Sprintf("%s?%s", UserInfoURL, params.Encode())

	c.logger.Debug("GetUserInfoByCode", log.Any("userInfoURL", userInfoURL))

	resp, err := c.httpClient.Get(userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("GetUserInfoByCode raw resp:", log.Any("raw", string(rawBody)))

	resp.Body = io.NopCloser(bytes.NewReader(rawBody))

	var userInfoResp UserInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&userInfoResp); err != nil {
		return nil, fmt.Errorf("failed to decode user info response: %w", err)
	}

	c.logger.Debug("GetUserInfoByCode resp:", log.Any("resp", userInfoResp))

	if userInfoResp.ErrCode != 0 {
		return nil, fmt.Errorf("failed to get user info: %s", userInfoResp.ErrMsg)
	}

	detailParams := url.Values{}
	detailParams.Set("access_token", accessToken)
	detailParams.Set("userid", userInfoResp.UserID)

	userDetailURL := fmt.Sprintf("%s?%s", UserDetailURL, detailParams.Encode())

	detailResp, err := c.httpClient.Get(userDetailURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user detail: %w", err)
	}
	defer detailResp.Body.Close()

	var UserDetailResp UserDetailResponse
	if err := json.NewDecoder(detailResp.Body).Decode(&UserDetailResp); err != nil {
		return nil, fmt.Errorf("failed to decode user detail response: %w", err)
	}

	c.logger.Debug("GetUserInfoByCode detail info", log.Any("resp", UserDetailResp))

	if UserDetailResp.Errcode != 0 {
		return nil, fmt.Errorf("failed to get user detail: %s", UserDetailResp.Errmsg)
	}

	return &UserDetailResp, nil
}

func (c *Client) GetDepartmentList(ctx context.Context) (*DepartmentListResponse, error) {

	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	params := url.Values{}
	params.Set("access_token", accessToken)

	departmentListURL := fmt.Sprintf("%s?%s", DepartmentListURL, params.Encode())

	resp, err := c.httpClient.Get(departmentListURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get department list: %w", err)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	c.logger.Debug("GetDepartmentList raw resp:", log.Any("raw", string(rawBody)))
	defer resp.Body.Close()

	resp.Body = io.NopCloser(bytes.NewReader(rawBody))

	var departmentListResponse DepartmentListResponse
	if err := json.NewDecoder(resp.Body).Decode(&departmentListResponse); err != nil {
		return nil, fmt.Errorf("failed to decode department list response: %w", err)
	}

	c.logger.Debug("GetDepartmentList resp:", log.Any("resp", departmentListResponse))

	if departmentListResponse.Errcode != 0 {
		return nil, fmt.Errorf("failed to get user info: %s", departmentListResponse.Errmsg)
	}

	return &departmentListResponse, nil
}

func (c *Client) GetUserList(ctx context.Context, deptID string) (*UserListResponse, error) {

	accessToken, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("department_id", deptID)

	userListUrl := fmt.Sprintf("%s?%s", UserListUrl, params.Encode())

	resp, err := c.httpClient.Get(userListUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get user list: %w", err)
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	c.logger.Debug("GetUserList raw resp:", log.Any("raw", string(rawBody)))

	resp.Body = io.NopCloser(bytes.NewReader(rawBody))

	var userListResponse UserListResponse
	if err := json.NewDecoder(resp.Body).Decode(&userListResponse); err != nil {
		return nil, fmt.Errorf("failed to decode user list response: %w", err)
	}

	c.logger.Debug("GetUserList resp:", log.Any("resp", userListResponse))

	if userListResponse.Errcode != 0 {
		return nil, fmt.Errorf("failed to get user info: %s", userListResponse.Errmsg)
	}

	return &userListResponse, nil
}
