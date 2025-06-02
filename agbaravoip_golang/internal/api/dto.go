package api

import (
	"fmt" 
	"time" 
	"github.com/user/agbaravoip_golang/internal/domain"
	// "regexp" // Not using ToSnakeCase from here anymore
	// "strings"
)

// === Account DTOs ===
type CreateAccountRequest struct {
	FriendlyName string `json:"friendly_name"`
	AuthToken    string `json:"auth_token" binding:"required"`
}
type UpdateAccountRequest struct {
	FriendlyName *string `json:"friendly_name"`
	Status       *string `json:"status"`
}
type AccountResponse struct {
	SID          string               `json:"sid"`
	ParentSID    *string              `json:"parent_sid,omitempty"`
	FriendlyName string               `json:"friendly_name"`
	PhoneNumber  string               `json:"phone_number,omitempty"`
	Type         domain.AccountType   `json:"type"`
	Status       domain.AccountStatus `json:"status"`
	CreatedAt    string               `json:"created_at"`
	UpdatedAt    string               `json:"updated_at"`
}
func ToAccountResponse(acc *domain.Account) AccountResponse {
	var parentSID *string
	if acc.ParentSID != nil { parentSID = acc.ParentSID }
	return AccountResponse{
		SID: acc.SID, ParentSID: parentSID, FriendlyName: acc.FriendlyName, PhoneNumber: acc.PhoneNumber,
		Type: acc.Type, Status: acc.Status, CreatedAt: acc.CreatedAt.Format(time.RFC3339), UpdatedAt: acc.UpdatedAt.Format(time.RFC3339),
	}
}
func ToAccountResponseList(accounts []*domain.Account) []AccountResponse {
	responses := make([]AccountResponse, len(accounts))
	for i, acc := range accounts { responses[i] = ToAccountResponse(acc) }
	return responses
}

// === Application DTOs ===
type CreateApplicationRequest struct {
	FriendlyName         string `json:"friendly_name" binding:"required"`
	VoiceURL             string `json:"voice_url,omitempty"`
	VoiceMethod          string `json:"voice_method,omitempty"` 
	VoiceFallbackURL     string `json:"voice_fallback_url,omitempty"`
	VoiceFallbackMethod  string `json:"voice_fallback_method,omitempty"` 
	SmsURL               string `json:"sms_url,omitempty"`
	SmsMethod            string `json:"sms_method,omitempty"` 
	SmsFallbackURL       string `json:"sms_fallback_url,omitempty"`
	SmsFallbackMethod    string `json:"sms_fallback_method,omitempty"` 
	StatusCallbackURL    string `json:"status_callback_url,omitempty"`
	StatusCallbackMethod string `json:"status_callback_method,omitempty"` 
}
type UpdateApplicationRequest struct {
	FriendlyName         *string `json:"friendly_name,omitempty"`
	VoiceURL             *string `json:"voice_url,omitempty"`
	VoiceMethod          *string `json:"voice_method,omitempty"`
	VoiceFallbackURL     *string `json:"voice_fallback_url,omitempty"`
	VoiceFallbackMethod  *string `json:"voice_fallback_method,omitempty"`
	SmsURL               *string `json:"sms_url,omitempty"`
	SmsMethod            *string `json:"sms_method,omitempty"`
	SmsFallbackURL       *string `json:"sms_fallback_url,omitempty"`
	SmsFallbackMethod    *string `json:"sms_fallback_method,omitempty"`
	StatusCallbackURL    *string `json:"status_callback_url,omitempty"`
	StatusCallbackMethod *string `json:"status_callback_method,omitempty"`
}
// Corrected ApplicationResponse struct definition
type ApplicationResponse struct {
	SID                  string `json:"sid"`
	AccountSID           string `json:"account_sid"`
	FriendlyName         string `json:"friendly_name"`
	VoiceURL             string `json:"voice_url,omitempty"`
	VoiceMethod          string `json:"voice_method,omitempty"`
	VoiceFallbackURL     string `json:"voice_fallback_url,omitempty"`
	VoiceFallbackMethod  string `json:"voice_fallback_method,omitempty"`
	SmsURL               string `json:"sms_url,omitempty"`
	SmsMethod            string `json:"sms_method,omitempty"`
	SmsFallbackURL       string `json:"sms_fallback_url,omitempty"`
	SmsFallbackMethod    string `json:"sms_fallback_method,omitempty"`
	StatusCallbackURL    string `json:"status_callback_url,omitempty"`
	StatusCallbackMethod string `json:"status_callback_method,omitempty"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}
func ToApplicationResponse(app *domain.Application) ApplicationResponse {
	return ApplicationResponse {
		SID: app.SID, AccountSID: app.AccountSID, FriendlyName: app.FriendlyName,
		VoiceURL: app.VoiceURL, VoiceMethod: app.VoiceMethod, VoiceFallbackURL: app.VoiceFallbackURL, VoiceFallbackMethod: app.VoiceFallbackMethod,
		SmsURL: app.SmsURL, SmsMethod: app.SmsMethod, SmsFallbackURL: app.SmsFallbackURL, SmsFallbackMethod: app.SmsFallbackMethod,
		StatusCallbackURL: app.StatusCallbackURL, StatusCallbackMethod: app.StatusCallbackMethod,
		CreatedAt: app.CreatedAt.Format(time.RFC3339), UpdatedAt: app.UpdatedAt.Format(time.RFC3339),
	}
}
func ToApplicationResponseList(apps []*domain.Application) []ApplicationResponse {
	responses := make([]ApplicationResponse, len(apps))
	for i, app := range apps { responses[i] = ToApplicationResponse(app) }
	return responses
}

// === Call DTOs ===
type CreateCallRequest struct {
	From             string  `json:"from" binding:"required"`
	To               string  `json:"to" binding:"required"`
	AnswerURL        string  `json:"answer_url,omitempty"` 
	ApplicationSID   *string `json:"application_sid,omitempty"` 
	TimeoutSeconds   *int    `json:"timeout_seconds,omitempty"`
}
type CallResponse struct {
	SID              string             `json:"sid"`
	AccountSID       string             `json:"account_sid"`
	ApplicationSID   *string            `json:"application_sid,omitempty"`
	From             string             `json:"from"`
	To               string             `json:"to"`
	AnswerURL        string             `json:"answer_url,omitempty"`
	Status           domain.CallStatus  `json:"status"`
	Direction        domain.CallDirection `json:"direction"`
	DurationSeconds  int                `json:"duration_seconds"`
	Price            string             `json:"price"` 
	AnsweredBy       *string            `json:"answered_by,omitempty"` 
	TimeoutSeconds   *int               `json:"timeout_seconds,omitempty"`
	HangupCause      *string            `json:"hangup_cause,omitempty"` 
	ForwardedFrom    *string            `json:"forwarded_from,omitempty"` 
	StartTime        string             `json:"start_time,omitempty"` 
	AnswerTime       string             `json:"answer_time,omitempty"` 
	EndTime          string             `json:"end_time,omitempty"`   
	CreatedAt        string             `json:"created_at"`         
	UpdatedAt        string             `json:"updated_at"`         
}
func ToCallResponse(call *domain.Call) CallResponse {
	formatTimePtr := func(t *time.Time) string { 
		if t == nil || t.IsZero() { return "" }
		return t.Format(time.RFC3339)
	}
	var appSid *string
	if call.ApplicationSID != nil && *call.ApplicationSID != "" {	appSid = call.ApplicationSID }
	
	return CallResponse{
		SID: call.SID, AccountSID: call.AccountSID, ApplicationSID: appSid, From: call.FromNum, To: call.ToNum,
		AnswerURL: call.AnswerURL, Status: call.Status, Direction: call.Direction, DurationSeconds: call.DurationSeconds,
		Price: fmt.Sprintf("%.5f", call.Price), AnsweredBy: call.AnsweredBy, TimeoutSeconds: call.TimeoutSeconds,
		HangupCause: call.HangupCause, ForwardedFrom: call.ForwardedFrom, 
		StartTime: formatTimePtr(call.StartTime), AnswerTime: formatTimePtr(call.AnswerTime), EndTime: formatTimePtr(call.EndTime),
		CreatedAt: call.CreatedAt.Format(time.RFC3339), UpdatedAt: call.UpdatedAt.Format(time.RFC3339),
	}
}
func ToCallResponseList(calls []*domain.Call) []CallResponse {
	responses := make([]CallResponse, len(calls))
	for i, call := range calls { responses[i] = ToCallResponse(call) }
	return responses
}

// GenericErrorResponse for API errors
type GenericErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}


