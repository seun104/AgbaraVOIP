package domain

import "github.com/sirupsen/logrus"

// MinimalCallContext provides the essential context for a call being controlled.
type MinimalCallContext interface {
	Log() *logrus.Entry
	GetUuid() string
	GetAccountSid() string
	GetApplicationSid() string
	GetAnswerURL() string
	// Add other relevant getters as needed by elements e.g. GetCallerID()

	IsHangupInitiated() bool
	SetHangupInitiated()

	SetPendingRecording(info interface{})
	GetPendingRecording() (info interface{}, exists bool)
	ClearPendingRecording()

	AddPendingDial(childChannelUUID string, info interface{})
	GetPendingDial(childChannelUUID string) (info interface{}, exists bool)
	RemovePendingDial(childChannelUUID string)
	GetAllPendingDials() map[string]interface{}

	SendNextElements(elements []CallControlElement) error
	GetNextElementsChannel() <-chan []CallControlElement

	// Conference related context methods
	EnterConference(confSID, confName, callbackURL, callbackMethod string)
	LeaveConference()
	IsInConference() bool
	GetCurrentConferenceSID() (string, bool)
	GetCurrentConferenceName() (string, bool)
	GetCurrentConferenceCallbackURL() (string, bool)
	GetCurrentConferenceCallbackMethod() (string, bool)
	SetCurrentConferenceParticipantSID(participantSID string)
	GetCurrentConferenceParticipantSID() (string, bool)
}

// EslConnectionExecutor defines the interface for executing commands on an ESL connection.
type EslConnectionExecutor interface {
	Execute(command string, args ...string) (event string, err error)
	ExecuteSofia(command string, args ...string) (event string, err error) // Example for sofia api commands
	SendMsg(args map[string]string) (event string, err error)             // For more complex commands like play_and_get_digits
	GetVar(varName string) (string, error)
	Answer() (string, error)
	Hangup(reason string) (string, error)
	RecordSession(filePath string, maxDurationSec uint32, silenceThreshold uint, silenceHits uint) (string, error) // For RecordElement

	// Originate attempts to create a new outbound call leg.
	// dialString: The target to dial (e.g., "user/1000", "sofia/gateway/mygw/12345").
	// vars: A map of channel variables to set on the new call leg.
	// Returns the UUID of the newly created channel if successful, or an error.
	Originate(dialString string, vars map[string]string) (newChannelUUID string, err error) // For DialElement

	// PlayAndGetDigits plays a file and collects digits.
	// Returns collected digits and an error if any. terminationKey might also be relevant.
	PlayAndGetDigits(minDigits, maxDigits int, maxAttempts int, timeout uint32, terminators string, audioFile string, invalidAudioFile string, varName string) (digits string, err error) // For GatherElement

	// Add other specific ESL commands if they have unique signatures not covered by Execute/SendMsg
	// For now, elements will use Execute or SendMsg.
}

// CallServicerForESL defines methods the ESL/CallControl layer needs from CallService.
// This helps break import cycles.
type CallServicerForESL interface {
	// Call related methods
	UpdateCallStatus(ctx MinimalCallContext, status string, hangupCause string) error
	// Note: UpdateCallRecording was specified in a previous prompt for CallServicerForESL,
	// but CreateRecording (below) is what was implemented in CallService and seems more appropriate
	// for the ESL layer to call after a recording is finished.
	// If UpdateCallRecording is also needed for other purposes (e.g. updating path after move), it can be kept.
	// For now, focusing on CreateRecording as per the latest service implementation.
	// UpdateCallRecording(ctx MinimalCallContext, recordingPath string, durationSec int, format string) error

	// Recording related methods
	CreateRecording(ctx MinimalCallContext, callSid *string, recordingSid, filePath string, duration uint32, format string, sizeBytes int64) error

	// Conference related methods (embedding ConferenceService interface)
	// This requires importing the services package.
	// "github.com/user/agbaravoip_golang/internal/services" - this would create import cycle domain -> services -> domain
	// So, we must list methods explicitly or use a different approach for interface segregation.
	// For now, listing explicitly to avoid import cycle with services package.
	GetConferenceBySID(ctx context.Context, sid string) (*Conference, error)
	GetConferenceByName(ctx context.Context, accountSid, name string) (*Conference, error)
	CreateConference(ctx context.Context, accountSid, name, sid string) (*Conference, error)
	GetOrCreateConference(ctx context.Context, accountSid, name string) (*Conference, error)
	UpdateConferenceStatus(ctx context.Context, sid string, status ConferenceStatus) error
	EndConference(ctx context.Context, sid string, endTime time.Time) error
	AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*ConferenceParticipant, error)
	GetParticipant(ctx context.Context, pSid string) (*ConferenceParticipant, error)
	GetParticipantByCallSID(ctx context.Context, callSid string) (*ConferenceParticipant, error)
	UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error
	UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error
	RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error
	ListParticipants(ctx context.Context, confSid string) ([]*ConferenceParticipant, error)

	// Add CreateCall if Dial verb needs it through this interface in the future
	// Example: CreateSubsequentCall(ctx MinimalCallContext, params CreateCallParams) (newCallUUID string, err error)

	// SMSService methods
	SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*SMSMessage, error)
	GetSMSBySID(ctx context.Context, sid string) (*SMSMessage, error)
	UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error
	RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*SMSMessage, error)

	// Application service methods needed by handlers using CallServicerForESL
	GetApplicationByIncomingDID(ctx context.Context, did string) (*Application, error)

	// Account service methods (for auth, etc.)
	ValidateCredentials(ctx context.Context, accountSid string, plainToken string) (*Account, error)
	GetAccountBySID(ctx context.Context, sid string) (*Account, error) // Might be needed for other auth/user purposes
}
