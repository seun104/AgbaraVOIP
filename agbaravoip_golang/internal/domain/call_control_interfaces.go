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
}

// EslConnectionExecutor defines the interface for executing commands on an ESL connection.
type EslConnectionExecutor interface {
	Execute(command string, args ...string) (event string, err error)
	ExecuteSofia(command string, args ...string) (event string, err error) // Example for sofia api commands
	SendMsg(args map[string]string) (event string, err error)             // For more complex commands like play_and_get_digits
	GetVar(varName string) (string, error)
	Answer() (string, error)
	Hangup(reason string) (string, error)
	// Add other specific ESL commands if they have unique signatures not covered by Execute/SendMsg
	// e.g. PlayAndGetDigits(params map[string]string) (digits string, err error)
	// For now, elements will use Execute or SendMsg.
}

// CallServicerForESL defines methods the ESL/CallControl layer needs from CallService.
// This helps break import cycles.
type CallServicerForESL interface {
	UpdateCallStatus(ctx MinimalCallContext, status string, hangupCause string) error
	UpdateCallRecording(ctx MinimalCallContext, recordingPath string, durationSec int, format string) error
	// Add CreateCall if Dial verb needs it through this interface in the future
	// Example: CreateSubsequentCall(ctx MinimalCallContext, params CreateCallParams) (newCallUUID string, err error)
}
