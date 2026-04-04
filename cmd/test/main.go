package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	imapClient "mailstore/internal/imap"
)

const (
	email        = "AnthonyGlenn9437@outlook.com"
	clientID     = "9e5f94bc-e8a4-4e73-b8be-63364c29d753"
	refreshToken = "M.C502_BAY.0.U.-CqVaiB2SIlHKLhifCvk0vtplBUDFXDIvLqfYj6tLIBOekUsisIW7S5uPs0XKygwwjUXwyZSOry42h*JunUGWEQ3QMSGonEvEA3VHBEpTc3LDmOIOcyotneKaQnMfU4O5PVzmw6S56hiODTIHqHHX0qXZ4mK2GlWC5ttB3ljCxi8rZIlFfZr43bDCq27BWDdBUZ2vbbmHzaihws1gGfwEKqiqycdAbIi0Px7CEw2oHBarZPksnahUkTLNsxGJo!7SU!ORIe1yWALRimyrXSA92a0qtbtDNgLAMPrU9e340OseindmtQlNbOPnr!uFD*Mo6SvxBVOX*M78QNIcvSacfDxbnH2UoNp8dCw8JSimmfS6PkFzBRNg5f0WbUNCSFg4FArSzfehWvk0noLEwHpgy1NgBerv03iRlZfOV654ORpMMq!jxHKk9ZI5hgRIbEOBnQ$$"
)

var endpoints = []struct {
	name  string
	url   string
	scope string
}{
	{
		name:  "consumers (IMAP scope)",
		url:   "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		scope: "https://outlook.office.com/IMAP.AccessAsUser.All offline_access",
	},
	{
		name:  "common (IMAP scope)",
		url:   "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		scope: "https://outlook.office.com/IMAP.AccessAsUser.All offline_access",
	},
	{
		name:  "live.com token endpoint",
		url:   "https://login.live.com/oauth20_token.srf",
		scope: "https://outlook.office.com/IMAP.AccessAsUser.All offline_access",
	},
	{
		name:  "consumers (openid scope)",
		url:   "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		scope: "openid profile email https://outlook.office.com/IMAP.AccessAsUser.All offline_access",
	},
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func tryRefresh(endpointURL, scope string) (string, string, error) {
	data := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"scope":         {scope},
	}
	req, _ := http.NewRequest("POST", endpointURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var tr tokenResponse
	json.Unmarshal(body, &tr)
	if tr.Error != "" {
		return "", "", fmt.Errorf("%s: %s", tr.Error, tr.ErrorDesc)
	}
	if tr.AccessToken == "" {
		return "", "", fmt.Errorf("empty access token, body: %s", string(body)[:min(200, len(body))])
	}
	return tr.AccessToken, tr.RefreshToken, nil
}

func main() {
	var accessToken string

	fmt.Println("=== Step 1: Try OAuth token refresh ===")
	for _, ep := range endpoints {
		fmt.Printf("  Trying %s...\n", ep.name)
		at, _, err := tryRefresh(ep.url, ep.scope)
		if err != nil {
			fmt.Printf("  FAIL: %v\n", err)
			continue
		}
		accessToken = at
		fmt.Printf("  OK! Access token: %s...\n\n", accessToken[:min(50, len(accessToken))])
		break
	}

	if accessToken == "" {
		log.Fatal("All token refresh attempts failed — the refresh token may be expired or already rotated.")
	}

	// Step 2: Fetch inbox mails
	fmt.Println("=== Step 2: Fetch INBOX ===")
	result, err := imapClient.FetchMails(email, accessToken, "inbox", 1, 5)
	if err != nil {
		log.Fatalf("FAIL fetch mails: %v", err)
	}
	fmt.Printf("OK  Total mails in inbox: %d\n", result.Total)
	for i, m := range result.Items {
		fmt.Printf("  [%d] UID=%-6s  %s\n      From: %s\n      Subj: %s\n",
			i+1, m.UID, m.Date, m.From, m.Subject)
	}

	// Step 3: Detail of first mail
	if len(result.Items) > 0 {
		uid := result.Items[0].UID
		fmt.Printf("\n=== Step 3: Fetch detail (UID=%s) ===\n", uid)
		detail, err := imapClient.FetchMailDetail(email, accessToken, "inbox", uid)
		if err != nil {
			fmt.Printf("FAIL detail: %v\n", err)
		} else {
			fmt.Printf("OK  Subject : %s\n", detail.Detail.Subject)
			fmt.Printf("    From    : %s\n", detail.Detail.From)
			fmt.Printf("    Date    : %s\n", detail.Detail.Date)
			fmt.Printf("    HTML    : %d bytes  Text: %d bytes\n",
				len(detail.Detail.BodyHTML), len(detail.Detail.BodyText))
		}
	}

	// Step 4: Junk folder
	fmt.Println("\n=== Step 4: Fetch Junk folder ===")
	junk, err := imapClient.FetchMails(email, accessToken, "junk", 1, 5)
	if err != nil {
		fmt.Printf("WARN junk: %v\n", err)
	} else {
		fmt.Printf("OK  Junk total: %d\n", junk.Total)
		for i, m := range junk.Items {
			fmt.Printf("  [%d] %s — %s\n", i+1, m.Date, m.Subject)
		}
	}

	fmt.Println("\n=== All done ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
