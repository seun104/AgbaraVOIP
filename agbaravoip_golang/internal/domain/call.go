package domain

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CallStatus defines the lifecycle status of a call.
type CallStatus string

const (
	CallStatusQueued     CallStatus = "queued"
	CallStatusInitiated  CallStatus = "initiated" // Origination sent to Freeswitch
	CallStatusRinging    CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusCanceled   CallStatus = "canceled"  // Canceled before completion
)

// CallDirection indicates whether the call is inbound or outbound.
type CallDirection string

const (
	CallDirectionInbound      CallDirection = "inbound"
	CallDirectionOutboundAPI  CallDirection = "outbound-api"  // Initiated via API
	CallDirectionOutboundDial CallDirection = "outbound-dial" // Initiated via Dial command (e.g. click-to-call)
)

// Call represents a voice call record in the system.
type Call struct {
	ID        uint   `gorm:"primaryKey"`
	SID       string `gorm:"type:varchar(64);uniqueIndex;not null"` // Publicly visible SID, e.g., CAxxxxxxxx
	AccountSID string `gorm:"type:varchar(64);index;not null"`    // Belongs to an Account SID
	ApplicationSID *string `gorm:"type:varchar(64);index"`          // Optional Application SID if call is tied to an app

	FromNum string `gorm:"type:varchar(100)"` // Originating number
	ToNum   string `gorm:"type:varchar(100)"` // Destination number
	
	AnswerURL string `gorm:"type:text"` // URL for AgbaraXML or call control logic

	Status    CallStatus    `gorm:"type:call_status;not null;default:'queued'"` // Using custom ENUM type from DB
	Direction CallDirection `gorm:"type:call_direction;not null"`           // Using custom ENUM type from DB

	DurationSeconds int     `gorm:"default:0"`         // Billable duration in seconds
	Price           float64 `gorm:"type:numeric(10,5);default:0.0"` // Cost of the call

	AnsweredBy    *string `gorm:"type:varchar(100)"`      // e.g., human, machine, fax
	TimeoutSeconds *int    `gorm:"default:null"`           // Call timeout specified at origination
	HangupCause   *string `gorm:"type:varchar(100)"`      // Freeswitch hangup cause
	ForwardedFrom *string `gorm:"type:varchar(100)"`      // If the call was forwarded from another number/SIP URI

	StartTime  *time.Time     `gorm:"index"` // Time of call initiation attempt
	AnswerTime *time.Time     // Time the call was answered
	EndTime    *time.Time     // Time the call ended
	
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"` // For soft deletes
}

// BeforeCreate is a GORM hook that runs before a new record is created.
func (call *Call) BeforeCreate(tx *gorm.DB) (err error) {
	if call.SID == "" {
		call.SID = "CA" + uuid.NewString()
	}
	if call.Status == "" { // Default status if not set
		call.Status = CallStatusQueued
	}
	// Direction must be set explicitly by the service creating the call.
	return
}
