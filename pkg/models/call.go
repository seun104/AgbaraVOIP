package models

import (
	"time"
)

// CallStatus represents the status of a call.
type CallStatus string

const (
	CallStatusQueued     CallStatus = "queued"
	CallStatusRinging    CallStatus = "ringing"
	CallStatusInProgress CallStatus = "in-progress"
	CallStatusCompleted  CallStatus = "completed"
	CallStatusFailed     CallStatus = "failed"
	CallStatusBusy       CallStatus = "busy"
	CallStatusNoAnswer   CallStatus = "no-answer"
	CallStatusCanceled   CallStatus = "canceled" // If a queued call is cancelled
)

// Call model, corresponds to Domain/Objects/Call.cs
type Call struct {
	Sid          string     `json:"sid"`
	AccountSid   string     `json:"accountSid"`
	CallerId     string     `json:"callerId,omitempty"` // From field in CallRequest
	CallTo       string     `json:"callTo"`           // To field in CallRequest
	AnswerUrl    string     `json:"answerUrl"`
	Status       CallStatus `json:"status"`
	Timeout      string     `json:"timeout,omitempty"`      // String to match C#; service layer converts to int for DB
	Direction    string     `json:"direction,omitempty"`
	Duration     int        `json:"duration"`             // In seconds
	Price        float64    `json:"price"`                // Assuming float64 for price
	StartTime    time.Time  `json:"startTime"`
	EndTime      time.Time  `json:"endTime,omitempty"`    // omitempty if not set
	DateCreated  time.Time  `json:"dateCreated"`
	DateUpdated  time.Time  `json:"dateUpdated"`
	AnsweredBy   string     `json:"answeredBy,omitempty"`
}

// CallRequest model for POST /Accounts/{AccountSid}/Calls/Call
// Based on src/AgbaraAPI/Model/Call/CallRequest.cs
type CallRequest struct {
	// AccountSid is set from path/auth context
	From                 string `json:"from,omitempty"` // Maps to Call.CallerId
	To                   string `json:"to" binding:"required"`
	ApplicationSid       string `json:"applicationSid,omitempty"` // SID of an Application to handle the call
	AnswerUrl            string `json:"answerUrl" binding:"required_without=ApplicationSid"` // Required if AppSID not given
	Method               string `json:"method,omitempty"`           // "GET" or "POST" for AnswerUrl
	FallbackUrl          string `json:"fallbackUrl,omitempty"`
	FallbackMethod       string `json:"fallbackMethod,omitempty"`
	StatusCallbackUrl    string `json:"statusCallbackUrl,omitempty"`
	StatusCallbackMethod string `json:"statusCallbackMethod,omitempty"`
	SendDigits           string `json:"sendDigits,omitempty"`
	TimeLimit            string `json:"timeLimit,omitempty"`    // Max duration of the call
	HangupOnRing         string `json:"hangupOnRing,omitempty"` // "true" or "false", or int for number of rings
}

// CallResponse model (simple generic response, C# has this)
type CallResponse struct {
	Message      string `json:"message"`
	IsSuccessful bool   `json:"isSuccessful"`
	CallSid      string `json:"callSid,omitempty"`
}

// --- Specific Action Request/Response DTOs (defined in Phase 1, Step 2, good to have them here) ---
// These were for deferred Call operations (Play, Record, Speak, Digit)

// CallPlayRequest model
type CallPlayRequest struct {
	// AccountSid, CallSid from path
	PlayUrl string `json:"playUrl" binding:"required"`
	Loop    string `json:"loop,omitempty"` // e.g., "1", "10", "0" for infinite
	Legs    string `json:"legs,omitempty"` // e.g., "aleg", "bleg", "both"
}

// CallPlayResponse model
type CallPlayResponse struct {
	CallSid string `json:"callSid"`
	Message string `json:"message"` // Corrected from "Mesage"
}

// StopCallPlayRequest model (No body, path params are enough)
// type StopCallPlayRequest struct {}

// StopCallPlayResponse model
type StopCallPlayResponse struct {
	CallSid string `json:"callSid"`
	Message string `json:"message"`
}

// CallRecordRequest model
type CallRecordRequest struct {
	// AccountSid, CallSid from path
	// Potentially other params like FileName, Format, MaxLength, etc.
}

// CallRecordResponse model
type CallRecordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	// RecordingUrl string `json:"recordingUrl,omitempty"`
}

// StopCallRecordRequest model (No body)
// type StopCallRecordRequest struct {}

// StopCallRecordResponse model
type StopCallRecordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CallSpeakRequest model
type CallSpeakRequest struct {
	// AccountSid, CallSid from path
	Text string `json:"text" binding:"required"`
	Loop string `json:"loop,omitempty"`
	// Voice, Language etc.
}

// CallSpeakResponse model
type CallSpeakResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CallDigitRequest model (for sending DTMF)
type CallDigitRequest struct {
	// AccountSid, CallSid from path
	Digits string `json:"digits" binding:"required"`
	// ToneDuration, etc.
}

// CallDigitResponse model
type CallDigitResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
