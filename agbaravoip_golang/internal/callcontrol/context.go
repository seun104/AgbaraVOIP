package callcontrol

import (
	"errors" // For SendNextElements error
	"sync"
	"time" // For SendNextElements timeout

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/sirupsen/logrus"
)

// PendingRecordInfo holds details about an active recording.
type PendingRecordInfo struct {
	OriginalElement  *domain.RecordElement
	ExpectedFilePath string
}

// PendingDialInfo holds details about a pending dial attempt.
type PendingDialInfo struct {
	OriginalElement     *domain.DialElement
	ParentAgbaraCallSID string
}

// CallContext holds all necessary information and state for a single call leg
// being controlled by the application via Freeswitch Outbound ESL.
type CallContext struct {
	uuid              string
	accountSid        string
	applicationSid    string
	answerURL         string
	variables         map[string]string
	logger            *logrus.Entry
	eslConnection     domain.EslConnectionExecutor
	hangupInitiated   bool
	mutex             sync.Mutex // General mutex for simple fields like hangupInitiated

	pendingRecording    *PendingRecordInfo
	pendingRecordingMutex sync.RWMutex // Mutex for pendingRecording field

	pendingDials        map[string]*PendingDialInfo // map[childChannelUUID]*PendingDialInfo
	pendingDialsMutex   sync.RWMutex                // Mutex for pendingDials map

	nextElementsChannel chan []domain.CallControlElement // Buffered channel for async XML

	// Conference related fields
	currentConferenceSID            *string
	currentConferenceName           *string
	currentConferenceCallbackURL    *string
	currentConferenceCallbackMethod *string
	currentConferenceParticipantSID *string // SID of this call leg as a participant
	confMutex                       sync.RWMutex // For conference fields
	hangupChan                      chan struct{} // For signaling hangup internally
}

// NewCallContext creates a new CallContext.
// The eslConnection parameter should be an object that satisfies domain.EslConnectionExecutor.
// The 'uuid' parameter is the Agbara Call SID. The Freeswitch specific UUID is expected to be in vars["uuid"].
func NewCallContext(uuid, accountSid, appSid, answerURL string, vars map[string]string, eslConn domain.EslConnectionExecutor, baseLogger *logrus.Logger) *CallContext {
	logger := baseLogger.WithFields(logrus.Fields{
		"agbara_call_sid": uuid, // Use agbara_call_sid for clarity in logs
		"account_sid":    accountSid,
		// "freeswitch_uuid": vars["uuid"], // Optionally log FS UUID if always present and useful
	})
	logger.Info("Creating new call context")

	// Ensure vars is not nil, as it's used by GetVariable and potentially GetFreeswitchUUID
	if vars == nil {
		vars = make(map[string]string)
	}
	// If agbara_call_sid is not already in vars, add it.
	if _, exists := vars["agbara_call_sid"]; !exists {
		vars["agbara_call_sid"] = uuid
	}


	return &CallContext{
		uuid:              uuid, // This 'uuid' field stores the Agbara Call SID
		accountSid:        accountSid,
		applicationSid:    appSid,
		answerURL:         answerURL,
		variables:         vars,
		logger:            logger,
		eslConnection:     eslConn,
		hangupInitiated:   false,
		pendingDials:      make(map[string]*PendingDialInfo),
		nextElementsChannel: make(chan []domain.CallControlElement, 1),
		hangupChan:        make(chan struct{}),
		// Mutexes (pendingDialsMutex, pendingRecordingMutex, confMutex) are zero-value ready.
	}
}

// Log returns the logger associated with this call context.
func (cc *CallContext) Log() *logrus.Entry {
	return cc.logger
}

// GetUuid returns the Freeswitch channel UUID for this call.
func (cc *CallContext) GetUuid() string {
	return cc.uuid
}

// GetAccountSid returns the Account SID associated with this call.
func (cc *CallContext) GetAccountSid() string {
	return cc.accountSid
}

// GetApplicationSid returns the Application SID that is handling this call.
func (cc *CallContext) GetApplicationSid() string {
	return cc.applicationSid
}

// GetAnswerURL returns the initial Answer URL for this call.
func (cc *CallContext) GetAnswerURL() string {
	return cc.answerURL
}

// GetEslConnection returns the ESL connection executor.
// This is not part of MinimalCallContext but useful within callcontrol package.
func (cc *CallContext) GetEslConnection() domain.EslConnectionExecutor {
    return cc.eslConnection
}

// IsHangupInitiated checks if a hangup has been signaled for this call.
func (cc *CallContext) IsHangupInitiated() bool {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()
	return cc.hangupInitiated
}

// SetHangupInitiated marks that a hangup signal has been received.
func (cc *CallContext) SetHangupInitiated() {
	cc.mutex.Lock()
	// Unlock before potential panic on close if already closed, though current logic aims for single closer.
	// defer cc.mutex.Unlock()
	alreadyInitiated := cc.hangupInitiated
	if !alreadyInitiated {
		cc.hangupInitiated = true
		// Close channel before logging to ensure signal is sent first.
		// This also prevents recursive lock if logger somehow calls SetHangupInitiated.
		// Ensure hangupChan is not nil (it's initialized in NewCallContext).
		if cc.hangupChan != nil {
			// Check if channel is already closed to prevent panic
			// This is a bit tricky. A select with a default is one way for non-blocking check,
			// but here we are in a critical section. Simpler to rely on single-closer principle for now.
			// If this method can be called concurrently by routines that might race to close,
			// a sync.Once around close(cc.hangupChan) would be safer.
			// For now, assume SetHangupInitiated's critical section (mutex) serializes calls.
			close(cc.hangupChan)
		}
		cc.logger.Info("Hangup signal received, marking call context as hangup initiated and closing hangupChan.")
	}
	cc.mutex.Unlock() // Unlock after modifications
}

// Ensure CallContext implements domain.MinimalCallContext
var _ domain.MinimalCallContext = &CallContext{}

// --- Pending Recording Methods ---

func (cc *CallContext) SetPendingRecording(info interface{}) {
	cc.pendingRecordingMutex.Lock()
	defer cc.pendingRecordingMutex.Unlock()
	if pRecInfo, ok := info.(PendingRecordInfo); ok {
		cc.pendingRecording = &pRecInfo
	} else if pRecInfoPtr, ok := info.(*PendingRecordInfo); ok {
		cc.pendingRecording = pRecInfoPtr
	} else {
		cc.logger.Error("SetPendingRecording received info of incorrect type")
		cc.pendingRecording = nil // Or handle error appropriately
	}
}

func (cc *CallContext) GetPendingRecording() (interface{}, bool) {
	cc.pendingRecordingMutex.RLock()
	defer cc.pendingRecordingMutex.RUnlock()
	if cc.pendingRecording == nil {
		return nil, false
	}
	return cc.pendingRecording, true // Return pointer to struct, which is an interface{}
}

func (cc *CallContext) ClearPendingRecording() {
	cc.pendingRecordingMutex.Lock()
	defer cc.pendingRecordingMutex.Unlock()
	cc.pendingRecording = nil
}

// --- Pending Dial Methods ---

func (cc *CallContext) AddPendingDial(childChannelUUID string, info interface{}) {
	cc.pendingDialsMutex.Lock()
	defer cc.pendingDialsMutex.Unlock()
	if pDialInfo, ok := info.(PendingDialInfo); ok {
		cc.pendingDials[childChannelUUID] = &pDialInfo
	} else if pDialInfoPtr, ok := info.(*PendingDialInfo); ok {
		cc.pendingDials[childChannelUUID] = pDialInfoPtr
	} else {
		cc.logger.Errorf("AddPendingDial received info of incorrect type for UUID %s", childChannelUUID)
	}
}

func (cc *CallContext) GetPendingDial(childChannelUUID string) (interface{}, bool) {
	cc.pendingDialsMutex.RLock()
	defer cc.pendingDialsMutex.RUnlock()
	info, exists := cc.pendingDials[childChannelUUID]
	if !exists || info == nil {
		return nil, false
	}
	return info, true // Return pointer to struct, which is an interface{}
}

func (cc *CallContext) RemovePendingDial(childChannelUUID string) {
	cc.pendingDialsMutex.Lock()
	defer cc.pendingDialsMutex.Unlock()
	delete(cc.pendingDials, childChannelUUID)
}

func (cc *CallContext) GetAllPendingDials() map[string]interface{} {
	cc.pendingDialsMutex.RLock()
	defer cc.pendingDialsMutex.RUnlock()

	// Create a new map to return to avoid issues with concurrent access to the map itself
	// if the caller iterates over it while another goroutine modifies cc.pendingDials.
	// The values are pointers, so modifications to the PendingDialInfo structs themselves
	// would still require care if done concurrently by multiple goroutines.
	resultMap := make(map[string]interface{}, len(cc.pendingDials))
	for k, v := range cc.pendingDials {
		resultMap[k] = v // v is already *PendingDialInfo, which satisfies interface{}
	}
	return resultMap
}

// --- Async XML Element Channel Methods ---

func (cc *CallContext) SendNextElements(elements []domain.CallControlElement) error {
	select {
	case cc.nextElementsChannel <- elements:
		cc.logger.Debugf("Sent %d elements to nextElementsChannel", len(elements))
		return nil
	case <-time.After(100 * time.Millisecond): // Non-blocking with a small timeout
		cc.logger.Warn("Failed to send next elements to channel: timeout/buffer full")
		return errors.New("SendNextElements: channel send timeout or buffer full")
	}
}

func (cc *CallContext) GetNextElementsChannel() <-chan []domain.CallControlElement {
	return cc.nextElementsChannel
}

// HangupChan returns a channel that is closed when hangup is initiated.
func (cc *CallContext) HangupChan() <-chan struct{} {
	return cc.hangupChan
}

// GetFreeswitchUUID retrieves the Freeswitch Channel UUID from variables.
// It's a helper method for CallContext users, not part of MinimalCallContext by default.
func (cc *CallContext) GetFreeswitchUUID() string {
	if fsUUID, ok := cc.variables["uuid"]; ok { // FS sets 'uuid' var with its channel UUID
		return fsUUID
	}
	cc.logger.Warn("Freeswitch UUID (vars[\"uuid\"]) not found in CallContext variables.")
	return "" // Or handle error appropriately
}

// GetCallSID returns the Agbara Call SID (which is cc.uuid) as a pointer.
// Useful for DB interactions where a *string might be needed for nullable fields,
// though for this specific SID, it should always be present.
func (cc *CallContext) GetCallSID() *string {
	if cc.uuid == "" {
		return nil // Should not happen for a valid CallContext
	}
	sid := cc.uuid
	return &sid
}


// --- Conference Context Methods ---

func (cc *CallContext) EnterConference(confSID, confName, callbackURL, callbackMethod string) {
	cc.confMutex.Lock()
	defer cc.confMutex.Unlock()
	cc.currentConferenceSID = &confSID
	cc.currentConferenceName = &confName
	if callbackURL != "" {
		cc.currentConferenceCallbackURL = &callbackURL
		cc.currentConferenceCallbackMethod = &callbackMethod
	} else {
		cc.currentConferenceCallbackURL = nil
		cc.currentConferenceCallbackMethod = nil
	}
}

func (cc *CallContext) LeaveConference() {
	cc.confMutex.Lock()
	defer cc.confMutex.Unlock()
	cc.currentConferenceSID = nil
	cc.currentConferenceName = nil
	cc.currentConferenceCallbackURL = nil
	cc.currentConferenceCallbackMethod = nil
	cc.currentConferenceParticipantSID = nil
}

func (cc *CallContext) IsInConference() bool {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	return cc.currentConferenceSID != nil && *cc.currentConferenceSID != ""
}

func (cc *CallContext) GetCurrentConferenceSID() (string, bool) {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	if cc.currentConferenceSID != nil {
		return *cc.currentConferenceSID, true
	}
	return "", false
}

func (cc *CallContext) GetCurrentConferenceName() (string, bool) {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	if cc.currentConferenceName != nil {
		return *cc.currentConferenceName, true
	}
	return "", false
}

func (cc *CallContext) GetCurrentConferenceCallbackURL() (string, bool) {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	if cc.currentConferenceCallbackURL != nil {
		return *cc.currentConferenceCallbackURL, true
	}
	return "", false
}

func (cc *CallContext) GetCurrentConferenceCallbackMethod() (string, bool) {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	if cc.currentConferenceCallbackMethod != nil {
		return *cc.currentConferenceCallbackMethod, true
	}
	return "", false
}

func (cc *CallContext) SetCurrentConferenceParticipantSID(participantSID string) {
	cc.confMutex.Lock()
	defer cc.confMutex.Unlock()
	if participantSID != "" {
		cc.currentConferenceParticipantSID = &participantSID
	} else {
		cc.currentConferenceParticipantSID = nil
	}
}

func (cc *CallContext) GetCurrentConferenceParticipantSID() (string, bool) {
	cc.confMutex.RLock()
	defer cc.confMutex.RUnlock()
	if cc.currentConferenceParticipantSID != nil {
		return *cc.currentConferenceParticipantSID, true
	}
	return "", false
}
