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

// IFreeswitchServerService defines the interface for managing Freeswitch server configurations.
type IFreeswitchServerService interface {
	CreateFreeswitchServer(host string, port int, password string, outboundAddress string, isActive *bool) (*domain.FreeswitchServer, error)
	GetFreeswitchServerBySID(sid string) (*domain.FreeswitchServer, error)
	ListFreeswitchServers(filters map[string]interface{}) ([]*domain.FreeswitchServer, error)
	UpdateFreeswitchServer(sid string, updates map[string]interface{}) (*domain.FreeswitchServer, error)
	DeleteFreeswitchServer(sid string) error
    // TODO: Consider a method to securely handle password updates if direct map update is not ideal.
}

// IGatewayService defines the interface for managing Gateway configurations.
type IGatewayService interface {
	CreateGateway(
		accountSID string, // Gateways might be global or account-specific. Assuming account-specific for now as per domain model.
		fsServerSID *string,
		friendlyName string,
		gatewayString string,
		codecs []string,
		retryCount *int,
		timeoutSeconds *int,
		routes domain.GatewayRoutes,
		isEnabled *bool,
	) (*domain.Gateway, error)
	GetGatewayBySID(accountSID string, sid string) (*domain.Gateway, error) // Scoped to account
	ListGateways(accountSID string, filters map[string]interface{}) ([]*domain.Gateway, error) // Scoped to account
	UpdateGateway(accountSID string, sid string, updates map[string]interface{}) (*domain.Gateway, error) // Scoped to account
	DeleteGateway(accountSID string, sid string) error // Scoped to account
    ListGlobalGateways(filters map[string]interface{}) ([]*domain.Gateway, error) // For admin listing all gateways
    GetGlobalGatewayBySID(sid string) (*domain.Gateway, error) // For admin getting any gateway
    UpdateGlobalGateway(sid string, updates map[string]interface{}) (*domain.Gateway, error)
    DeleteGlobalGateway(sid string) error
}

