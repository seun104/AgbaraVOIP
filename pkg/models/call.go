package models

import (
	"time"
)

// CallStatus (already defined, ensure it's complete)
type CallStatus string
const (
	CallStatusQueued     CallStatus = "queued"
	CallStatusInitiating CallStatus = "initiating" // New status for when FS origination starts
	CallStatusRinging    CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusCanceled   CallStatus = "canceled"
)

// Call model update
type Call struct {
	Sid              string     `json:"sid"`
	AccountSid       string     `json:"accountSid"`
	CallerId         string     `json:"callerId,omitempty"`
	CallTo           string     `json:"callTo"`
	AnswerUrl        string     `json:"answerUrl,omitempty"` // Used if ApplicationSid is not provided or App has no VoiceUrl
	ApplicationSid   string     `json:"applicationSid,omitempty"` // Reference to an Application for call handling logic
	Status           CallStatus `json:"status"`
	Timeout          string     `json:"timeout,omitempty"`      
	Direction        string     `json:"direction,omitempty"`
	Duration         int        `json:"duration"`             
	Price            float64    `json:"price"`                
	StartTime        time.Time  `json:"startTime"`
	EndTime          time.Time  `json:"endTime,omitempty"`    
	DateCreated      time.Time  `json:"dateCreated"`
	DateUpdated      time.Time  `json:"dateUpdated"`
	AnsweredBy       string     `json:"answeredBy,omitempty"`
	FreeswitchCallID string     `json:"freeswitchCallId,omitempty"` // To store FS Channel UUID or Job UUID
}

// CallRequest model update
type CallRequest struct {
	From                 string `json:"from,omitempty"` // Caller ID. Can be "Name <Number>" or just Number.
	To                   string `json:"to" binding:"required"`
	ApplicationSid       string `json:"applicationSid,omitempty"` 
	AnswerUrl            string `json:"answerUrl,omitempty"` // Fallback/alternative if ApplicationSid is not used. One of them should provide call handling instructions.
	Method               string `json:"method,omitempty"`           // For AnswerUrl if it's a direct webhook Agbara-Go needs to call (less likely with FS originate)
	FallbackUrl          string `json:"fallbackUrl,omitempty"`      // Similar to above
	FallbackMethod       string `json:"fallbackMethod,omitempty"`   // Similar to above
	StatusCallbackUrl    string `json:"statusCallbackUrl,omitempty"` // URL to send call status events
	StatusCallbackMethod string `json:"statusCallbackMethod,omitempty"`
	SendDigits           string `json:"sendDigits,omitempty"`       // Digits to send after call connects
	TimeLimit            string `json:"timeLimit,omitempty"`        // Max duration (maps to Call.Timeout string)
	HangupOnRing         string `json:"hangupOnRing,omitempty"`     // e.g., "true", "false", or number of rings
}
// Ensure one of ApplicationSid or AnswerUrl is effectively required by API handler logic if not by binding tags.
// The C# had: AnswerUrl [Required(ErrorMessage = "Answer Url cannot be empty"), DataType(DataType.Url,ErrorMessage="Url Not Properly Formatted")]
// This implies AnswerUrl was primary. If ApplicationSid is used, its VoiceUrl becomes the effective "AnswerUrl" for FreeSWITCH.
// We can adjust binding later in API handlers if needed. For now, model reflects both.

// CallResponse (already defined, no changes needed for this step)
type CallResponse struct {
	Message      string `json:"message"`
	IsSuccessful bool   `json:"isSuccessful"`
	CallSid      string `json:"callSid,omitempty"`
}

// Other DTOs (CallPlayRequest, etc.) remain as they were.
// CallPlayRequest model
type CallPlayRequest struct {
	PlayUrl string `json:"playUrl" binding:"required"`
	Loop    string `json:"loop,omitempty"` 
	Legs    string `json:"legs,omitempty"` 
}
// CallPlayResponse model
type CallPlayResponse struct {
	CallSid string `json:"callSid"`
	Message string `json:"message"` 
}
// CallRecordRequest model
type CallRecordRequest struct {}
// CallRecordResponse model
type CallRecordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
// CallSpeakRequest model
type CallSpeakRequest struct {
	Text string `json:"text" binding:"required"`
	Loop string `json:"loop,omitempty"`
}
// CallSpeakResponse model
type CallSpeakResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
// CallDigitRequest model
type CallDigitRequest struct {
	Digits string `json:"digits" binding:"required"`
}
// CallDigitResponse model
type CallDigitResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
