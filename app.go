package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
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

	var accounts []store.Account
	offset := (query.Page - 1) * query.PageSize
	if err := q.Order("id DESC").Offset(offset).Limit(query.PageSize).Find(&accounts).Error; err != nil {
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
		Email:        strings.TrimSpace(req.Email),
		Password:     req.Password,
		ClientID:     strings.TrimSpace(req.ClientID),
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
