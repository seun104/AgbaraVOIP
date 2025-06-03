package services
import ( "errors"; "fmt"; "strings"; "time"; "net/url";
	"github.com/user/agbaravoip_golang/internal/domain"; "github.com/user/agbaravoip_golang/internal/esl"; 
	"github.com/sirupsen/logrus"; "gorm.io/gorm" )

// Ensure FreeswitchOutboundConfigProvider is defined here or imported if common
type FreeswitchOutboundConfigProvider interface { GetESLOutboundServerListenAddress() string }

// Errors for CallService
var ( ErrCallNotFound_CS = errors.New("call not found"); ErrCallValidationFailed_CS = errors.New("call validation failed");
	ErrCallCreationFailed_CS = errors.New("call creation failed in DB"); ErrESLClientNotAvailable_CS = errors.New("ESL client unavailable");
	ErrESLCommandFailed_CS = errors.New("ESL command failed"); ErrAppLogicError_CS = errors.New("app logic error for call"); )

type CallService struct {
	db *gorm.DB; eslClient *esl.FSInboundClient; appService IApplicationService; 
	logger *logrus.Entry; cfgProvider FreeswitchOutboundConfigProvider
	confService ConferenceService // Added conference service
}
func NewCallService(db *gorm.DB, eslClient *esl.FSInboundClient, appSvc IApplicationService, cfgProvider FreeswitchOutboundConfigProvider, logger *logrus.Logger) *CallService {
	// Initialize conferenceService, assuming NewConferenceService takes *gorm.DB and *logrus.Logger
	// Note: NewConferenceService was defined to take *logrus.Logger, not *logrus.Entry.
	// The CallService logger is *logrus.Entry. We'll pass the base logger.
	conferenceSvc := NewConferenceService(db, logger)
	return &CallService{ db: db, eslClient: eslClient, appService: appSvc, logger: logger.WithField("service", "call"), cfgProvider: cfgProvider, confService: conferenceSvc }
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
