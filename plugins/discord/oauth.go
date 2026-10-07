package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const tokenURL = "https://discord.com/api/oauth2/token"

// scopes are requested once, when the user presses Connect.
var scopes = []string{"rpc", "rpc.voice.read", "rpc.voice.write", "rpc.video.read", "rpc.video.write", "rpc.screenshare.read", "rpc.screenshare.write"}

// errInvalidGrant means the code or refresh token is no longer valid; the
// user must approve again.
var errInvalidGrant = errors.New("Discord authorization expired")

// tokens is a successful token response. The access token is held in memory
// only; the refresh token is saved in the plugin settings.
type tokens struct {
	Access  string `json:"access_token"`
	Refresh string `json:"refresh_token"`
}

// exchange posts a token request: an authorization code, or a refresh token
// when code is empty.
func exchange(ctx context.Context, client *http.Client, endpoint string, s Settings, code string) (tokens, error) {
	form := url.Values{"client_id": {s.ClientID}, "client_secret": {s.ClientSecret}}
	if code != "" {
		form.Set("grant_type", "authorization_code")
		form.Set("code", code)
		form.Set("redirect_uri", s.redirect())
	} else {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", s.RefreshToken)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return tokens{}, fmt.Errorf("Discord token request: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		return tokens{}, err
	}
	if res.StatusCode != http.StatusOK {
		var failure struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &failure) == nil && failure.Error == "invalid_grant" {
			return tokens{}, errInvalidGrant
		}
		// The body may echo request details; report only the status and error code.
		return tokens{}, fmt.Errorf("Discord token request: HTTP %d %s", res.StatusCode, failure.Error)
	}
	var t tokens
	if err := json.Unmarshal(body, &t); err != nil {
		return tokens{}, fmt.Errorf("Discord token response: %w", err)
	}
	if t.Access == "" {
		return tokens{}, errors.New("Discord token response has no access token")
	}
	return t, nil
}
