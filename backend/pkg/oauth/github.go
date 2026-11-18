package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/log"
)

const (
	githubAuthorizeURL    = "https://github.com/login/oauth/authorize"
	githubTokenURL        = "https://github.com/login/oauth/access_token"
	githubUserInfoURL     = "https://api.github.com/user"
	githubUserEmailURL    = "https://api.github.com/user/emails"
	githubCallbackPathPro = "/share/pro/v1/openapi/github/callback"
	githubCallbackPath    = "/share/v1/openapi/github/callback"
)

func NewGithubClient(ctx context.Context, logger *log.Logger, clientID, clientSecret, redirectURI, proxyURL string) (*Client, error) {
	licenseEdition, ok := ctx.Value(consts.ContextKeyEdition).(consts.LicenseEdition)
	if !ok {
		return nil, fmt.Errorf("failed to retrieve license edition from context")
	}

	redirectURL, _ := url.Parse(redirectURI)
	redirectURL.Path = githubCallbackPath

	if licenseEdition > consts.LicenseEditionFree {
		redirectURL.Path = githubCallbackPathPro
	}

	redirectURI = redirectURL.String()

	var httpClient *http.Client
	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}

		httpClient = &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURLParsed),
			},
		}
		logger.Info("GitHub OAuth client configured with proxy", log.String("proxy", proxyURL))
	} else {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	config := Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:email"},
		AuthorizeURL: githubAuthorizeURL,
		TokenURL:     githubTokenURL,
		UserInfoURL:  githubUserInfoURL,
		IDField:      "id",
		NameField:    "login",
		AvatarField:  "avatar_url",
		EmailField:   "email",
		RedirectURI:  redirectURI,
	}

	oauthConfig := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  config.AuthorizeURL,
			TokenURL: config.TokenURL,
		},
		RedirectURL: redirectURI,
		Scopes:      config.Scopes,
	}

	if proxyURL != "" {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}

	return &Client{
		ctx:        ctx,
		logger:     logger.WithModule("pkg.oauth"),
		oauth:      oauthConfig,
		httpClient: httpClient,
		config:     &config,
	}, nil
}

func (c *Client) GetGithubPrimaryEmail(token *oauth2.Token) (string, error) {
	var client *http.Client
	if c.httpClient != nil {
		ctx := context.WithValue(c.ctx, oauth2.HTTPClient, c.httpClient)
		client = c.oauth.Client(ctx, token)
	} else {
		client = c.oauth.Client(c.ctx, token)
	}

	type Email struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	resp, err := client.Get(githubUserEmailURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	c.logger.Info("GetGithubPrimaryEmail:", log.Any("buf", string(buf)))

	var emails []Email
	if err := json.Unmarshal(buf, &emails); err != nil {
		return "", err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	return "", errors.New("no primary verified email found")
}
