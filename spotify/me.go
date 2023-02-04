package spotify

import (
	"errors"
	"github.com/goccy/go-json"
	"io"
	"net/http"
)

const (
	nowPlayingURL = "https://api.spotify.com/v1/me/player/"
)

var (
	NotPlaying = errors.New("nothing is playing")
)

type (
	Device struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Id   string `json:"id"`
	}
	NowPlaying struct {
		DeviceData Device `json:"device"`
		Timestamp  int64  `json:"timestamp"`
		ProgressMs int    `json:"progress_ms"`
		Item       struct {
			Artists []struct {
				ExternalUrls struct {
					Spotify string `json:"spotify"`
				} `json:"external_urls"`
				Name string `json:"name"`
			} `json:"artists"`
			ExternalUrls struct {
				Spotify string `json:"spotify"`
			} `json:"external_urls"`
			DurationMs int    `json:"duration_ms"`
			Name       string `json:"name"`
		} `json:"item"`
		IsPlaying bool `json:"is_playing"`
	}
)

// GetNowPlaying Get the now playing data from Spotify's API
func (client Client) GetNowPlaying(refreshToken string) (*NowPlaying, error) {
	// get access token
	accessToken, err := client.getAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// build new request
	req, err := http.NewRequest("GET", nowPlayingURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+accessToken)

	// execute request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	// read all body
	bod, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if len(bod) == 0 {
		return nil, NotPlaying
	}
	// make body
	var body NowPlaying
	// fill in data from spotify api
	err = json.Unmarshal(bod, &body)

	return &body, err
}
