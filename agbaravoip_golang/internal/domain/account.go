package domain

import "time"

// AccountType defines the type of account (e.g., trial, full).
type AccountType string

const (
	AccountTypeTrial AccountType = "trial"
	AccountTypeFull  AccountType = "full"
)

// AccountStatus defines the status of an account.
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account represents a user or sub-user account in the system.
type Account struct {
	ID          int64         `json:"-" gorm:"primaryKey"` // Internal DB ID
	SID         string        `json:"sid" gorm:"type:varchar(64);uniqueIndex"`
	ParentSID   *string       `json:"parent_sid,omitempty" gorm:"type:varchar(64);index"` // Pointer to allow null
	FriendlyName string        `json:"friendly_name" gorm:"type:varchar(255)"`
	PhoneNumber string        `json:"phone_number" gorm:"type:varchar(64)"`
	AuthToken   string        `json:"-" gorm:"type:varchar(255)"` // Store hashed token
	Type        AccountType   `json:"type" gorm:"type:varchar(20);default:trial"`
	Status      AccountStatus `json:"status" gorm:"type:varchar(20);default:active"`
	CreatedAt   time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
}
