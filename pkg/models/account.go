package models

import (
	"time"
)

// AccountType represents the type of an account (e.g., trial, full).
type AccountType string

// AccountStatus represents the status of an account (e.g., active, suspended).
type AccountStatus string

const (
	// Default Account Types (examples, actual values might differ based on AgbaraCommon)
	AccountTypeTrial AccountType = "trial"
	AccountTypeFull  AccountType = "full"

	// Default Account Statuses (examples)
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account model, corresponds to Domain/Objects/Account.cs
type Account struct {
	Sid          string        `json:"sid"`
	ParentSid    string        `json:"parentSid,omitempty"` // Omit if empty (master account)
	FriendlyName string        `json:"friendlyName"`
	PhoneNumber  string        `json:"phoneNumber,omitempty"`
	DateCreated  time.Time     `json:"dateCreated"`
	DateUpdated  time.Time     `json:"dateUpdated"`
	Type         AccountType   `json:"type"`
	Status       AccountStatus `json:"status"`
	AuthToken    string        `json:"-"` // Exclude AuthToken from JSON responses by default for security
                                         // It will be selected and used internally by the service/auth logic.
}

// CreateAccountRequest model for POST /Accounts and /Accounts/Master
type CreateAccountRequest struct {
	FriendlyName string `json:"friendlyName" binding:"required"`
}

// ChangeAccountStatusRequest model for POST /Accounts/{AccountSid} (for modifying status)
type ChangeAccountStatusRequest struct {
	Status AccountStatus `json:"status" binding:"required"` // Use AccountStatus type for validation
}

// Note: The C# AccountModule also implies a ChangeAccountType functionality in the IAccountService,
// but there's no specific request model shown in AccountModule for it, nor an endpoint.
// If ChangeAccountType needs an API endpoint, a ChangeAccountTypeRequest model would be needed.
// For now, only CreateAccountRequest and ChangeAccountStatusRequest are defined based on AccountModule.cs.
