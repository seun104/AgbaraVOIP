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

// === Conference Management DTOs ===

// ConferenceResponse represents a conference resource in API responses.
type ConferenceResponse struct {
	SID           string                `json:"sid"`
	AccountSID    string                `json:"account_sid"`
	FriendlyName  string                `json:"friendly_name"`
	Status        domain.ConferenceStatus `json:"status"`
	StartTime     *string               `json:"start_time,omitempty"` // RFC3339 format
	EndTime       *string               `json:"end_time,omitempty"`   // RFC3339 format
	CreatedAt     string                `json:"created_at"`           // RFC3339 format
	UpdatedAt     string                `json:"updated_at"`           // RFC3339 format
	// ParticipantsLink string             `json:"participants_link,omitempty"` // Link to list participants
}

// ToConferenceResponse converts a domain.Conference object to a ConferenceResponse DTO.
func ToConferenceResponse(conf *domain.Conference) ConferenceResponse {
	var startTime, endTime *string
	if conf.StartTime != nil && !conf.StartTime.IsZero() {
		st := conf.StartTime.Format(time.RFC3339)
		startTime = &st
	}
	if conf.EndTime != nil && !conf.EndTime.IsZero() {
		et := conf.EndTime.Format(time.RFC3339)
		endTime = &et
	}
	return ConferenceResponse{
		SID:          conf.SID,
		AccountSID:   conf.AccountSID,
		FriendlyName: conf.FriendlyName,
		Status:       conf.Status,
		StartTime:    startTime,
		EndTime:      endTime,
		CreatedAt:    conf.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    conf.UpdatedAt.Format(time.RFC3339),
	}
}

// ToConferenceResponseList converts a slice of domain.Conference objects to a slice of ConferenceResponse DTOs.
func ToConferenceResponseList(confs []*domain.Conference) []ConferenceResponse {
	responses := make([]ConferenceResponse, len(confs))
	for i, conf := range confs {
		responses[i] = ToConferenceResponse(conf)
	}
	return responses
}

// ParticipantResponse represents a conference participant in API responses.
type ParticipantResponse struct {
	SID           string     `json:"sid"`
	ConferenceSID string     `json:"conference_sid"`
	CallSID       string     `json:"call_sid"`
	AccountSID    string     `json:"account_sid"`
	IsMuted       bool       `json:"is_muted"`
	IsModerator   bool       `json:"is_moderator"`
	JoinTime      string     `json:"join_time"`      // RFC3339 format
	LeaveTime     *string    `json:"leave_time,omitempty"` // RFC3339 format
}

// ToParticipantResponse converts a domain.ConferenceParticipant object to a ParticipantResponse DTO.
func ToParticipantResponse(p *domain.ConferenceParticipant) ParticipantResponse {
	var leaveTime *string
	if p.LeaveTime != nil && !p.LeaveTime.IsZero() {
		lt := p.LeaveTime.Format(time.RFC3339)
		leaveTime = &lt
	}
	return ParticipantResponse{
		SID:           p.SID,
		ConferenceSID: p.ConferenceSID,
		CallSID:       p.CallSID,
		AccountSID:    p.AccountSID,
		IsMuted:       p.IsMuted,
		IsModerator:   p.IsModerator,
		JoinTime:      p.JoinTime.Format(time.RFC3339),
		LeaveTime:     leaveTime,
	}
}

// ToParticipantResponseList converts a slice of domain.ConferenceParticipant objects to a slice of ParticipantResponse DTOs.
func ToParticipantResponseList(participants []*domain.ConferenceParticipant) []ParticipantResponse {
	responses := make([]ParticipantResponse, len(participants))
	for i, p := range participants {
		responses[i] = ToParticipantResponse(p)
	}
	return responses
}

// ConferenceControlPlayRequest defines the request for playing audio in a conference.
// (Identical to CallPlayRequest, can be aliased or duplicated for clarity in Swagger docs)
type ConferenceControlPlayRequest CallPlayRequest

// ConferenceControlSayRequest defines the request for speaking text in a conference.
// (Identical to CallSayRequest)
type ConferenceControlSayRequest CallSayRequest

// ConferenceControlRecordRequest defines the request for recording a conference.
// (Similar to CallRecordRequest, action might be specific like 'start', 'stop', 'pause', 'resume')
// For now, using the same CallRecordRequest structure.
type ConferenceControlRecordRequest CallRecordRequest


// ParticipantMuteRequest defines the request to mute/unmute a participant.
type ParticipantMuteRequest struct {
	Mute *bool `json:"mute" binding:"required"` // Pointer to distinguish false from not set
}

// ParticipantKickRequest (No body needed, just action via URL)

// Note: CallActionResponse can be reused for conference control actions.
// type ConferenceActionResponse CallActionResponse

// === Recording Management DTOs ===

// RecordingResponse represents a recording resource in API responses.
type RecordingResponse struct {
	SID              string    `json:"sid"`
	AccountSID       string    `json:"account_sid"`
	CallSID          *string   `json:"call_sid,omitempty"`
	ConferenceSID    *string   `json:"conference_sid,omitempty"`
	DurationSeconds  uint32    `json:"duration_seconds"`
	FilePath         string    `json:"file_path"` // Consider if this should be a downloadable URL or an internal path
	Format           string    `json:"format"`
	SizeBytes        int64     `json:"size_bytes"`
	CreatedAt        string    `json:"created_at"` // RFC3339 format
	UpdatedAt        string    `json:"updated_at"` // RFC3339 format
	// Status        *string   `json:"status,omitempty"` // If status field is added to domain.Recording
}

// ToRecordingResponse converts a domain.Recording object to a RecordingResponse DTO.
func ToRecordingResponse(rec *domain.Recording) RecordingResponse {
	return RecordingResponse{
		SID:              rec.SID,
		AccountSID:       rec.AccountSID,
		CallSID:          rec.CallSID,
		ConferenceSID:    rec.ConferenceSID,
		DurationSeconds:  rec.DurationSeconds,
		FilePath:         rec.FilePath, // Security: Ensure this path is safe to expose or transform to a secure URL
		Format:           rec.Format,
		SizeBytes:        rec.SizeBytes,
		CreatedAt:        rec.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        rec.UpdatedAt.Format(time.RFC3339),
		// Status:        rec.Status // If status field is added
	}
}

// ToRecordingResponseList converts a slice of domain.Recording objects to a slice of RecordingResponse DTOs.
func ToRecordingResponseList(recs []*domain.Recording) []RecordingResponse {
	responses := make([]RecordingResponse, len(recs))
	for i, rec := range recs {
		responses[i] = ToRecordingResponse(rec)
	}
	return responses
}

// === SMS Management DTOs ===

// SendSMSRequest defines the request for sending an SMS message.
type SendSMSRequest struct {
	From              string  `json:"from" binding:"required"` // Sender ID/number
	To                string  `json:"to" binding:"required"`   // Recipient number
	Body              string  `json:"body" binding:"required"`
	StatusCallbackURL *string `json:"status_callback_url,omitempty,url"` // Optional URL for status updates
}

// SMSMessageResponse represents an SMS message resource in API responses.
type SMSMessageResponse struct {
	SID               string              `json:"sid"`
	AccountSID        string              `json:"account_sid"`
	To                string              `json:"to"`
	From              string              `json:"from"`
	Body              string              `json:"body"`
	Status            domain.SMSStatus    `json:"status"`
	Direction         domain.SMSDirection `json:"direction"`
	Price             *string             `json:"price,omitempty"`
	PriceUnit         *string             `json:"price_unit,omitempty"`
	ErrorCode         *int32              `json:"error_code,omitempty"`
	ErrorMessage      *string             `json:"error_message,omitempty"`
	GatewayMessageSID *string             `json:"gateway_message_sid,omitempty"`
	SentAt            *string             `json:"sent_at,omitempty"`     // RFC3339 format
	DeliveredAt       *string             `json:"delivered_at,omitempty"` // RFC3339 format
	CreatedAt         string              `json:"created_at"`             // RFC3339 format
	UpdatedAt         string              `json:"updated_at"`             // RFC3339 format
}

// ToSMSMessageResponse converts a domain.SMSMessage object to an SMSMessageResponse DTO.
func ToSMSMessageResponse(sms *domain.SMSMessage) SMSMessageResponse {
	var price, priceUnit, errorMessage, gatewayMsgSid *string
	var sentAt, deliveredAt *string
	var errorCode *int32

	if sms.Price.Valid {
		price = &sms.Price.String
	}
	if sms.PriceUnit.Valid {
		priceUnit = &sms.PriceUnit.String
	}
	if sms.ErrorMessage.Valid {
		errorMessage = &sms.ErrorMessage.String
	}
	if sms.GatewayMessageSID.Valid {
		gatewayMsgSid = &sms.GatewayMessageSID.String
	}
	if sms.ErrorCode.Valid {
		errorCode = &sms.ErrorCode.Int32
	}
	if sms.SentAt.Valid {
		sAt := sms.SentAt.Time.Format(time.RFC3339)
		sentAt = &sAt
	}
	if sms.DeliveredAt.Valid {
		dAt := sms.DeliveredAt.Time.Format(time.RFC3339)
		deliveredAt = &dAt
	}

	return SMSMessageResponse{
		SID:               sms.SID,
		AccountSID:        sms.AccountSID,
		To:                sms.To,
		From:              sms.From,
		Body:              sms.Body,
		Status:            sms.Status,
		Direction:         sms.Direction,
		Price:             price,
		PriceUnit:         priceUnit,
		ErrorCode:         errorCode,
		ErrorMessage:      errorMessage,
		GatewayMessageSID: gatewayMsgSid,
		SentAt:            sentAt,
		DeliveredAt:       deliveredAt,
		CreatedAt:         sms.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         sms.UpdatedAt.Format(time.RFC3339),
	}
}

// ToSMSMessageResponseList converts a slice of domain.SMSMessage objects to a slice of SMSMessageResponse DTOs.
func ToSMSMessageResponseList(smsList []*domain.SMSMessage) []SMSMessageResponse {
	responses := make([]SMSMessageResponse, len(smsList))
	for i, sms := range smsList {
		responses[i] = ToSMSMessageResponse(sms)
	}
	return responses
}

// === Live Call Control DTOs ===

// CallPlayRequest defines the request for playing audio on a live call.
type CallPlayRequest struct {
	URL  string `json:"url" binding:"required,url"`
	Loop *int   `json:"loop,omitempty,gte=0"`      // 0 or 1 for no loop effectively in most FS apps, specific loop app might be needed for >1
	Legs string `json:"legs,omitempty,oneof=aleg bleg both"` // aleg, bleg, both (defaults to aleg if empty)
}

// CallSayRequest defines the request for speaking text on a live call.
type CallSayRequest struct {
	Text     string `json:"text" binding:"required"`
	Language *string `json:"language,omitempty"` // e.g., "en-US"
	Voice    *string `json:"voice,omitempty"`    // e.g., "man", "woman", specific TTS engine voice
	Legs     string  `json:"legs,omitempty,oneof=aleg bleg both"`
}

// CallDTMFRequest defines the request for sending DTMF tones on a live call.
type CallDTMFRequest struct {
	Digits     string `json:"digits" binding:"required"` // e.g., "1234#"
	DurationMs *int   `json:"duration_ms,omitempty,gte=100,lte=2000"` // Duration for each digit in ms
	Legs       string `json:"legs,omitempty,oneof=aleg bleg both"`
}

// CallRecordAction defines the type for recording actions.
type CallRecordAction string
const (
    CallRecordActionStart CallRecordAction = "start"
    CallRecordActionStop  CallRecordAction = "stop"
)

// CallRecordRequest defines the request for starting or stopping call recording.
type CallRecordRequest struct {
	Action             CallRecordAction `json:"action" binding:"required,oneof=start stop"`
	FileName           *string          `json:"file_name,omitempty"`            // Optional: Desired filename (server might add UUIDs etc.)
	MaxDurationSeconds *int             `json:"max_duration_seconds,omitempty,gte=1"`
	Format             *string          `json:"format,omitempty,oneof=wav mp3"` // Example formats
	PlayBeep           *bool            `json:"play_beep,omitempty"`
	// Add other params like silence_thresh, silence_hits if needed
}

// CallActionResponse is a generic response for live call control actions.
type CallActionResponse struct {
	CallSID string `json:"call_sid"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	JobID   string `json:"job_id,omitempty"` // Optional: If the action is async and returns a job ID
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

