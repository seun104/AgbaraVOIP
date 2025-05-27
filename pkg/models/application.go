package models

import (
	"strings"
	"time"
)

// HTTPMethod (already defined)
type HTTPMethod string
const (
	HTTPMethodGET  HTTPMethod = "GET"
	HTTPMethodPost HTTPMethod = "POST"
)
func (h HTTPMethod) Validate() bool {
	s := strings.ToUpper(string(h))
	switch s {
	case "": return true
	case string(HTTPMethodGET): return true
	case string(HTTPMethodPost): return true
	default: return false
	}
}


// Application model (ensure comments reflect usage by FreeSWITCH for call control)
type Application struct {
	Sid                   string     `json:"sid"`
	AccountSid            string     `json:"accountSid"`
	FriendlyName          string     `json:"friendlyName"`
	// VoiceUrl: FreeSWITCH will make an HTTP request to this URL for call control instructions (TwiML).
	VoiceUrl              string     `json:"voiceUrl,omitempty"`
	VoiceMethod           HTTPMethod `json:"voiceMethod,omitempty"` 
	// VoiceFallbackUrl: If VoiceUrl request fails, FreeSWITCH may try this URL.
	VoiceFallbackUrl      string     `json:"voiceFallbackUrl,omitempty"`
	VoiceFallbackMethod   HTTPMethod `json:"voiceFallbackMethod,omitempty"`
	// StatusCallback: Agbara-Go (or FS directly) could send call progress events here.
	StatusCallback        string     `json:"statusCallback,omitempty"`
	StatusCallbackMethod  HTTPMethod `json:"statusCallbackMethod,omitempty"`
	// SmsUrl: URL for FreeSWITCH to request when an SMS is received for this application.
	SmsUrl                string     `json:"smsUrl,omitempty"`
	SmsMethod             HTTPMethod `json:"smsMethod,omitempty"`
	// SmsFallbackUrl: Fallback URL if SmsUrl request fails.
	SmsFallbackUrl        string     `json:"smsFallbackUrl,omitempty"`
	SmsFallbackMethod     HTTPMethod `json:"smsFallbackMethod,omitempty"`
	// SmsStatusCallback: URL for Agbara-Go to send SMS delivery status events.
	SmsStatusCallback     string     `json:"smsStatusCallback,omitempty"`
	SmsStatusCallbackMethod HTTPMethod `json:"smsStatusCallbackMethod,omitempty"`
	// HeartbeatUrl: URL for an external system to check the health of this application (if Agbara-Go needs to implement it).
	HeartbeatUrl          string     `json:"heartbeatUrl,omitempty"` 
	DateCreated           time.Time  `json:"dateCreated"`
	DateUpdated           time.Time  `json:"dateUpdated"`
}

// ApplicationRequest DTO - Add binding tags for URL validation
type ApplicationRequest struct {
	FriendlyName          string     `json:"friendlyName" binding:"required"`
	VoiceUrl              string     `json:"voiceUrl,omitempty" binding:"omitempty,url"`
	VoiceMethod           HTTPMethod `json:"voiceMethod,omitempty"` // Validation via HTTPMethod.Validate()
	VoiceFallbackUrl      string     `json:"voiceFallbackUrl,omitempty" binding:"omitempty,url"`
	VoiceFallbackMethod   HTTPMethod `json:"voiceFallbackMethod,omitempty"`
	StatusCallback        string     `json:"statusCallback,omitempty" binding:"omitempty,url"`
	StatusCallbackMethod  HTTPMethod `json:"statusCallbackMethod,omitempty"`
	SmsUrl                string     `json:"smsUrl,omitempty" binding:"omitempty,url"`
	SmsMethod             HTTPMethod `json:"smsMethod,omitempty"`
	SmsFallbackUrl        string     `json:"smsFallbackUrl,omitempty" binding:"omitempty,url"`
	SmsFallbackMethod     HTTPMethod `json:"smsFallbackMethod,omitempty"`
	SmsStatusCallback     string     `json:"smsStatusCallback,omitempty" binding:"omitempty,url"`
	SmsStatusCallbackMethod HTTPMethod `json:"smsStatusCallbackMethod,omitempty"`
	HeartbeatUrl          string     `json:"heartbeatUrl,omitempty" binding:"omitempty,url"`
}

// ToAppModel (already defined, ensure it correctly handles HTTPMethod conversion)
func (ar *ApplicationRequest) ToAppModel() *Application {
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
