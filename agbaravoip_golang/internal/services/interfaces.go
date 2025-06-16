package services

import (
	"context" // Added import for context.Context
	"github.com/user/agbaravoip_golang/internal/domain"
)

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

	// Live Call Control methods
	PlayAudioOnCall(ctx context.Context, accountSid, callSid string, playURL string, loop int, legs string) (jobID string, err error)
	SayTextOnCall(ctx context.Context, accountSid, callSid string, text string, language *string, voice *string, legs string) (jobID string, err error)
	SendDTMFOnCall(ctx context.Context, accountSid, callSid string, digits string, durationMs *int, legs string) (jobID string, err error)
	StartRecordingCall(ctx context.Context, accountSid, callSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (recordingName string, jobID string, err error)
	StopRecordingCall(ctx context.Context, accountSid, callSid string, recordingNameOrUUID string) (jobID string, err error)
    HangupCall(ctx context.Context, accountSid, callSid string, cause string) (jobID string, err error)
    // Placeholder for more complex actions like redirecting/transferring a live call if needed:
    // TransferCall(ctx context.Context, accountSid, callSid string, destinationXMLUrl string) (jobID string, err error)
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

// ISMSService defines the interface for managing SMS messages.
type ISMSService interface {
	// For AgbaraXML <Sms> element execution
	SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error)

	// For API-driven SMS sending
	SendSMSViaAPI(ctx context.Context, accountSid, from, to, body string, statusCallbackURL *string) (*domain.SMSMessage, error)

	GetSMSBySID(ctx context.Context, accountSid string, sid string) (*domain.SMSMessage, error) // Ensure account scoping
	ListSMSMessages(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.SMSMessage, error)

	UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error
	RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error)
}

// SMSGatewayClient defines the interface for an external SMS gateway client.
// This should ideally be in a more specific gateway package, but placing here for now if not existing.
type SMSGatewayClient interface {
    SendSMS(ctx context.Context, to, from, body string, statusCallbackURL string) (gatewayMessageID string, err error)
}

// IRecordingService defines the interface for managing recording metadata.
type IRecordingService interface {
	GetRecordingBySID(ctx context.Context, accountSid string, recordingSid string) (*domain.Recording, error)
	ListRecordings(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Recording, error)
	DeleteRecording(ctx context.Context, accountSid string, recordingSid string) error // Deletes metadata

	// CreateRecording is currently part of CallServicerForESL (implemented by CallService)
	// and is used during call processing. If needed for other contexts, it could be added here
	// or CallServicerForESL could be embedded if a RecordingService needs to call it.
	// For now, assuming creation happens within call context.
}

// IConferenceService defines the interface for managing conference metadata and live control.
type IConferenceService interface {
	// Metadata methods (ensure account scoping)
	ListConferences(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Conference, error)
	GetConferenceBySID(ctx context.Context, accountSid string, confSid string) (*domain.Conference, error)
	// GetConferenceByName is more for internal use by AgbaraXML <Conference> element, may not need accountSid if name is unique per account already.
	// GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error)
	GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) // Used by XML, ensure account safety

	ListParticipants(ctx context.Context, accountSid string, confSid string) ([]*domain.ConferenceParticipant, error)
	GetParticipant(ctx context.Context, accountSid string, confSid string, participantSid string) (*domain.ConferenceParticipant, error)
    // GetParticipantByCallSID is useful internally and for events.
    GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error)


	// Status updates (ensure account scoping if called from an API context)
	UpdateConferenceStatus(ctx context.Context, accountSid string, confSid string, status domain.ConferenceStatus) error
	EndConference(ctx context.Context, accountSid string, confSid string, endTime time.Time) error

	// Participant metadata updates (ensure account scoping)
    // AddParticipant is more for ESL event handling based on call joining a conference.
	// AddParticipant(ctx context.Context, accountSid string, confSid string, callSid string, pSid string, isMuted bool, isModerator bool) (*domain.ConferenceParticipant, error)
	UpdateParticipantMuteStatus(ctx context.Context, accountSid string, confSid string, pSid string, isMuted bool) error // pSid is ConferenceParticipant SID
	UpdateParticipantModeratorStatus(ctx context.Context, accountSid string, confSid string, pSid string, isModerator bool) error
	RemoveParticipant(ctx context.Context, accountSid string, confSid string, pSid string, leaveTime time.Time) error


	// Live Conference Control methods (via ESL)
	PlayAudioInConference(ctx context.Context, accountSid string, confSid string, playURL string, loop int) (jobID string, err error)
	SayTextInConference(ctx context.Context, accountSid string, confSid string, text string, language *string, voice *string) (jobID string, err error)
	StartRecordingConference(ctx context.Context, accountSid string, confSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (recordingName string, jobID string, err error)
	StopRecordingConference(ctx context.Context, accountSid string, confSid string, recordingNameOrUUID string) (jobID string, err error)

	// Live Participant Control methods (via ESL)
    // These typically use member ID (Freeswitch concept, often CallSID of participant) or 'all'.
	MuteParticipantInConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string, mute bool) (jobID string, err error)
	KickParticipantFromConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string) (jobID string, err error)
    // AddParticipantToConference (Dial-out to conference - involves call origination then transfer to conference)
    // AddParticipantToConference(ctx context.Context, accountSid, confSid, to, fromNum, callerName string, timeout int) (*domain.Call, error)
}
