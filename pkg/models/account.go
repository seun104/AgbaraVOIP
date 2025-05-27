package models

import (
	"time"
)

// AccountType (already defined)
type AccountType string
const (
	AccountTypeTrial AccountType = "trial"
	AccountTypeFull  AccountType = "full"
)

// AccountStatus (already defined)
type AccountStatus string
const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusClosed    AccountStatus = "closed"
)

// Account model update for gateway settings
type Account struct {
	Sid          string        `json:"sid"`
	ParentSid    string        `json:"parentSid,omitempty"` 
	FriendlyName string        `json:"friendlyName"`
	PhoneNumber  string        `json:"phoneNumber,omitempty"`
	DateCreated  time.Time     `json:"dateCreated"`
	DateUpdated  time.Time     `json:"dateUpdated"`
	Type         AccountType   `json:"type"`
	Status       AccountStatus `json:"status"`
	AuthToken    string        `json:"-"` 
	
	// New Gateway Settings
	DefaultOutboundGateway string `json:"defaultOutboundGateway,omitempty"`
	GatewaySelectionScript string `json:"gatewaySelectionScript,omitempty"` // e.g., name of a Lua script in FreeSWITCH
}

// CreateAccountRequest (already defined, no changes needed for this step)
type CreateAccountRequest struct {
	FriendlyName string `json:"friendlyName" binding:"required"`
}

// ChangeAccountStatusRequest (already defined)
type ChangeAccountStatusRequest struct {
	Status AccountStatus `json:"status" binding:"required"`
}

// Add a new DTO for updating account settings, including gateway config
type UpdateAccountSettingsRequest struct {
    FriendlyName           *string `json:"friendlyName,omitempty"` // Pointer to distinguish empty from not provided
    PhoneNumber            *string `json:"phoneNumber,omitempty"`
    Type                   *AccountType `json:"type,omitempty"`
    // Status is changed via ChangeAccountStatusRequest for clarity
    DefaultOutboundGateway *string `json:"defaultOutboundGateway,omitempty"`
    GatewaySelectionScript *string `json:"gatewaySelectionScript,omitempty"`
}
