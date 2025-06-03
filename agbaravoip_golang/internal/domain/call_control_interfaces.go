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
	UpdateCallStatus(ctx MinimalCallContext, status string, hangupCause string) error
	UpdateCallRecording(ctx MinimalCallContext, recordingPath string, durationSec int, format string) error
	// CreateRecording saves metadata about a completed recording.
	// callSid is the SID of the call associated with the recording (can be nil if it's a conference recording).
	// recordingSid is the new unique SID for this recording.
	// filePath is the path/URL to the recording file.
	// duration is the length of the recording in seconds.
	// format is the file format (e.g., "wav", "mp3").
	// sizeBytes is the size of the recording file in bytes.
	CreateRecording(ctx MinimalCallContext, callSid *string, recordingSid, filePath string, duration uint32, format string, sizeBytes int64) error
	// Add CreateCall if Dial verb needs it through this interface in the future
	// Example: CreateSubsequentCall(ctx MinimalCallContext, params CreateCallParams) (newCallUUID string, err error)
}
