package services

import "github.com/user/agbaravoip_golang/internal/domain"

type IAccountService interface {
	CreateMasterAccount(friendlyName string, authToken string) (*domain.Account, error)
	CreateSubAccount(parentAccountSid string, friendlyName string, authToken string) (*domain.Account, error)
	GetAccountBySID(accountSid string) (*domain.Account, error)
	GetSubAccounts(parentAccountSid string) ([]*domain.Account, error)
	UpdateAccount(accountSid string, friendlyName *string, statusStr *string) (*domain.Account, error)
	ValidateCredentials(accountSid string, plainToken string) (*domain.Account, error)
}

type IApplicationService interface {
	CreateApplication(accountSid string, app *domain.Application) (*domain.Application, error)
	GetApplicationBySID(accountSid string, appSid string) (*domain.Application, error)
	ListApplications(accountSid string) ([]*domain.Application, error)
	UpdateApplication(accountSid string, appSid string, updates map[string]interface{}) (*domain.Application, error)
	DeleteApplication(accountSid string, appSid string) error
	GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error)
}

type ICallService interface {
	OriginateCall(accountSid string, appSidOrNil *string, fromNum string, toNum string, answerURL string, timeoutSeconds *int) (*domain.Call, error)
	GetCallBySID(accountSid string, callSid string) (*domain.Call, error)
	ListCalls(accountSid string, filters map[string]interface{}) ([]*domain.Call, error) // Corrected signature
}


