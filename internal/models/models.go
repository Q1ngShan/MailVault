// Package models contains DTO types used as Wails frontend bindings.
// IMPORTANT: Do NOT import "time" or any package with time.Time here.
package models

// AccountResponse is the DTO returned to the frontend.
type AccountResponse struct {
	ID               uint   `json:"id"`
	Email            string `json:"email"`
	Password         string `json:"password"`
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
}

type AccountsResponse struct {
	Items    []AccountResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}

type CreateAccountRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
	AccountType  string `json:"account_type"`
	Remark       string `json:"remark"`
}

type UpdateAccountRequest struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ClientID     string `json:"client_id"`
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
