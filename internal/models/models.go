// Package models contains DTO types used as Wails frontend bindings.
// IMPORTANT: Do NOT import "time" or any package with time.Time here.
package models

// AccountResponse is the DTO returned to the frontend.
type AccountResponse struct {
	ID               uint   `json:"id"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	CodexPassword    string `json:"codex_password"`
	ClientID         string `json:"client_id"`
	RefreshToken     string `json:"refresh_token"`
	LastRefreshTime  string `json:"last_refresh_time"`
	AccountType      string `json:"account_type"`
	Remark           string `json:"remark"`
	IsActive         bool   `json:"is_active"`
	DaysSinceRefresh int    `json:"days_since_refresh"`
}

type AccountTypeResponse struct {
	ID    uint   `json:"id"`
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

type AccountQuery struct {
	Search      string `json:"search"`
	AccountType string `json:"account_type"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	ActiveOnly  bool   `json:"active_only"`
	SortBy      string `json:"sort_by"`    // id | account_type | last_refresh | email
	SortOrder   string `json:"sort_order"` // asc | desc
}

type AccountsResponse struct {
	Items    []AccountResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type CreateAccountRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CodexPassword string `json:"codex_password"`
	ClientID      string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	AccountType  string `json:"account_type"`
	Remark       string `json:"remark"`
}

type UpdateAccountRequest struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	CodexPassword string `json:"codex_password"`
	ClientID      string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	AccountType  string `json:"account_type"`
	Remark       string `json:"remark"`
	IsActive     bool   `json:"is_active"`
}

type CreateTypeRequest struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

type UpdateTypeRequest struct {
	Code  string `json:"code"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// CheckResult holds the liveness result for a single account.
type CheckResult struct {
	ID      uint   `json:"id"`
	Email   string `json:"email"`
	Alive   bool   `json:"alive"`
	Error   string `json:"error"`
}

type CheckAllResult struct {
	Total   int           `json:"total"`
	Alive   int           `json:"alive"`
	Dead    int           `json:"dead"`
	Results []CheckResult `json:"results"`
}

type ImportResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

type RefreshAllResult struct {
	Total   int      `json:"total"`
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

type CodexConfig struct {
	Proxy            string `json:"proxy"`
	OAuthClientID    string `json:"oauth_client_id"`
	OAuthRedirectURI string `json:"oauth_redirect_uri"`
}

type CodexOAuthResult struct {
	Success bool   `json:"success"`
	JSON    string `json:"json"`
	Error   string `json:"error"`
}

type CLIProxyConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

type CLIProxyAuthFile struct {
	ID             string `json:"id"`
	AuthIndex      string `json:"auth_index"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Provider       string `json:"provider"`
	Label          string `json:"label"`
	Status         string `json:"status"`
	StatusMessage  string `json:"status_message"`
	Disabled       bool   `json:"disabled"`
	Unavailable    bool   `json:"unavailable"`
	RuntimeOnly    bool   `json:"runtime_only"`
	Source         string `json:"source"`
	Size           int64  `json:"size"`
	Email          string `json:"email,omitempty"`
	AccountType    string `json:"account_type,omitempty"`
	Account        string `json:"account,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
	LastRefresh    string `json:"last_refresh,omitempty"`
	NextRetryAfter string `json:"next_retry_after,omitempty"`
	Path           string `json:"path,omitempty"`
	Priority       int    `json:"priority,omitempty"`
	Note           string `json:"note,omitempty"`
}

type CLIProxyAuthFilesResult struct {
	Files []CLIProxyAuthFile `json:"files"`
}

type SyncResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type CLIProxyStatus struct {
	Connected    bool   `json:"connected"`
	StatsEnabled bool   `json:"stats_enabled"`
	Error        string `json:"error"`
}

type CLIProxyModelStats struct {
	TotalRequests int64 `json:"total_requests"`
	TotalTokens   int64 `json:"total_tokens"`
}

type CLIProxyAPIStats struct {
	TotalRequests int64                         `json:"total_requests"`
	TotalTokens   int64                         `json:"total_tokens"`
	Models        map[string]CLIProxyModelStats `json:"models"`
}

type CLIProxyUsage struct {
	TotalRequests int64                       `json:"total_requests"`
	SuccessCount  int64                       `json:"success_count"`
	FailureCount  int64                       `json:"failure_count"`
	TotalTokens   int64                       `json:"total_tokens"`
	RequestsByDay map[string]int64            `json:"requests_by_day"`
	TokensByDay   map[string]int64            `json:"tokens_by_day"`
	APIs          map[string]CLIProxyAPIStats `json:"apis"`
}

// Mail types

type MailItem struct {
	UID     string `json:"uid"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	Date    string `json:"date"`
	Folder  string `json:"folder"`
}

type MailListResponse struct {
	AccountID uint       `json:"account_id"`
	Email     string     `json:"email"`
	Folder    string     `json:"folder"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	Total     int        `json:"total"`
	Items     []MailItem `json:"items"`
}

type MailDetail struct {
	Subject  string `json:"subject"`
	From     string `json:"from"`
	To       string `json:"to"`
	Date     string `json:"date"`
	BodyText string `json:"body_text"`
	BodyHTML string `json:"body_html"`
}

type MailDetailResponse struct {
	AccountID uint       `json:"account_id"`
	Email     string     `json:"email"`
	Folder    string     `json:"folder"`
	MessageID string     `json:"message_id"`
	Detail    MailDetail `json:"detail"`
}
