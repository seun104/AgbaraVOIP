package esl

import "github.com/user/agbaravoip_golang/internal/domain"

// CallServicerForESL defines the subset of ICallService needed by FSOutboundServer.
// This helps break the import cycle: esl -> services.
type CallServicerForESL interface {
	UpdateCallStatus(agbaraCallSid string, status domain.CallStatus, hangupCause string, durationSeconds int) (*domain.Call, error)
	// Add other methods here if FSOutboundServer needs more from CallService in the future
	// For example, GetCallBySID if it needs to fetch full call details not present in CallContext.
}

