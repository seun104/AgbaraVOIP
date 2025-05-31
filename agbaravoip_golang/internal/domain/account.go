package domain

import (
	"fmt"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AccountType defines the type of account (e.g., trial, full).
type AccountType string

const (
	AccountTypeTrial AccountType = "trial"
	AccountTypeFull  AccountType = "full"
)

// AccountStatus defines the status of an account (e.g., active, suspended).
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account represents a user account in the system.
type Account struct {
	ID        uint            `gorm:"primaryKey"`
	SID       string          `gorm:"type:varchar(64);uniqueIndex;not null"` // Publicly visible SID
	ParentSID *string         `gorm:"type:varchar(64);index"`           // For sub-accounts, references another Account.SID
	FriendlyName string       `gorm:"type:varchar(255)"`
	PhoneNumber string        `gorm:"type:varchar(50)"`
	HashedAuthToken string    `gorm:"type:varchar(255);not null"` // Store hashed token
	Type      AccountType     `gorm:"type:varchar(50);not null"`
	Status    AccountStatus   `gorm:"type:varchar(50);not null"`
	CreatedAt time.Time       `gorm:"autoCreateTime"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt  `gorm:"index"` // For soft deletes

	// Associations (optional, GORM can use these)
	// ParentAccount *Account `gorm:"foreignKey:ParentSID;references:SID"`
	// SubAccounts   []Account `gorm:"foreignKey:ParentSID;references:SID"`
}

// BeforeCreate is a GORM hook that runs before a new record is created.
func (account *Account) BeforeCreate(tx *gorm.DB) (err error) {
	if account.SID == "" {
		account.SID = "AC" + uuid.NewString()
	}
	// Default status and type if not set
	if account.Status == "" {
		account.Status = AccountStatusActive
	}
	if account.Type == "" {
		account.Type = AccountTypeTrial
	}
	return
}

func (s AccountStatus) String() string {
    return string(s)
}

func (t AccountType) String() string {
    return string(t)
}

// IsValidStatus checks if the provided status is a valid AccountStatus
func IsValidAccountStatus(statusStr string) (AccountStatus, bool) {
	status := AccountStatus(statusStr)
	switch status {
	case AccountStatusActive, AccountStatusSuspended, AccountStatusClosed:
		return status, true
	default:
		return "", false
	}
}

// IsValidAccountType checks if the provided type is a valid AccountType
func IsValidAccountType(typeStr string) (AccountType, bool) {
	acctType := AccountType(typeStr)
	switch acctType {
	case AccountTypeTrial, AccountTypeFull:
		return acctType, true
	default:
		return "", false
	}
}

// Helper function to generate a display name if friendly name is empty
func (account *Account) GetDisplayName() string {
	if account.FriendlyName != "" {
		return account.FriendlyName
	}
	return fmt.Sprintf("Account %s", account.SID)
}
