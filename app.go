package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"mailvault/internal/codex"
	imapClient "mailvault/internal/imap"
	"mailvault/internal/models"
	"mailvault/internal/oauth"
	"mailvault/internal/store"
)

// MailService is the Wails service exposing all app methods to the frontend.
type MailService struct {
	db *gorm.DB
}

func NewMailService(db *gorm.DB) *MailService {
	return &MailService{db: db}
}

func toResponse(acc store.Account) models.AccountResponse {
	return models.AccountResponse{
		ID:               acc.ID,
		Email:            acc.Email,
		Password:         acc.Password,
		CodexPassword:    acc.CodexPassword,
		ClientID:         acc.ClientID,
		RefreshToken:     acc.RefreshToken,
		LastRefreshTime:  acc.LastRefreshTimeStr(),
		AccountType:      acc.AccountType,
		Remark:           acc.Remark,
		IsActive:         acc.IsActive,
		DaysSinceRefresh: acc.DaysSinceRefresh(),
	}
}

func toTypeResponse(t store.AccountType) models.AccountTypeResponse {
	return models.AccountTypeResponse{
		ID:    t.ID,
		Code:  t.Code,
		Label: t.Label,
		Color: t.Color,
	}
}

// ─── Account CRUD ────────────────────────────────────────────────────────────

func (s *MailService) GetAccounts(query models.AccountQuery) (models.AccountsResponse, error) {
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}

	q := s.db.Model(&store.Account{})
	if query.ActiveOnly {
		q = q.Where("is_active = ?", true)
	}
	if query.Search != "" {
		like := "%" + query.Search + "%"
		q = q.Where("email LIKE ? OR remark LIKE ?", like, like)
	}
	if query.AccountType != "" {
		q = q.Where("account_type = ?", query.AccountType)
	}

	var total int64
	q.Count(&total)

	orderDir := "DESC"
	if strings.ToLower(query.SortOrder) == "asc" {
		orderDir = "ASC"
	}
	orderClause := "id DESC"
	switch query.SortBy {
	case "account_type":
		orderClause = "account_type " + orderDir + ", id DESC"
	case "last_refresh":
		orderClause = "last_refresh_time " + orderDir
	case "email":
		orderClause = "email " + orderDir
	}

	var accounts []store.Account
	offset := (query.Page - 1) * query.PageSize
	if err := q.Order(orderClause).Offset(offset).Limit(query.PageSize).Find(&accounts).Error; err != nil {
		return models.AccountsResponse{}, err
	}

	items := make([]models.AccountResponse, len(accounts))
	for i, acc := range accounts {
		items[i] = toResponse(acc)
	}

	return models.AccountsResponse{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *MailService) GetAccount(id uint) (models.AccountResponse, error) {
	var acc store.Account
	if err := s.db.First(&acc, id).Error; err != nil {
		return models.AccountResponse{}, err
	}
	return toResponse(acc), nil
}

func (s *MailService) CreateAccount(req models.CreateAccountRequest) (models.AccountResponse, error) {
	acc := store.Account{
		Email:         strings.TrimSpace(req.Email),
		Password:      req.Password,
		CodexPassword: req.CodexPassword,
		ClientID:      strings.TrimSpace(req.ClientID),
		RefreshToken: strings.TrimSpace(req.RefreshToken),
		AccountType:  req.AccountType,
		Remark:       req.Remark,
		IsActive:     true,
	}
	if err := s.db.Create(&acc).Error; err != nil {
		return models.AccountResponse{}, err
	}
	return toResponse(acc), nil
}

func (s *MailService) UpdateAccount(id uint, req models.UpdateAccountRequest) (models.AccountResponse, error) {
	var acc store.Account
	if err := s.db.First(&acc, id).Error; err != nil {
		return models.AccountResponse{}, err
	}
	acc.Email = strings.TrimSpace(req.Email)
	acc.Password = req.Password
	acc.CodexPassword = req.CodexPassword
	acc.ClientID = strings.TrimSpace(req.ClientID)
	acc.RefreshToken = strings.TrimSpace(req.RefreshToken)
	acc.AccountType = req.AccountType
	acc.Remark = req.Remark
	acc.IsActive = req.IsActive
	if err := s.db.Save(&acc).Error; err != nil {
		return models.AccountResponse{}, err
	}
	return toResponse(acc), nil
}

func (s *MailService) DeleteAccount(id uint) error {
	return s.db.Delete(&store.Account{}, id).Error
}

func (s *MailService) ArchiveAccount(id uint) error {
	return s.db.Model(&store.Account{}).Where("id = ?", id).Update("is_active", false).Error
}

func (s *MailService) ArchiveAllAccounts() error {
	return s.db.Model(&store.Account{}).Where("is_active = ?", true).Update("is_active", false).Error
}

// ─── Import / Export ─────────────────────────────────────────────────────────

// ImportAccounts parses lines in format: email----password----client_id----refresh_token
func (s *MailService) ImportAccounts(text string) (models.ImportResult, error) {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	result := models.ImportResult{Errors: []string{}}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "----")
		if len(parts) < 4 {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("invalid line: %s", line))
			continue
		}
		result.Total++
		acc := store.Account{
			Email:        strings.TrimSpace(parts[0]),
			Password:     strings.TrimSpace(parts[1]),
			ClientID:     strings.TrimSpace(parts[2]),
			RefreshToken: strings.TrimSpace(parts[3]),
			IsActive:     true,
		}
		if err := s.db.Where(store.Account{Email: acc.Email}).FirstOrCreate(&acc).Error; err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", acc.Email, err))
		} else {
			result.Success++
		}
	}

	return result, nil
}

func (s *MailService) ExportAccounts() (string, error) {
	var accounts []store.Account
	if err := s.db.Where("is_active = ?", true).Find(&accounts).Error; err != nil {
		return "", err
	}

	var sb strings.Builder
	for _, acc := range accounts {
		fmt.Fprintf(&sb, "%s----%s----%s----%s\n",
			acc.Email, acc.Password, acc.ClientID, acc.RefreshToken)
	}
	return sb.String(), nil
}

// ─── Token Refresh ───────────────────────────────────────────────────────────

func (s *MailService) RefreshToken(id uint) error {
	var acc store.Account
	if err := s.db.First(&acc, id).Error; err != nil {
		return err
	}
	_, newRT, err := oauth.RefreshAccessToken(acc.ClientID, acc.RefreshToken)
	if err != nil {
		return err
	}
	now := time.Now()
	return s.db.Model(&acc).Updates(map[string]any{
		"refresh_token":     newRT,
		"last_refresh_time": now,
	}).Error
}

func (s *MailService) RefreshAllTokens() (models.RefreshAllResult, error) {
	var accounts []store.Account
	if err := s.db.Where("is_active = ?", true).Find(&accounts).Error; err != nil {
		return models.RefreshAllResult{}, err
	}

	result := models.RefreshAllResult{
		Total:  len(accounts),
		Errors: []string{},
	}

	for _, acc := range accounts {
		_, newRT, err := oauth.RefreshAccessToken(acc.ClientID, acc.RefreshToken)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", acc.Email, err))
			continue
		}
		now := time.Now()
		s.db.Model(&acc).Updates(map[string]any{
			"refresh_token":     newRT,
			"last_refresh_time": now,
		})
		result.Success++
	}

	return result, nil
}

// ─── Account Types ────────────────────────────────────────────────────────────

func (s *MailService) GetAccountTypes() ([]models.AccountTypeResponse, error) {
	var types []store.AccountType
	if err := s.db.Find(&types).Error; err != nil {
		return nil, err
	}
	result := make([]models.AccountTypeResponse, len(types))
	for i, t := range types {
		result[i] = toTypeResponse(t)
	}
	return result, nil
}

func (s *MailService) CreateAccountType(req models.CreateTypeRequest) (models.AccountTypeResponse, error) {
	color := req.Color
	if color == "" {
		color = "#409EFF"
	}
	t := store.AccountType{Code: req.Code, Label: req.Label, Color: color}
	if err := s.db.Create(&t).Error; err != nil {
		return models.AccountTypeResponse{}, err
	}
	return toTypeResponse(t), nil
}

func (s *MailService) UpdateAccountType(id uint, req models.UpdateTypeRequest) (models.AccountTypeResponse, error) {
	var t store.AccountType
	if err := s.db.First(&t, id).Error; err != nil {
		return models.AccountTypeResponse{}, err
	}
	t.Code = req.Code
	t.Label = req.Label
	if req.Color != "" {
		t.Color = req.Color
	}
	if err := s.db.Save(&t).Error; err != nil {
		return models.AccountTypeResponse{}, err
	}
	return toTypeResponse(t), nil
}

func (s *MailService) DeleteAccountType(id uint) error {
	return s.db.Delete(&store.AccountType{}, id).Error
}

// ─── Mail ─────────────────────────────────────────────────────────────────────

func (s *MailService) getAccessToken(accountID uint) (store.Account, string, error) {
	var acc store.Account
	if err := s.db.First(&acc, accountID).Error; err != nil {
		return acc, "", err
	}
	accessToken, newRT, err := oauth.RefreshAccessToken(acc.ClientID, acc.RefreshToken)
	if err != nil {
		return acc, "", fmt.Errorf("token refresh failed: %w", err)
	}
	now := time.Now()
	s.db.Model(&acc).Updates(map[string]any{
		"refresh_token":     newRT,
		"last_refresh_time": now,
	})
	acc.RefreshToken = newRT
	return acc, accessToken, nil
}

func (s *MailService) GetMails(accountID uint, folder string, page, pageSize int) (models.MailListResponse, error) {
	acc, accessToken, err := s.getAccessToken(accountID)
	if err != nil {
		return models.MailListResponse{}, err
	}

	result, err := imapClient.FetchMails(acc.Email, accessToken, folder, page, pageSize)
	if err != nil {
		return models.MailListResponse{}, err
	}
	result.AccountID = accountID
	return *result, nil
}

func (s *MailService) GetMailDetail(accountID uint, folder, messageID string) (models.MailDetailResponse, error) {
	acc, accessToken, err := s.getAccessToken(accountID)
	if err != nil {
		return models.MailDetailResponse{}, err
	}

	result, err := imapClient.FetchMailDetail(acc.Email, accessToken, folder, messageID)
	if err != nil {
		return models.MailDetailResponse{}, err
	}
	result.AccountID = accountID
	return *result, nil
}

// ─── Liveness Check ──────────────────────────────────────────────────────────

// CheckAllAccounts tries to refresh the OAuth token for every active account
// concurrently (max 10 goroutines) and returns alive/dead status for each.
func (s *MailService) CheckAllAccounts() (models.CheckAllResult, error) {
	var accounts []store.Account
	if err := s.db.Where("is_active = ?", true).Find(&accounts).Error; err != nil {
		return models.CheckAllResult{}, err
	}

	results := make([]models.CheckResult, len(accounts))
	sem := make(chan struct{}, 10) // max 10 concurrent
	var wg sync.WaitGroup

	for i, acc := range accounts {
		wg.Add(1)
		go func(idx int, a store.Account) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			cr := models.CheckResult{ID: a.ID, Email: a.Email}
			_, newRT, err := oauth.RefreshAccessToken(a.ClientID, a.RefreshToken)
			if err != nil {
				cr.Alive = false
				cr.Error = err.Error()
			} else {
				cr.Alive = true
				// Save updated token
				now := time.Now()
				s.db.Model(&a).Updates(map[string]any{
					"refresh_token":     newRT,
					"last_refresh_time": now,
				})
			}
			results[idx] = cr
		}(i, acc)
	}
	wg.Wait()

	result := models.CheckAllResult{
		Total:   len(accounts),
		Results: results,
	}
	for _, r := range results {
		if r.Alive {
			result.Alive++
		} else {
			result.Dead++
		}
	}
	return result, nil
}

// DeleteDeadAccounts deletes all accounts whose IDs are in the provided list.
func (s *MailService) DeleteDeadAccounts(ids []uint) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res := s.db.Where("id IN ?", ids).Delete(&store.Account{})
	return int(res.RowsAffected), res.Error
}

// ─── Codex OAuth ─────────────────────────────────────────────────────────────

// codexConfigPath returns the path to codex_config.json next to the executable.
func codexConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config/codex_config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config", "codex_config.json")
}

func (s *MailService) GetCodexConfig() (models.CodexConfig, error) {
	data, err := os.ReadFile(codexConfigPath())
	if err != nil {
		return models.CodexConfig{}, nil
	}
	var cfg models.CodexConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.CodexConfig{}, err
	}
	return cfg, nil
}

func (s *MailService) SaveCodexConfig(cfg models.CodexConfig) error {
	p := codexConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func (s *MailService) GetCodexOAuth(id uint) (models.CodexOAuthResult, error) {
	var acc store.Account
	if err := s.db.First(&acc, id).Error; err != nil {
		return models.CodexOAuthResult{}, err
	}

	cfg, err := s.GetCodexConfig()
	if err != nil {
		return models.CodexOAuthResult{Success: false, Error: err.Error()}, nil
	}

	// Get MS access token to poll IMAP inbox for OTP
	accessToken, newRT, err := oauth.RefreshAccessToken(acc.ClientID, acc.RefreshToken)
	if err != nil {
		return models.CodexOAuthResult{Success: false, Error: "获取邮箱访问令牌失败: " + err.Error()}, nil
	}
	// Update refresh token
	now := time.Now()
	s.db.Model(&acc).Updates(map[string]any{
		"refresh_token":     newRT,
		"last_refresh_time": now,
	})

	// Snapshot current inbox UIDs before triggering OAuth (to detect new OTP email)
	oldUIDs, _ := imapClient.SnapshotInboxUIDs(acc.Email, accessToken)

	// OTP provider polls IMAP for new OTP email
	otpFn := codex.OTPProvider(func() (string, error) {
		return imapClient.WaitForOTPCode(acc.Email, accessToken, oldUIDs, 5*time.Minute)
	})

	oauthCfg := codex.OAuthConfig{
		Proxy:            cfg.Proxy,
		OAuthClientID:    cfg.OAuthClientID,
		OAuthRedirectURI: cfg.OAuthRedirectURI,
	}

	tokenJSON, err := codex.PerformCodexOAuthLogin(oauthCfg, acc.Email, acc.CodexPassword, otpFn, nil)
	if err != nil {
		return models.CodexOAuthResult{Success: false, Error: err.Error()}, nil
	}

	prettyJSON, err := json.MarshalIndent(tokenJSON, "", "  ")
	if err != nil {
		return models.CodexOAuthResult{Success: false, Error: err.Error()}, nil
	}
	return models.CodexOAuthResult{Success: true, JSON: string(prettyJSON)}, nil
}

// SaveCodexToken writes the given JSON content to the specified file path.
func (s *MailService) SaveCodexToken(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ─── CLIProxy Config ──────────────────────────────────────────────────────────

func cliProxyConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		return "config/cliproxy_config.json"
	}
	return filepath.Join(filepath.Dir(exe), "config", "cliproxy_config.json")
}

func (s *MailService) GetCLIProxyConfig() (models.CLIProxyConfig, error) {
	data, err := os.ReadFile(cliProxyConfigPath())
	if err != nil {
		return models.CLIProxyConfig{}, nil
	}
	var cfg models.CLIProxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return models.CLIProxyConfig{}, err
	}
	return cfg, nil
}

func (s *MailService) SaveCLIProxyConfig(cfg models.CLIProxyConfig) error {
	p := cliProxyConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// SyncCodexToken uploads the token JSON to CLIProxy via POST /v0/management/auth-files?name={email}.json
func (s *MailService) SyncCodexToken(content string) (models.SyncResult, error) {
	cfg, err := s.GetCLIProxyConfig()
	if err != nil {
		return models.SyncResult{Error: err.Error()}, nil
	}
	if cfg.URL == "" {
		return models.SyncResult{Error: "CLIProxy URL 未配置"}, nil
	}

	// Extract email from token JSON to use as filename
	var tokenData struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(content), &tokenData); err != nil || tokenData.Email == "" {
		return models.SyncResult{Error: "无法从 token 中提取邮箱"}, nil
	}
	filename := tokenData.Email + ".json"

	baseURL := strings.TrimRight(cfg.URL, "/")
	endpoint := baseURL + "/v0/management/auth-files?name=" + filename

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(content))
	if err != nil {
		return models.SyncResult{Error: err.Error()}, nil
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.SyncResult{Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.SyncResult{Error: fmt.Sprintf("服务器返回 %d", resp.StatusCode)}, nil
	}
	return models.SyncResult{Success: true}, nil
}

// GetCLIProxyAuthFiles fetches the list of auth files from CLIProxy management API.
func (s *MailService) GetCLIProxyAuthFiles() (models.CLIProxyAuthFilesResult, error) {
	cfg, err := s.GetCLIProxyConfig()
	if err != nil {
		return models.CLIProxyAuthFilesResult{}, err
	}
	if cfg.URL == "" {
		return models.CLIProxyAuthFilesResult{}, fmt.Errorf("CLIProxy URL 未配置")
	}

	baseURL := strings.TrimRight(cfg.URL, "/")
	req, err := http.NewRequest("GET", baseURL+"/v0/management/auth-files", nil)
	if err != nil {
		return models.CLIProxyAuthFilesResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.CLIProxyAuthFilesResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.CLIProxyAuthFilesResult{}, fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	var result models.CLIProxyAuthFilesResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return models.CLIProxyAuthFilesResult{}, err
	}
	return result, nil
}

func (s *MailService) GetCLIProxyStatus() (models.CLIProxyStatus, error) {
	cfg, err := s.GetCLIProxyConfig()
	if err != nil {
		return models.CLIProxyStatus{Error: err.Error()}, nil
	}
	if cfg.URL == "" {
		return models.CLIProxyStatus{Error: "CLIProxy URL 未配置"}, nil
	}

	baseURL := strings.TrimRight(cfg.URL, "/")
	client := &http.Client{Timeout: 10 * time.Second}

	doGet := func(path string) (*http.Response, error) {
		req, err := http.NewRequest("GET", baseURL+path, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
		return client.Do(req)
	}

	// Verify connection by hitting auth-files
	resp, err := doGet("/v0/management/auth-files")
	if err != nil {
		return models.CLIProxyStatus{Error: "连接失败: " + err.Error()}, nil
	}
	resp.Body.Close()
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return models.CLIProxyStatus{Error: fmt.Sprintf("认证失败 (HTTP %d)，请检查 Management Key", resp.StatusCode)}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.CLIProxyStatus{Error: fmt.Sprintf("服务器返回 HTTP %d", resp.StatusCode)}, nil
	}

	// Check if usage statistics are enabled
	statsEnabled := false
	resp2, err := doGet("/v0/management/usage-statistics-enabled")
	if err == nil && resp2.StatusCode == 200 {
		var body struct {
			Enabled bool `json:"usage-statistics-enabled"`
		}
		json.NewDecoder(resp2.Body).Decode(&body)
		resp2.Body.Close()
		statsEnabled = body.Enabled
	}

	return models.CLIProxyStatus{Connected: true, StatsEnabled: statsEnabled}, nil
}

func (s *MailService) GetCLIProxyUsage() (models.CLIProxyUsage, error) {
	cfg, err := s.GetCLIProxyConfig()
	if err != nil {
		return models.CLIProxyUsage{}, err
	}
	if cfg.URL == "" {
		return models.CLIProxyUsage{}, fmt.Errorf("CLIProxy URL 未配置")
	}

	baseURL := strings.TrimRight(cfg.URL, "/")
	req, err := http.NewRequest("GET", baseURL+"/v0/management/usage", nil)
	if err != nil {
		return models.CLIProxyUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return models.CLIProxyUsage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return models.CLIProxyUsage{}, fmt.Errorf("服务器返回 %d", resp.StatusCode)
	}

	// The API wraps the snapshot: {"usage": {...}, "failed_requests": ...}
	var wrapper struct {
		Usage models.CLIProxyUsage `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return models.CLIProxyUsage{}, err
	}
	return wrapper.Usage, nil
}
