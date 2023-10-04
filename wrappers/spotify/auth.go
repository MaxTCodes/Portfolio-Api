package spotify

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/goccy/go-json"
)

const (
	refreshTokenURL = "https://accounts.spotify.com/authorize?client_id=%s&response_type=code&redirect_uri=%s&scope=user-read-playback-state&state=%s"
	accessTokenURL  = "https://accounts.spotify.com/api/token"
)

type (
	Client struct {
		Secret string
		ID     string
		Auth   string
	}
	tokenBody struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
)

func New(secret, id string) *Client {
	return &Client{
		Secret: secret,
		ID:     id,
		Auth:   base64.URLEncoding.EncodeToString([]byte(id + ":" + secret)),
	}
}

func (client Client) GetRefreshTokenURL(callbackURI string) string {
	return fmt.Sprintf(refreshTokenURL, client.ID, callbackURI, strconv.Itoa(int(time.Now().Unix())))
}

// GetRefreshToken get a valid refresh token from the spotify api
func (client Client) GetRefreshToken(oauthCode, callbackURI string) (string, error) {
	postBody := url.Values{}
	postBody.Add("grant_type", "authorization_code")
	postBody.Add("code", oauthCode)
	postBody.Add("redirect_uri", callbackURI)
	// only did this to make it easier to read
	bodyBytes := []byte(postBody.Encode())
	req, err := http.NewRequest("POST", accessTokenURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Basic "+client.Auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokens, err := makeTokenReq(req)
	return tokens.RefreshToken, err
}

func (client Client) getAccessToken(refreshToken string) (string, error) {
	postBody := url.Values{}
	postBody.Add("grant_type", "refresh_token")
	postBody.Add("refresh_token", refreshToken)
	// only did this to make it easier to read
	bodyBytes := []byte(postBody.Encode())
	req, err := http.NewRequest("POST", accessTokenURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "Basic "+client.Auth)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokens, err := makeTokenReq(req)
	return tokens.AccessToken, err
}

func makeTokenReq(req *http.Request) (tokenBody, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenBody{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenBody{}, err
	}
	var body tokenBody
	// fewer lines = better lmao
	err = json.Unmarshal(response, &body)
	return body, nil
}
