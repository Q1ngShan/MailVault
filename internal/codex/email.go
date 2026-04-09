package codex

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// CFWorkerConfig holds Cloudflare Worker email service settings.
type CFWorkerConfig struct {
	Domain        string `json:"cf_worker_domain"`
	EmailDomain   string `json:"cf_email_domain"`
	AdminPassword string `json:"cf_admin_password"`
}

type mailItem struct {
	ID      string `json:"id"`
	Raw     string `json:"raw"`
	Source  string `json:"source"`
	Subject string `json:"subject"`
}

type mailListResponse struct {
	Results []mailItem `json:"results"`
}

// GetCFToken obtains a JWT for an existing email address via the CF Worker admin API.
func GetCFToken(client *HTTPClient, cfg CFWorkerConfig, emailName string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"enablePrefix": false,
		"name":         emailName,
		"domain":       cfg.EmailDomain,
	})
	headers := map[string]string{
		"Content-Type": "application/json",
		"x-admin-auth": cfg.AdminPassword,
	}
	resp, err := client.Post(
		fmt.Sprintf("https://%s/admin/new_address", cfg.Domain),
		headers,
		body,
	)
	if err != nil {
		return "", err
	}
	var result struct {
		JWT string `json:"jwt"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.JWT, nil
}

// fetchEmails fetches emails from the CF Worker API using a JWT.
func fetchEmails(client *HTTPClient, domain, cfToken string) ([]mailItem, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + cfToken,
	}
	body, _, err := client.Get(
		fmt.Sprintf("https://%s/api/mails?limit=10&offset=0", domain),
		headers,
	)
	if err != nil {
		return nil, err
	}
	var resp mailListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return resp.Results, nil
}

var (
	// Regex patterns for extracting 6-digit OTP from email content
	reOTPStyled  = regexp.MustCompile(`background-color:\s*#F3F3F3[^>]*>[\s\S]*?(\d{6})[\s\S]*?</p>`)
	reOTPTag     = regexp.MustCompile(`>\s*(\d{6})\s*<`)
	reOTPGeneric = regexp.MustCompile(`(?:^|[^#&])\b(\d{6})\b`)
)

// extractVerificationCode extracts a 6-digit OTP from email content.
func extractVerificationCode(content string) string {
	if content == "" {
		return ""
	}
	// Strategy 1: styled HTML
	if m := reOTPStyled.FindStringSubmatch(content); len(m) > 1 && m[1] != "177010" {
		return m[1]
	}
	// Strategy 2: between tags
	if m := reOTPTag.FindStringSubmatch(content); len(m) > 1 && m[1] != "177010" {
		return m[1]
	}
	// Strategy 3: generic
	if m := reOTPGeneric.FindStringSubmatch(content); len(m) > 1 && m[1] != "177010" {
		return m[1]
	}
	return ""
}

// WaitForVerificationCode polls the CF Worker email API for an OTP code.
// It records existing email IDs first, then waits for a new email with a code.
func WaitForVerificationCode(client *HTTPClient, domain, cfToken string, oldIDs map[string]bool, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		emails, err := fetchEmails(client, domain, cfToken)
		if err != nil {
			time.Sleep(3 * time.Second)
			continue
		}
		for _, item := range emails {
			if oldIDs[item.ID] {
				continue
			}
			code := extractVerificationCode(item.Raw)
			if code != "" {
				return code, nil
			}
		}
		time.Sleep(3 * time.Second)
	}
	return "", fmt.Errorf("等待验证码超时")
}

// CollectOldMailIDs fetches existing emails and returns their IDs as a set.
func CollectOldMailIDs(client *HTTPClient, domain, cfToken string) map[string]bool {
	ids := make(map[string]bool)
	emails, err := fetchEmails(client, domain, cfToken)
	if err != nil {
		return ids
	}
	for _, item := range emails {
		ids[item.ID] = true
	}
	return ids
}
