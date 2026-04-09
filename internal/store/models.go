package store

import "time"

// Account is the GORM database model
type Account struct {
	ID              uint       `gorm:"primaryKey"`
	Email           string     `gorm:"uniqueIndex;not null"`
	Password        string
	CodexPassword   string
	ClientID        string
	RefreshToken    string
	LastRefreshTime *time.Time
	AccountType     string
	Remark          string
	IsActive        bool `gorm:"default:true"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (a *Account) DaysSinceRefresh() int {
	if a.LastRefreshTime == nil {
		return -1
	}
	return int(time.Since(*a.LastRefreshTime).Hours() / 24)
}

func (a *Account) LastRefreshTimeStr() string {
	if a.LastRefreshTime == nil {
		return ""
	}
	return a.LastRefreshTime.Format(time.DateTime)
}

// AccountType is the GORM database model for account categories
type AccountType struct {
	ID    uint   `gorm:"primaryKey"`
	Code  string `gorm:"uniqueIndex;not null"`
	Label string
	Color string `gorm:"default:#409EFF"`
}
