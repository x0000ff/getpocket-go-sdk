package pocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	host         = "https://getpocket.com/v3"
	authorizeUrl = "https://getpocket.com/auth/authorize?request_token=%s&redirect_uri=%s"

	endpointAdd          = "/add"
	endpointRequestToken = "/oauth/request"
	endpointAuthorize    = "/oauth/authorize"

	// xErrorHeader used to parse error message from Headers on non-2XX responses
	xErrorHeader = "X-Error"

	defaultTimeout = 5 * time.Second
)

type Client struct {
	client      *http.Client
	consumerKey string
}

type requestTokenRequest struct {
	ConsumerKey string `json:"consumer_key"`
	RedirectURI string `json:"redirect_uri"`
}

type authorizationRequest struct {
	ConsumerKey string `json:"consumer_key"`
	Code        string `json:"code"`
}

type AuthorizationResponse struct {
	AccessToken string `json:"access_token"`
	Username    string `json:"username"`
}

type addRequest struct {
	URL         string `json:"url"`
	Title       string `json:"title,omitempty"`
	Tags        string `json:"tags,omitempty"`
	AccessToken string `json:"access_token"`
	ConsumerKey string `json:"consumer_key"`
}

// AddInput holds data necessary to create new item in Pocket list
type AddInput struct {
	URL         string
	Title       string
	Tags        []string
	AccessToken string
}

func NewClient(consumerKey string) (*Client, error) {
	if consumerKey == "" {
		return nil, errors.New("consumer key is empty")
	}

	return &Client{
		client: &http.Client{
			Timeout: defaultTimeout,
		},
		consumerKey: consumerKey,
	}, nil
}

// GetRequestToken obtains the request token that is used to authorize user in your application
func (c *Client) GetRequestToken(ctx context.Context, redirectUrl string) (string, error) {
	input := &requestTokenRequest{
		ConsumerKey: c.consumerKey,
		RedirectURI: redirectUrl,
	}

	values, err := c.doHTTP(ctx, endpointRequestToken, input)
	if err != nil {
		return "", err
	}

	code := values.Get("code")
	if code == "" {
		return "", errors.New("empty request token in API response")
	}

	return code, nil
}

// GetAuthorizationURL generates link to authorize user
func (c *Client) GetAuthorizationURL(requestToken string, redirectURL string) (string, error) {
	if requestToken == "" {
		return "", errors.New("requestToken is empty")
	}

	if redirectURL == "" {
		return "", errors.New("redirectURL is empty")
	}

	return fmt.Sprintf(authorizeUrl, requestToken, redirectURL), nil
}

// Authorize generates access token for user, that authorized in your app via link
func (c *Client) Authorize(ctx context.Context, requestToken string) (*AuthorizationResponse, error) {
	if requestToken == "" {
		return nil, errors.New("empty request token")
	}

	input := &authorizationRequest{
		Code:        requestToken,
		ConsumerKey: c.consumerKey,
	}

	values, err := c.doHTTP(ctx, endpointAuthorize, input)
	if err != nil {
		return nil, err
	}

	accessToken, username := values.Get("access_token"), values.Get("username")
	if accessToken == "" {
		return nil, errors.New("empty acess token in API response")
	}

	return &AuthorizationResponse{
		AccessToken: accessToken,
		Username:    username,
	}, nil
}

func (i AddInput) validate() error {
	if i.URL == "" {
		return errors.New("required URL values is empty")
	}

	if i.AccessToken == "" {
		return errors.New("access token is empty")
	}

	return nil
}

func (i AddInput) generateRequest(consumerKey string) addRequest {
	return addRequest{
		URL:         i.URL,
		Tags:        strings.Join(i.Tags, ","),
		Title:       i.Title,
		AccessToken: i.AccessToken,
		ConsumerKey: consumerKey,
	}
}

// Add creates new item in Pocket list
func (c *Client) Add(ctx context.Context, input AddInput) error {
	if err := input.validate(); err != nil {
		return err
	}

	req := input.generateRequest(c.consumerKey)
	_, err := c.doHTTP(ctx, endpointAdd, req)

	return err
}

func (c *Client) doHTTP(ctx context.Context, endpoint string, body interface{}) (url.Values, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return url.Values{}, errors.WithMessage(err, "failed to marshal body")
	}

	newRequestURL := host + endpoint
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, newRequestURL, bytes.NewBuffer(b))
	if err != nil {
		return url.Values{}, errors.WithMessage(err, "failed to create new request")
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF8")

	resp, err := c.client.Do(req)
	if err != nil {
		return url.Values{}, errors.WithMessage(err, "failed to send http request")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Sprintf("API Error: %s", resp.Header.Get(xErrorHeader))
		return url.Values{}, errors.New(err)
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return url.Values{}, errors.WithMessage(err, "failed to read response body")
	}

	values, err := url.ParseQuery(string(responseBody))
	if err != nil {
		return url.Values{}, errors.WithMessage(err, "failed to parse response body")
	}

	return values, nil
}
