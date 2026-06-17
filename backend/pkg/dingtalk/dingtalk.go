package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dingtalkcard_1_0 "github.com/alibabacloud-go/dingtalk/card_1_0"
	dingtalkoauth2_1_0 "github.com/alibabacloud-go/dingtalk/v2/oauth2_1_0"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/store/cache"
)

const (
	callbackPath      = "/share/pro/v1/openapi/dingtalk/callback"
	userInfoUrl       = "https://api.dingtalk.com/v1.0/contact/users/me"
	DepartmentListUrl = "https://oapi.dingtalk.com/department/list"
	// https://open.dingtalk.com/document/isvapp/queries-the-complete-information-of-a-department-user
	UserListUrl = "https://oapi.dingtalk.com/topapi/v2/user/list"
)

type Client struct {
	ctx             context.Context
	logger          *log.Logger
	httpClient      *http.Client
	clientID        string
	clientSecret    string
	oauthClient     *dingtalkoauth2_1_0.Client
	cardClient      *dingtalkcard_1_0.Client
	dingTalkAuthURL string
	cache           *cache.Cache
}

// UserInfo 用于解析获取用户信息的接口返回
type UserInfo struct {
	Nick      string `json:"nick"`
	UnionID   string `json:"unionId"`
	OpenID    string `json:"openId"`
	AvatarURL string `json:"avatarUrl"`
	StateCode string `json:"stateCode"`
}

// DepartmentListRsp 用于解析组织信息接口返回
type DepartmentListRsp struct {
	Errcode    int `json:"errcode"`
	Department []struct {
		CreateDeptGroup bool   `json:"createDeptGroup"`
		Name            string `json:"name"`
		Id              int    `json:"id"`
		AutoAddUser     bool   `json:"autoAddUser"`
		Parentid        int    `json:"parentid,omitempty"`
	} `json:"department"`
	Errmsg string `json:"errmsg"`
}

type GetUserListResp struct {
	Errcode int `json:"errcode"`
	Result  struct {
		HasMore bool         `json:"has_more"`
		List    []UserDetail `json:"list"`
	} `json:"result"`
	Errmsg string `json:"errmsg"`
}

type UserDetail struct {
	Active           bool   `json:"active"`
	Admin            bool   `json:"admin"`
	Avatar           string `json:"avatar"`
	Boss             bool   `json:"boss"`
	DeptIdList       []int  `json:"dept_id_list"`
	DeptOrder        int64  `json:"dept_order"`
	Email            string `json:"email"`
	ExclusiveAccount bool   `json:"exclusive_account"`
	HideMobile       bool   `json:"hide_mobile"`
	JobNumber        string `json:"job_number"`
	Leader           bool   `json:"leader"`
	Mobile           string `json:"mobile"`
	Name             string `json:"name"`
	Remark           string `json:"remark"`
	StateCode        string `json:"state_code"`
	Telephone        string `json:"telephone"`
	Title            string `json:"title"`
	Unionid          string `json:"unionid"`
	Userid           string `json:"userid"`
	WorkPlace        string `json:"work_place"`
}

func NewDingTalkClient(ctx context.Context, logger *log.Logger, clientId, clientSecret string, cache *cache.Cache) (*Client, error) {
	config := &openapi.Config{}
	config.Protocol = tea.String("https")
	config.RegionId = tea.String("central")
	oauthClient, err := dingtalkoauth2_1_0.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create oauth client: %w", err)
	}
	cardClient, err := dingtalkcard_1_0.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create card client: %w", err)
	}
	return &Client{
		ctx:             ctx,
		logger:          logger.WithModule("pkg.dingtalk"),
		httpClient:      &http.Client{},
		clientID:        clientId,
		clientSecret:    clientSecret,
		oauthClient:     oauthClient,
		cardClient:      cardClient,
		dingTalkAuthURL: "https://login.dingtalk.com/oauth2/auth",
		cache:           cache,
	}, nil
}

// GenerateAuthURL 生成钉钉授权URL
func (c *Client) GenerateAuthURL(baseUrl string, state string) string {
	redirectURI, err := url.JoinPath(baseUrl, callbackPath)
	if err != nil {
		c.logger.Error("failed to join path", log.Error(err))
		return ""
	}

	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", c.clientID)
	params.Add("redirect_uri", redirectURI)
	params.Add("scope", "openid")
	params.Add("state", state)
	params.Add("prompt", "consent")

	return fmt.Sprintf("%s?%s", c.dingTalkAuthURL, params.Encode())
}

func (c *Client) GetAccessTokenByCode(code string) (string, error) {
	request := &dingtalkoauth2_1_0.GetUserTokenRequest{
		ClientId:     tea.String(c.clientID),
		ClientSecret: tea.String(c.clientSecret),
		Code:         tea.String(code),
		GrantType:    tea.String("authorization_code"),
	}
	response, err := c.oauthClient.GetUserToken(request)
	if err != nil {
		return "", fmt.Errorf("failed to get user access token: %w", err)
	}
	accessToken := tea.StringValue(response.Body.AccessToken)
	return accessToken, nil
}

func (c *Client) GetAccessToken() (string, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("dingtalk-access-token:%s", c.clientID)
	cachedData, err := c.cache.Get(ctx, cacheKey).Result()
	if err == nil && cachedData != "" {
		return cachedData, nil
	}

	request := &dingtalkoauth2_1_0.GetAccessTokenRequest{
		AppKey:    tea.String(c.clientID),
		AppSecret: tea.String(c.clientSecret),
	}
	response, tryErr := func() (_resp *dingtalkoauth2_1_0.GetAccessTokenResponse, _e error) {
		defer func() {
			if r := tea.Recover(recover()); r != nil {
				_e = r
			}
		}()
		_resp, _err := c.oauthClient.GetAccessToken(request)
		if _err != nil {
			return nil, _err
		}

		return _resp, nil
	}()
	if tryErr != nil {
		return "", tryErr
	}
	accessToken := *response.Body.AccessToken
	c.logger.Debug("get access token", log.String("access_token", accessToken), log.Int("expire_in", int(*response.Body.ExpireIn)))

	if err := c.cache.Set(ctx, cacheKey, accessToken, time.Duration(*response.Body.ExpireIn-300)*time.Second).Err(); err != nil {
		c.logger.Warn("failed to set cache", log.Error(err))
	}

	return accessToken, nil
}

func (c *Client) GetUserInfoByCode(code string) (*UserInfo, error) {
	req, err := http.NewRequest("GET", userInfoUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	accessToken, err := c.GetAccessTokenByCode(code)
	if err != nil {
		return nil, err
	}

	// Set request headers
	req.Header.Set("x-acs-dingtalk-access-token", accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DingTalk API returned non-200 status: %s, response: %s", resp.Status, string(body))
	}

	var userInfo UserInfo
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	return &userInfo, nil
}

func (c *Client) GetDepartmentList() (*DepartmentListRsp, error) {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("access_token", accessToken)
	requestURL := fmt.Sprintf("%s?%s", DepartmentListUrl, params.Encode())

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DingTalk API returned non-200 status: %s, response: %s", resp.Status, string(body))
	}

	c.logger.Debug("DepartmentListUrl:", log.String("body", string(body)))
	var departmentListRsp DepartmentListRsp
	if err := json.Unmarshal(body, &departmentListRsp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	if departmentListRsp.Errcode != 0 {
		return nil, fmt.Errorf("DingTalk API error: errcode=%d errmsg=%s", departmentListRsp.Errcode, departmentListRsp.Errmsg)
	}

	return &departmentListRsp, nil
}

func (c *Client) GetAllUserList(deptID int) ([]UserDetail, error) {
	depth := 0
	const maxDepth = 10

	userList := make([]UserDetail, 0)
	for depth < maxDepth {
		resp, err := c.GetUserList(deptID)
		if err != nil {
			return nil, err
		}
		if len(resp.Result.List) > 0 {
			userList = append(userList, resp.Result.List...)
		}
		if !resp.Result.HasMore {
			break
		}
		depth++
	}
	return userList, nil
}

func (c *Client) GetUserList(deptID int) (*GetUserListResp, error) {
	accessToken, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Add("access_token", accessToken)
	requestURL := fmt.Sprintf("%s?%s", UserListUrl, params.Encode())

	bodyMap := map[string]interface{}{
		"dept_id": deptID,
		"size":    100,
		"cursor":  0,
	}

	jsonData, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DingTalk API returned non-200 status: %s, response: %s", resp.Status, string(body))
	}

	c.logger.Debug("GetUserList:", log.String("body", string(body)))
	var getUserListResp GetUserListResp
	if err := json.Unmarshal(body, &getUserListResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	if getUserListResp.Errcode != 0 {
		return nil, fmt.Errorf("DingTalk GetUserList error: errcode=%d errcode=%s", getUserListResp.Errcode, getUserListResp.Errmsg)
	}

	return &getUserListResp, nil
}
