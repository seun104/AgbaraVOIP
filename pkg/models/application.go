package models

import (
	"strings"
	"time"
)

// HTTPMethod represents common HTTP methods.
type HTTPMethod string

const (
	HTTPMethodGET  HTTPMethod = "GET"
	HTTPMethodPost HTTPMethod = "POST"
)

// Application model, corresponds to Domain/Objects/Application.cs
type Application struct {
	Sid                   string     `json:"sid"`
	AccountSid            string     `json:"accountSid"`
	FriendlyName          string     `json:"friendlyName"`
	VoiceUrl              string     `json:"voiceUrl,omitempty"`
	VoiceMethod           HTTPMethod `json:"voiceMethod,omitempty"` // Use HTTPMethod type
	VoiceFallbackUrl      string     `json:"voiceFallbackUrl,omitempty"`
	VoiceFallbackMethod   HTTPMethod `json:"voiceFallbackMethod,omitempty"`
	StatusCallback        string     `json:"statusCallback,omitempty"`
	StatusCallbackMethod  HTTPMethod `json:"statusCallbackMethod,omitempty"`
	SmsUrl                string     `json:"smsUrl,omitempty"`
	SmsMethod             HTTPMethod `json:"smsMethod,omitempty"`
	SmsFallbackUrl        string     `json:"smsFallbackUrl,omitempty"`
	SmsFallbackMethod     HTTPMethod `json:"smsFallbackMethod,omitempty"`
	SmsStatusCallback     string     `json:"smsStatusCallback,omitempty"`
	SmsStatusCallbackMethod HTTPMethod `json:"smsStatusCallbackMethod,omitempty"`
	HeartbeatUrl          string     `json:"heartbeatUrl,omitempty"`
	DateCreated           time.Time  `json:"dateCreated"`
	DateUpdated           time.Time  `json:"dateUpdated"`
}

// ApplicationRequest DTO for creating or updating an Application.
// For updates, fields not provided (e.g., empty string for string types)
// should ideally not overwrite existing values unless that's the intent.
// Service logic will need to handle partial updates carefully if this DTO is used for PATCH,
// or assume full replacement if used for PUT/POST-update.
type ApplicationRequest struct {
	FriendlyName          string     `json:"friendlyName" binding:"required"`
	VoiceUrl              string     `json:"voiceUrl,omitempty"`
	VoiceMethod           HTTPMethod `json:"voiceMethod,omitempty"`
	VoiceFallbackUrl      string     `json:"voiceFallbackUrl,omitempty"`
	VoiceFallbackMethod   HTTPMethod `json:"voiceFallbackMethod,omitempty"`
	StatusCallback        string     `json:"statusCallback,omitempty"`
	StatusCallbackMethod  HTTPMethod `json:"statusCallbackMethod,omitempty"`
	SmsUrl                string     `json:"smsUrl,omitempty"`
	SmsMethod             HTTPMethod `json:"smsMethod,omitempty"`
	SmsFallbackUrl        string     `json:"smsFallbackUrl,omitempty"`
	SmsFallbackMethod     HTTPMethod `json:"smsFallbackMethod,omitempty"`
	SmsStatusCallback     string     `json:"smsStatusCallback,omitempty"`
	SmsStatusCallbackMethod HTTPMethod `json:"smsStatusCallbackMethod,omitempty"`
	HeartbeatUrl          string     `json:"heartbeatUrl,omitempty"`
}

// ToAppModel converts ApplicationRequest DTO to Application model.
// AccountSid needs to be set separately.
func (ar *ApplicationRequest) ToAppModel() *Application {
	// Default HTTP methods if not provided or invalid, or keep them empty
	// and let DB default or validation handle it.
	// For now, ensure they are uppercase if provided.
	return &Application{
		FriendlyName:          ar.FriendlyName,
		VoiceUrl:              ar.VoiceUrl,
		VoiceMethod:           HTTPMethod(strings.ToUpper(string(ar.VoiceMethod))),
		VoiceFallbackUrl:      ar.VoiceFallbackUrl,
		VoiceFallbackMethod:   HTTPMethod(strings.ToUpper(string(ar.VoiceFallbackMethod))),
		StatusCallback:        ar.StatusCallback,
		StatusCallbackMethod:  HTTPMethod(strings.ToUpper(string(ar.StatusCallbackMethod))),
		SmsUrl:                ar.SmsUrl,
		SmsMethod:             HTTPMethod(strings.ToUpper(string(ar.SmsMethod))),
		SmsFallbackUrl:        ar.SmsFallbackUrl,
		SmsFallbackMethod:     HTTPMethod(strings.ToUpper(string(ar.SmsFallbackMethod))),
		SmsStatusCallback:     ar.SmsStatusCallback,
		SmsStatusCallbackMethod: HTTPMethod(strings.ToUpper(string(ar.SmsStatusCallbackMethod))),
		HeartbeatUrl:          ar.HeartbeatUrl,
	}
}

// ValidateHTTPMethod ensures the method is one of the allowed types, or empty.
// This can be used during binding or service logic.
func (h HTTPMethod) Validate() bool {
	s := strings.ToUpper(string(h))
	switch s {
	case "": // Allow empty (will not be set or will use DB default if any)
		return true
	case string(HTTPMethodGET):
		return true
	case string(HTTPMethodPost):
		return true
	default:
		return false
	}
}
