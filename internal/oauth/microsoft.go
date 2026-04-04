package oauth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const tokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// RefreshAccessToken exchanges a refresh token for a new access token and refresh token.
func RefreshAccessToken(clientID, refreshToken string) (accessToken, newRefreshToken string, err error) {
	data := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {"https://outlook.office.com/IMAP.AccessAsUser.All offline_access"},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tr.Error != "" {
		return "", "", fmt.Errorf("token refresh failed: %s - %s", tr.Error, tr.ErrorDesc)
	}

	if tr.AccessToken == "" {
		return "", "", fmt.Errorf("no access token in response")
	}

	newRT := refreshToken
	if tr.RefreshToken != "" {
		newRT = tr.RefreshToken
	}

	return tr.AccessToken, newRT, nil
}
