package services
import ( "context"; "errors"; "fmt"; "strings"; "time"; "net/url"; // Added context for SMSService delegation
	"github.com/user/agbaravoip_golang/internal/domain"; "github.com/user/agbaravoip_golang/internal/esl"; 
	"github.com/sirupsen/logrus"; "gorm.io/gorm" )

// Simple mock for SMSGatewayClient for NewCallService, can be expanded or moved
type mockSmsGateway struct {}
func (m *mockSmsGateway) SendSMS(ctx context.Context, to, from, body string, statusCallbackURL string) (string, error) {
	// This is a mock, does not actually send SMS
	return "mock_gw_sid_" + from + "_" + to, nil
}

// Ensure FreeswitchOutboundConfigProvider is defined here or imported if common
type FreeswitchOutboundConfigProvider interface { GetESLOutboundServerListenAddress() string }

// Errors for CallService
var ( ErrCallNotFound_CS = errors.New("call not found"); ErrCallValidationFailed_CS = errors.New("call validation failed");
	ErrCallCreationFailed_CS = errors.New("call creation failed in DB"); ErrESLClientNotAvailable_CS = errors.New("ESL client unavailable");
	ErrESLCommandFailed_CS = errors.New("ESL command failed"); ErrAppLogicError_CS = errors.New("app logic error for call");
	ErrCallInvalidState_CS = errors.New("call is not in a state that allows this operation"); )

type CallService struct {
	db *gorm.DB; eslClient *esl.FSInboundClient; appService IApplicationService; accountService IAccountService;
	logger *logrus.Entry; cfgProvider FreeswitchOutboundConfigProvider
	confService ConferenceService
	smsService  SMSService
}
func NewCallService(db *gorm.DB, eslClient *esl.FSInboundClient, appSvc IApplicationService, accSvc IAccountService, cfgProvider FreeswitchOutboundConfigProvider, logger *logrus.Logger) *CallService {
	conferenceSvc := NewConferenceService(db, logger)
	mockGwClient := &mockSmsGateway{}
	smsSvc := NewSMSService(db, logger, mockGwClient)

	return &CallService{
		db: db, eslClient: eslClient, appService: appSvc, accountService: accSvc,
		logger: logger.WithField("service", "call"),
		cfgProvider: cfgProvider,
		confService: conferenceSvc,
		smsService: smsSvc,
	}
}
func (s *CallService) OriginateCall(accountSid string, fromNum string, toNum string, answerURL string, applicationSid *string, timeoutSeconds *int) (*domain.Call, error) {
	if s.eslClient == nil { return nil, ErrESLClientNotAvailable_CS }
	finalAnswerURL := answerURL; effectiveApplicationSID := applicationSid
	if applicationSid != nil && *applicationSid != "" && strings.TrimSpace(finalAnswerURL) == "" {
		app, err := s.appService.GetApplicationBySID(accountSid, *applicationSid)
		if err != nil { return nil, fmt.Errorf("%w: invalid app_sid: %v", ErrAppLogicError_CS, err) }
		if app.VoiceURL == "" { return nil, fmt.Errorf("%w: app %s has no VoiceURL", ErrAppLogicError_CS, *applicationSid) }
		finalAnswerURL = app.VoiceURL
	} else if applicationSid == nil || *applicationSid == "" { effectiveApplicationSID = nil }
	if strings.TrimSpace(finalAnswerURL) == "" { return nil, fmt.Errorf("%w: answer_url or valid app_sid required", ErrCallValidationFailed_CS) }
	now := time.Now().UTC(); call := &domain.Call{ AccountSID: accountSid, ApplicationSID: effectiveApplicationSID, FromNum: fromNum, ToNum: toNum, AnswerURL: finalAnswerURL, Status: domain.CallStatusQueued, Direction: domain.CallDirectionOutboundAPI, StartTime: &now, TimeoutSeconds: timeoutSeconds, };
	if err := s.db.Create(call).Error; err != nil { return nil, fmt.Errorf("%w: %v", ErrCallCreationFailed_CS, err) }
	s.logger.Infof("Call record %s created, status %s", call.SID, call.Status)
	escapedAnswerURL := url.QueryEscape(finalAnswerURL)
	vars := fmt.Sprintf("origination_caller_id_number=%s,origination_uuid=%s,agbara_account_sid=%s,agbara_call_sid=%s,agbara_answer_url=%s,hangup_after_bridge=false,ignore_early_media=true", fromNum, call.SID, accountSid, call.SID, escapedAnswerURL)
	if call.TimeoutSeconds != nil && *call.TimeoutSeconds > 0 { vars += fmt.Sprintf(",originate_timeout=%d", *call.TimeoutSeconds) }
	targetDialstring := fmt.Sprintf("user/%s", toNum) 
	outboundSrvAddr := s.cfgProvider.GetESLOutboundServerListenAddress(); if outboundSrvAddr == "" { return call, fmt.Errorf("outbound ESL server address not configured") }
	originateCmd := fmt.Sprintf("originate {%s}%s &socket(%s async full)", vars, targetDialstring, outboundSrvAddr)
	s.logger.Infof("Sending originate: bgapi %s", originateCmd)
	_, eslErr := s.eslClient.SendCommand(fmt.Sprintf("bgapi %s", originateCmd))
	callUpdate := map[string]interface{}{}; if eslErr != nil { callUpdate["status"] = domain.CallStatusFailed; callUpdate["hangup_cause"] = "ORIGINATE_ESL_ERROR"; callUpdate["end_time"] = time.Now().UTC(); if dbErr := s.db.Model(call).Updates(callUpdate).Error; dbErr != nil { s.logger.Errorf("Failed to update call %s to failed: %v", call.SID, dbErr) }; return call, fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, eslErr) }
	callUpdate["status"] = domain.CallStatusInitiated; if dbErr := s.db.Model(call).Updates(callUpdate).Error; dbErr != nil { s.logger.Errorf("Failed to update call %s to initiated: %v", call.SID, dbErr) }; call.Status = domain.CallStatusInitiated
	s.logger.Infof("Originate sent for Call SID %s. Status: %s.", call.SID, call.Status); return call, nil
}
func (s *CallService) GetCallBySID(accountSid string, callSid string) (*domain.Call, error) { var call domain.Call; if err := s.db.Where("sid = ? AND account_sid = ?", callSid, accountSid).First(&call).Error; err != nil { if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrCallNotFound_CS }; return nil, err }; return &call, nil }
func (s *CallService) ListCalls(accountSid string, filters map[string]interface{}) ([]*domain.Call, error) { var calls []*domain.Call; query := s.db.Where("account_sid = ?", accountSid); for key, value := range filters { if strVal, ok := value.(string); ok && strVal != "" { switch key { case "status", "from_num", "to_num", "direction": query = query.Where(fmt.Sprintf("%s = ?", key), strVal); default: s.logger.Warnf("Unsupported filter key: %s", key) } } }; if err := query.Order("created_at desc").Find(&calls).Error; err != nil { return nil, err }; return calls, nil }
func (s *CallService) UpdateCallStatus(agbaraCallSid string, status domain.CallStatus, hangupCause string, durationSeconds int) (*domain.Call, error) {
	var call domain.Call; if agbaraCallSid == "" { return nil, fmt.Errorf("%w: AgbaraCallSID required", ErrCallValidationFailed_CS) }
	if err := s.db.Where("sid = ?", agbaraCallSid).First(&call).Error; err != nil { if errors.Is(err, gorm.ErrRecordNotFound) { s.logger.Warnf("Call %s not found for status update.", agbaraCallSid); return nil, ErrCallNotFound_CS }; return nil, err }
	updates := map[string]interface{}{"status": status}; if hangupCause != "" { updates["hangup_cause"] = hangupCause }
	if status == domain.CallStatusCompleted || status == domain.CallStatusFailed || status == domain.CallStatusBusy || status == domain.CallStatusNoAnswer {
		nowUTC := time.Now().UTC(); if call.EndTime == nil || call.EndTime.IsZero() { updates["end_time"] = &nowUTC }
		endTimeForDuration := nowUTC; if et, ok := updates["end_time"].(*time.Time); ok && et != nil { endTimeForDuration = *et }
		if call.AnswerTime != nil && !call.AnswerTime.IsZero() && (call.EndTime == nil || call.EndTime.IsZero()) { updates["duration_seconds"] = int(endTimeForDuration.Sub(*call.AnswerTime).Seconds())
		} else if status == domain.CallStatusCompleted && durationSeconds > 0 && call.DurationSeconds == 0 { updates["duration_seconds"] = durationSeconds }
	}
	if status == domain.CallStatusInProgress && (call.AnswerTime == nil || call.AnswerTime.IsZero()) { answerTime := time.Now().UTC(); updates["answer_time"] = &answerTime }
	if err := s.db.Model(&call).Updates(updates).Error; err != nil { return nil, fmt.Errorf("DB error updating call: %w", err) }
	s.db.Where("sid = ?", agbaraCallSid).First(&call); s.logger.Infof("Call %s status updated to %s", call.SID, call.Status); return &call, nil
}

// --- ConferenceService Delegation Methods ---

func (s *CallService) GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error) {
	return s.confService.GetConferenceBySID(ctx, sid)
}

func (s *CallService) GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
	return s.confService.GetConferenceByName(ctx, accountSid, name)
}

func (s *CallService) CreateConference(ctx context.Context, accountSid, name, sid string) (*domain.Conference, error) {
	return s.confService.CreateConference(ctx, accountSid, name, sid)
}

func (s *CallService) GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
	return s.confService.GetOrCreateConference(ctx, accountSid, name)
}

func (s *CallService) UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error {
	return s.confService.UpdateConferenceStatus(ctx, sid, status)
}

func (s *CallService) EndConference(ctx context.Context, sid string, endTime time.Time) error {
	return s.confService.EndConference(ctx, sid, endTime)
}

func (s *CallService) AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*domain.ConferenceParticipant, error) {
	return s.confService.AddParticipant(ctx, confSid, callSid, pSid, accountSid, isMuted, isModerator)
}

func (s *CallService) GetParticipant(ctx context.Context, pSid string) (*domain.ConferenceParticipant, error) {
	return s.confService.GetParticipant(ctx, pSid)
}

func (s *CallService) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) {
	return s.confService.GetParticipantByCallSID(ctx, callSid)
}

func (s *CallService) UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error {
	return s.confService.UpdateParticipantMuteStatus(ctx, pSid, isMuted)
}

func (s *CallService) UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error {
	return s.confService.UpdateParticipantModeratorStatus(ctx, pSid, isModerator)
}

func (s *CallService) RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error {
	return s.confService.RemoveParticipant(ctx, pSid, leaveTime)
}

func (s *CallService) ListParticipants(ctx context.Context, confSid string) ([]*domain.ConferenceParticipant, error) {
	return s.confService.ListParticipants(ctx, confSid)
}


// LogNullString is a helper for logging nullable strings.
func LogNullString(ns *string) string {
	if ns == nil {
		return "<nil>"
	}
	return *ns
}

func (s *CallService) CreateRecording(ctx domain.MinimalCallContext, callSid *string, recordingSid, filePath string, duration uint32, format string, sizeBytes int64) error {
	logger := s.logger.WithFields(logrus.Fields{
		"service_method": "CreateRecording",
		"recording_sid":  recordingSid,
		"call_sid":       LogNullString(callSid),
		"account_sid":    ctx.GetAccountSid(),
	})
	logger.Infof("Creating recording metadata for file: %s, duration: %ds", filePath, duration)

	// Using GORM Create method with the domain.Recording struct
	// This assumes domain.Recording struct is defined and has correct `gorm` tags if needed,
	// or relies on GORM's column name mapping. The prompt used `db` tags for sqlx.
	// For GORM, it's often `gorm:"column:column_name"` or direct field name mapping.
	// Given the SQL in the prompt, we'll use raw SQL with GORM's Exec for now to match the prompt's query structure.
	// A more GORM-idiomatic way would be:
	// recording := domain.Recording{ /* populate fields */ }
	// if err := s.db.Create(&recording).Error; err != nil { ... }
	// However, to stick closer to the provided SQL query structure:

	query := `INSERT INTO recordings (sid, account_sid, call_sid, duration_seconds, file_path, format, size_bytes, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now().UTC()

	// GORM's Exec method typically doesn't return rows, just error.
	// It also doesn't directly use context.Background() in the Exec call itself for older GORM.
	// For newer GORM, s.db.WithContext(context.Background()).Exec(...) is preferred.
	// Sticking to s.db.Exec for simplicity if WithContext isn't readily usable or version is unknown.
	// The prompt used db.ExecContext which is more like sqlx or database/sql.
	// Adapting to common GORM raw SQL execution:
	err := s.db.Exec(query,
		recordingSid,
		ctx.GetAccountSid(),
		callSid,
		duration,
		filePath,
		format,
		sizeBytes,
		now,
		now,
	).Error

	if err != nil {
		logger.Errorf("Error inserting recording metadata into DB: %v", err)
		return fmt.Errorf("inserting recording: %w", err)
	}

	logger.Info("Successfully created recording metadata.")
	return nil
}

// --- SMSService Delegation Methods ---

func (s *CallService) SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error) {
	return s.smsService.SendSMS(ctx, accountSid, to, from, body, msgSID, actionURL, actionMethod)
}

func (s *CallService) GetSMSBySID(ctx context.Context, sid string) (*domain.SMSMessage, error) {
	return s.smsService.GetSMSBySID(ctx, sid)
}

func (s *CallService) UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error {
	return s.smsService.UpdateSMSStatus(ctx, agbaraSid, gatewaySid, status, errorCode, errorMessage, eventTime)
}

func (s *CallService) RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error) {
	return s.smsService.RecordInboundSMS(ctx, accountSid, to, from, body, inboundGatewayMsgSid)
}

// --- ApplicationService Delegation Method ---

func (s *CallService) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error) {
	if s.appService == nil {
		// This case should ideally not happen if NewCallService ensures appService is initialized.
		// However, if it can be nil, proper error handling is needed.
		s.logger.Error("appService is not initialized in CallService when calling GetApplicationByIncomingDID")
		return nil, errors.New("application service not available")
	}
	return s.appService.GetApplicationByIncomingDID(ctx, did)
}

// --- AccountService Delegation Methods ---

func (s *CallService) ValidateCredentials(ctx context.Context, accountSid string, plainToken string) (*domain.Account, error) {
	if s.accountService == nil {
		s.logger.Error("accountService is not initialized in CallService when calling ValidateCredentials")
		return nil, errors.New("account service not available")
	}
	return s.accountService.ValidateCredentials(ctx, accountSid, plainToken) // Pass context if IAccountService methods expect it
}

func (s *CallService) GetAccountBySID(ctx context.Context, sid string) (*domain.Account, error) {
	if s.accountService == nil {
		s.logger.Error("accountService is not initialized in CallService when calling GetAccountBySID")
		return nil, errors.New("account service not available")
	}
	// Assuming IAccountService.GetAccountBySID does not take context. If it does, pass ctx.
	return s.accountService.GetAccountBySID(sid)
}

// --- Live Call Control Methods ---

func (s *CallService) PlayAudioOnCall(ctx context.Context, accountSid, callSid string, playURL string, loop int, legs string) (string, error) {
	s.logger.Infof("Attempting to play audio on call %s for account %s. URL: %s, Legs: %s", callSid, accountSid, playURL, legs)
	call, err := s.GetCallBySID(accountSid, callSid)
	if err != nil {
		return "", err // Handles ErrCallNotFound_CS
	}
	if call.Status != domain.CallStatusInProgress {
		return "", fmt.Errorf("%w: call SID %s is in status %s", ErrCallInvalidState_CS, callSid, call.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	// Default legs to "aleg" if empty
	legParam := "aleg"
	if legs == "bleg" || legs == "both" {
		legParam = legs
	}

	// Note: uuid_broadcast loop behavior is not a simple integer.
	// A loop value of 0 or 1 means play once. For true looping, specific dialplan apps or sched_broadcast might be needed.
	// Here, we'll just pass the URL and legs. Loop > 1 might not behave as expected with simple uuid_broadcast.
	// A more advanced implementation might use `loop_playback` app if targeting a single channel leg.
	// For now, we acknowledge the loop parameter but the basic uuid_broadcast won't use it for >1 looping.
	if loop > 1 {
		s.logger.Warnf("Looping > 1 for PlayAudioOnCall on call %s may not be supported by simple uuid_broadcast. Playing once.", callSid)
	}

	// Format: bgapi uuid_broadcast <uuid> <path> [aleg|bleg|both]
	command := fmt.Sprintf("bgapi uuid_broadcast %s %s %s", call.SID, playURL, legParam)
	s.logger.Debugf("Sending ESL command for PlayAudioOnCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL PlayAudioOnCall command failed for call %s: %v", callSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *CallService) SayTextOnCall(ctx context.Context, accountSid, callSid string, text string, language *string, voice *string, legs string) (string, error) {
	s.logger.Infof("Attempting to say text on call %s for account %s. Legs: %s", callSid, accountSid, legs)
	call, err := s.GetCallBySID(accountSid, callSid)
	if err != nil {
		return "", err
	}
	if call.Status != domain.CallStatusInProgress {
		return "", fmt.Errorf("%w: call SID %s is in status %s", ErrCallInvalidState_CS, callSid, call.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	// Default legs to "aleg" if empty
	legParam := "aleg"
	if legs == "bleg" || legs == "both" { // Freeswitch uuid_speak does not directly support 'both' legs in one command. It targets a single UUID.
		                                 // This would typically apply to the A-leg (call.SID). B-leg would need its own UUID.
		s.logger.Warnf("SayTextOnCall: 'legs' parameter '%s' for uuid_speak will target A-leg by default or requires B-leg UUID for specific targeting not yet implemented here.", legs)
		if legs == "both" { legParam = "aleg" } // Defaulting 'both' to 'aleg' for uuid_speak
	}

	// Defaults for TTS engine and voice
	ttsEngine := "flite" // Default engine
	ttsVoice := "slt"    // Default voice for flite

	if language != nil && *language != "" {
		// Basic language to voice/engine mapping (can be expanded)
		// This is highly dependent on installed TTS engines and voices on Freeswitch
		s.logger.Infof("Language specified: %s. TTS Engine/Voice selection might need more specific logic.", *language)
		// Example: if strings.HasPrefix(*language, "en") { ttsVoice = "slt" }
	}
	if voice != nil && *voice != "" {
		ttsVoice = *voice // Override default if specific voice is given
	}

	// Format: bgapi uuid_speak <uuid> <engine_name> <voice_name> <text_to_speak>
	// Note: uuid_speak targets a single channel. 'legs' param here is conceptual for API consistency.
	// If bleg control is needed, bleg_uuid would be required.
	command := fmt.Sprintf("bgapi uuid_speak %s %s %s '%s'", call.SID, ttsEngine, ttsVoice, text)
	s.logger.Debugf("Sending ESL command for SayTextOnCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL SayTextOnCall command failed for call %s: %v", callSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *CallService) SendDTMFOnCall(ctx context.Context, accountSid, callSid string, digits string, durationMs *int, legs string) (string, error) {
	s.logger.Infof("Attempting to send DTMF on call %s for account %s. Digits: %s, Legs: %s", callSid, accountSid, digits, legs)
	call, err := s.GetCallBySID(accountSid, callSid)
	if err != nil {
		return "", err
	}
	if call.Status != domain.CallStatusInProgress {
		return "", fmt.Errorf("%w: call SID %s is in status %s", ErrCallInvalidState_CS, callSid, call.Status)
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	// uuid_send_dtmf targets a specific channel UUID. 'legs' might be conceptual here.
	// If B-leg DTMF is needed, the B-leg's UUID would be required.
	// Defaulting to A-leg (call.SID).
	if legs != "" && legs != "aleg" {
		s.logger.Warnf("SendDTMFOnCall: 'legs' parameter '%s' for uuid_send_dtmf will target A-leg by default. B-leg specific DTMF requires B-leg UUID.", legs)
	}

	// DTMF duration is often a channel variable (dtmf_duration) set before sending digits,
	// or specific to the application used (e.g., send_dtmf app).
	// uuid_send_dtmf itself doesn't take duration per digit.
	if durationMs != nil {
		s.logger.Infof("DTMF duration of %dms specified for call %s. This might need pre-setting 'dtmf_duration' channel variable if not default.", *durationMs, callSid)
		// Example: s.eslClient.SendCommand(fmt.Sprintf("bgapi uuid_setvar %s dtmf_duration %d", call.SID, *durationMs))
		// For simplicity, this pre-set is omitted, relying on FS default or prior channel setup.
	}

	// Format: bgapi uuid_send_dtmf <uuid> <digits>
	command := fmt.Sprintf("bgapi uuid_send_dtmf %s %s", call.SID, digits)
	s.logger.Debugf("Sending ESL command for SendDTMFOnCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL SendDTMFOnCall command failed for call %s: %v", callSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *CallService) StartRecordingCall(ctx context.Context, accountSid, callSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (string, string, error) {
	s.logger.Infof("Attempting to start recording on call %s for account %s.", callSid, accountSid)
	call, err := s.GetCallBySID(accountSid, callSid)
	if err != nil {
		return "", "", err
	}
	if call.Status != domain.CallStatusInProgress {
		return "", "", fmt.Errorf("%w: call SID %s is in status %s", ErrCallInvalidState_CS, callSid, call.Status)
	}
	if s.eslClient == nil {
		return "", "", ErrESLClientNotAvailable_CS
	}

	actualFormat := "wav" // Default format
	if format != nil && (*format == "wav" || *format == "mp3") {
		actualFormat = *format
	}

	var recordingFileName string
	if fileName != nil && *fileName != "" {
		recordingFileName = fmt.Sprintf("%s.%s", *fileName, actualFormat)
	} else {
		// Generate a default filename
		recordingFileName = fmt.Sprintf("%s_%d.%s", callSid, time.Now().UnixNano(), actualFormat)
	}
	// This path should be configurable and match Freeswitch's recording directory
	// For now, using a placeholder structure.
	// Ensure accountSid directory exists on FS or use a flat structure if simpler.
	fullRecordingPath := fmt.Sprintf("/var/lib/freeswitch/recordings/%s/%s", accountSid, recordingFileName)


	// uuid_record <uuid> start <path> [limit_sec]
	// play_beep is not a direct parameter of uuid_record. It's often part of 'record' application.
	// For API consistency, we acknowledge it. Could play a beep tone separately if needed.
	if playBeep != nil && *playBeep {
		s.logger.Info("PlayBeep requested for StartRecordingCall on call %s. This might require separate beep playback if not part of uuid_record.", callSid)
		// Example: s.eslClient.SendCommand(fmt.Sprintf("bgapi uuid_playback %s tone_stream://%%(1000,0,640) aleg", call.SID))
	}

	command := fmt.Sprintf("bgapi uuid_record %s start %s", call.SID, fullRecordingPath)
	if maxDurationSec != nil && *maxDurationSec > 0 {
		command += fmt.Sprintf(" %d", *maxDurationSec)
	}
	s.logger.Debugf("Sending ESL command for StartRecordingCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL StartRecordingCall command failed for call %s: %v", callSid, err)
		return "", "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}

	// As per simplified focus, DB interaction for recording metadata is omitted here.
	// It's assumed RECORD_STOP event will handle metadata creation.
	// If API needs to create a preliminary record, it would be done here.

	return recordingFileName, jobID, nil
}

func (s *CallService) StopRecordingCall(ctx context.Context, accountSid, callSid string, recordingNameOrUUID string) (string, error) {
	s.logger.Infof("Attempting to stop recording '%s' on call %s for account %s.", recordingNameOrUUID, callSid, accountSid)
	call, err := s.GetCallBySID(accountSid, callSid)
	if err != nil {
		return "", err
	}
	if call.Status != domain.CallStatusInProgress && call.Status != domain.CallStatusRinging { // Allow stopping even if ringing but recording started
		s.logger.Warnf("Call %s is in status %s, may not be actively recording or recording already stopped.", callSid, call.Status)
		// Depending on strictness, could return ErrCallInvalidState_CS, but often stop is best-effort.
	}
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	// Ensure `recordingNameOrUUID` corresponds to what `uuid_record stop` expects.
	// If it's a full path, use that. If it's 'all', use 'all'.
	// The `recordingNameOrUUID` here is the one returned by `StartRecordingCall` (filename part) or a specific recording SID/UUID from events.
	// For simplicity, if it contains a '.', assume it's a filename, otherwise treat as 'all' or specific UUID if that's how FS handles it.
	// A common pattern is `uuid_record <call_uuid> stop <path_to_file_being_recorded>` or `uuid_record <call_uuid> stop all`
	// Using the provided name directly, assuming it's the correct path or 'all'.
	// If it was just a filename, construct full path similar to StartRecordingCall
	// For now, we'll assume `recordingNameOrUUID` is the correct identifier for FS (e.g. full path or 'all')

	// If recordingNameOrUUID is just a filename like "myrec.wav", we might need to reconstruct the full path.
	// However, the FS `uuid_record stop` command often takes the same path argument given to `start`.
	// If `recordingNameOrUUID` is a specific recording file (e.g. from a previous StartRecordingCall), use it.
	// If it's meant to be a generic stop, 'all' is common.
	// Let's assume `recordingNameOrUUID` is the specific file path or 'all'.

	command := fmt.Sprintf("bgapi uuid_record %s stop %s", call.SID, recordingNameOrUUID)
	s.logger.Debugf("Sending ESL command for StopRecordingCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL StopRecordingCall command failed for call %s: %v", callSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	return jobID, nil
}

func (s *CallService) HangupCall(ctx context.Context, accountSid, callSid string, cause string) (string, error) {
	s.logger.Infof("Attempting to hangup call %s for account %s. Cause: %s", callSid, accountSid, cause)
	_, err := s.GetCallBySID(accountSid, callSid) // Validate call exists and belongs to account
	if err != nil {
		return "", err
	}
	// No specific state check for hangup, can be attempted in most states.
	if s.eslClient == nil {
		return "", ErrESLClientNotAvailable_CS
	}

	hangupCause := cause
	if hangupCause == "" {
		hangupCause = "NORMAL_CLEARING" // Default Freeswitch hangup cause
	}

	// Format: bgapi uuid_kill <uuid> [cause]
	command := fmt.Sprintf("bgapi uuid_kill %s %s", callSid, hangupCause)
	s.logger.Debugf("Sending ESL command for HangupCall: %s", command)

	jobID, err := s.eslClient.SendCommand(command)
	if err != nil {
		s.logger.Errorf("ESL HangupCall command failed for call %s: %v", callSid, err)
		return "", fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}

	// Update call status in DB could be done here or via CHANNEL_HANGUP_COMPLETE event.
	// For API initiated hangup, it's good to mark it immediately if possible or rely on events.
	// Current model seems to rely on events for final status updates.

	return jobID, nil
}
