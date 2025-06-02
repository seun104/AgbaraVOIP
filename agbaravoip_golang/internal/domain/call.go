package domain

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CallStatus defines the status of a call.
type CallStatus string

const (
	CallStatusQueued         CallStatus = "queued"      
	CallStatusInitiated      CallStatus = "initiated"    
	CallStatusRinging        CallStatus = "ringing"      
	CallStatusInProgress     CallStatus = "in-progress"   
	CallStatusInProgressXML  CallStatus = "in-progress-xml" 
	CallStatusCompleted      CallStatus = "completed"    
	CallStatusFailed         CallStatus = "failed"       
	CallStatusFailedXML      CallStatus = "failed-xml"   
	CallStatusBusy           CallStatus = "busy"
	CallStatusNoAnswer       CallStatus = "no-answer"
	CallStatusCanceled       CallStatus = "canceled"     
)

// CallDirection defines the direction of a call.
type CallDirection string

const (
	CallDirectionInbound      CallDirection = "inbound"
	CallDirectionOutboundAPI  CallDirection = "outbound-api"  
	CallDirectionOutboundDial CallDirection = "outbound-dial" 
)

// Call represents a voice call in the system.
type Call struct {
	ID        uint   `gorm:"primaryKey"`
	SID       string `gorm:"type:varchar(64);uniqueIndex;not null"`
	AccountSID string `gorm:"type:varchar(64);index;not null"`
	ApplicationSID *string `gorm:"type:varchar(64);index;null"` 

	FromNum string `gorm:"type:varchar(100)"`
	ToNum   string `gorm:"type:varchar(100)"`
	
	AnswerURL string `gorm:"type:text"`

	Status    CallStatus    `gorm:"type:call_status;not null"` 
	Direction CallDirection `gorm:"type:call_direction;not null"`

	DurationSeconds int     `gorm:"default:0"`
	Price           float64 `gorm:"type:numeric(10,5);default:0.0"`

	AnsweredBy     *string  `gorm:"type:varchar(100)"` 
	TimeoutSeconds *int     `gorm:"default:null"` 

	HangupCause   *string `gorm:"type:varchar(100)"` 
	ForwardedFrom *string `gorm:"type:varchar(100)"` 

	StartTime  *time.Time     `gorm:"default:null;index"` 
	AnswerTime *time.Time     `gorm:"default:null"`
	EndTime    *time.Time     `gorm:"default:null"`
	
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate is a GORM hook.
func (call *Call) BeforeCreate(tx *gorm.DB) (err error) {
	if call.SID == "" {
		call.SID = "CA" + uuid.NewString()
	}
	if call.Status == "" { 
		call.Status = CallStatusQueued
	}
	// It's generally better for the service layer to set StartTime explicitly.
	// This hook can serve as a fallback if necessary.
	// if call.StartTime == nil { 
	// 	 now := time.Now().UTC() // Simplified comment
	//   call.StartTime = &now
	// }
	return
}
