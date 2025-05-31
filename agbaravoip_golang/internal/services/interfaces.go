package services

import "github.com/user/agbaravoip_golang/internal/domain"

// IAccountService defines the interface for account management operations.
type IAccountService interface {
	CreateMasterAccount(friendlyName string, authToken string) (*domain.Account, error)
	CreateSubAccount(parentAccountSid string, friendlyName string, authToken string) (*domain.Account, error)
	GetAccountBySID(accountSid string) (*domain.Account, error)
	GetSubAccounts(parentAccountSid string) ([]*domain.Account, error)
	UpdateAccount(accountSid string, friendlyName *string, statusStr *string) (*domain.Account, error)
	ValidateCredentials(accountSid string, plainToken string) (*domain.Account, error)
	// Add other methods as needed, e.g., DeleteAccount, ChangeAuthToken, etc.
}

// Ensure AccountService implements IAccountService
var _ IAccountService = (*AccountService)(nil)
