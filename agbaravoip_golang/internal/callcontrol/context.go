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
}

// NewCallContext creates a new CallContext.
// The eslConnection parameter should be an object that satisfies domain.EslConnectionExecutor.
func NewCallContext(uuid, accountSid, appSid, answerURL string, vars map[string]string, eslConn domain.EslConnectionExecutor, baseLogger *logrus.Logger) *CallContext {
	logger := baseLogger.WithFields(logrus.Fields{
		"call_uuid": uuid,
		"account_sid": accountSid,
	})
	logger.Info("Creating new call context")

	return &CallContext{
		uuid:              uuid,
		accountSid:        accountSid,
		applicationSid:    appSid,
		answerURL:         answerURL,
		variables:         vars,
		logger:            logger,
		eslConnection:     eslConn,
		hangupInitiated:   false,
		pendingDials:      make(map[string]*PendingDialInfo),
		// pendingDialsMutex is value type, initialized implicitly
		// pendingRecordingMutex is value type, initialized implicitly
		nextElementsChannel: make(chan []domain.CallControlElement, 1), // Buffered channel of size 1
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
	defer cc.mutex.Unlock()
	if !cc.hangupInitiated {
		cc.hangupInitiated = true
		cc.logger.Info("Hangup signal received, marking call context as hangup initiated.")
	}
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
