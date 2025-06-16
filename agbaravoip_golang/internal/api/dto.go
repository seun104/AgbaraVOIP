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

// === FreeswitchServer DTOs ===

type CreateFreeswitchServerRequest struct {
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port" binding:"required,gte=1,lte=65535"`
	Password        string `json:"password" binding:"required"` // Will not be stored in plain text
	OutboundAddress string `json:"outbound_address"`
	IsActive        *bool  `json:"is_active"` // Pointer to allow explicit true/false, defaults to true if omitted by service
}

type UpdateFreeswitchServerRequest struct {
	Host            *string `json:"host,omitempty"`
	Port            *int    `json:"port,omitempty,gte=1,lte=65535"`
	Password        *string `json:"password,omitempty"` // For updating password
	OutboundAddress *string `json:"outbound_address,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
}

type FreeswitchServerResponse struct {
	SID             string `json:"sid"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	OutboundAddress string `json:"outbound_address,omitempty"`
	IsActive        bool   `json:"is_active"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func ToFreeswitchServerResponse(fs *domain.FreeswitchServer) FreeswitchServerResponse {
	return FreeswitchServerResponse{
		SID:             fs.SID,
		Host:            fs.Host,
		Port:            fs.Port,
		OutboundAddress: fs.OutboundAddress,
		IsActive:        fs.IsActive,
		CreatedAt:       fs.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       fs.UpdatedAt.Format(time.RFC3339),
	}
}

func ToFreeswitchServerResponseList(servers []*domain.FreeswitchServer) []FreeswitchServerResponse {
	responses := make([]FreeswitchServerResponse, len(servers))
	for i, srv := range servers {
		responses[i] = ToFreeswitchServerResponse(srv)
	}
	return responses
}

// === Gateway DTOs ===

type CreateGatewayRequest struct {
	AccountSID          string                `json:"account_sid" binding:"required"` // Assuming admin specifies which account it's for
	FreeswitchServerSID *string               `json:"freeswitch_server_sid,omitempty"`
	FriendlyName        string                `json:"friendly_name" binding:"required"`
	GatewayString       string                `json:"gateway_string" binding:"required"`
	Codecs              []string              `json:"codecs,omitempty"`
	RetryCount          *int                  `json:"retry_count,omitempty,gte=0"`    // Pointer for optional with default
	TimeoutSeconds      *int                  `json:"timeout_seconds,omitempty,gte=1"` // Pointer for optional with default
	Routes              domain.GatewayRoutes  `json:"routes,omitempty"`
	IsEnabled           *bool                 `json:"is_enabled,omitempty"` // Pointer for optional with default
}

type UpdateGatewayRequest struct {
	FreeswitchServerSID *string               `json:"freeswitch_server_sid,omitempty"`
	FriendlyName        *string               `json:"friendly_name,omitempty"`
	GatewayString       *string               `json:"gateway_string,omitempty"`
	Codecs              []string              `json:"codecs,omitempty"` // Send full list for update, or handle partial
	RetryCount          *int                  `json:"retry_count,omitempty,gte=0"`
	TimeoutSeconds      *int                  `json:"timeout_seconds,omitempty,gte=1"`
	Routes              *domain.GatewayRoutes `json:"routes,omitempty"`
	IsEnabled           *bool                 `json:"is_enabled,omitempty"`
}

type GatewayResponse struct {
	SID                 string                `json:"sid"`
	AccountSID          string                `json:"account_sid"`
	FreeswitchServerSID *string               `json:"freeswitch_server_sid,omitempty"`
	FriendlyName        string                `json:"friendly_name"`
	GatewayString       string                `json:"gateway_string"`
	Codecs              []string              `json:"codecs,omitempty"`
	RetryCount          int                   `json:"retry_count"`
	TimeoutSeconds      int                   `json:"timeout_seconds"`
	Routes              domain.GatewayRoutes  `json:"routes,omitempty"`
	IsEnabled           bool                  `json:"is_enabled"`
	CreatedAt           string                `json:"created_at"`
	UpdatedAt           string                `json:"updated_at"`
}

func ToGatewayResponse(gw *domain.Gateway) GatewayResponse {
	return GatewayResponse{
		SID:                 gw.SID,
		AccountSID:          gw.AccountSID,
		FreeswitchServerSID: gw.FreeswitchServerSID,
		FriendlyName:        gw.FriendlyName,
		GatewayString:       gw.GatewayString,
		Codecs:              gw.Codecs,
		RetryCount:          gw.RetryCount,
		TimeoutSeconds:      gw.TimeoutSeconds,
		Routes:              gw.Routes,
		IsEnabled:           gw.IsEnabled,
		CreatedAt:           gw.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           gw.UpdatedAt.Format(time.RFC3339),
	}
}

func ToGatewayResponseList(gateways []*domain.Gateway) []GatewayResponse {
	responses := make([]GatewayResponse, len(gateways))
	for i, gw := range gateways {
		responses[i] = ToGatewayResponse(gw)
	}
	return responses
}

