package callcontrol

import (
	"sync"

	"github.com/user/agbaravoip_golang/internal/domain" // Corrected import path
	"github.com/sirupsen/logrus"
)

// CallContext holds all necessary information and state for a single call leg
// being controlled by the application via Freeswitch Outbound ESL.
type CallContext struct {
	uuid              string
	accountSid        string
	applicationSid    string
	answerURL         string
	variables         map[string]string
	logger            *logrus.Entry
	eslConnection     domain.EslConnectionExecutor // Changed to interface
	hangupInitiated   bool
	mutex             sync.Mutex
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
