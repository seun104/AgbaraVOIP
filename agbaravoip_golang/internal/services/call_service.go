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
}
func NewCallService(db *gorm.DB, eslClient *esl.FSInboundClient, appSvc IApplicationService, cfgProvider FreeswitchOutboundConfigProvider, logger *logrus.Logger) *CallService {
	return &CallService{ db: db, eslClient: eslClient, appService: appSvc, logger: logger.WithField("service", "call"), cfgProvider: cfgProvider, }
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

