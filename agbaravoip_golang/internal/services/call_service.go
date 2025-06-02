package services

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"net/url" 

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/esl" 
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ErrCallNotFound_CS          = errors.New("call not found in CallService") // To avoid conflict if errors are not shared
	ErrCallValidationFailed_CS  = errors.New("call validation failed in CallService")
	ErrCallCreationFailed_CS    = errors.New("call creation failed in DB in CallService")
	ErrESLClientNotAvailable_CS = errors.New("ESL client is not available for call origination in CallService")
	ErrESLCommandFailed_CS      = errors.New("ESL command failed during call origination in CallService")
	ErrAppLogicError_CS         = errors.New("application logic error for call in CallService")
)

type FreeswitchOutboundConfigProvider interface {
    GetESLOutboundServerListenAddress() string
}

type CallService struct {
	db         *gorm.DB
	eslClient  *esl.FSInboundClient 
	appService IApplicationService  
	logger     *logrus.Entry
	config     FreeswitchOutboundConfigProvider 
}

func NewCallService(db *gorm.DB, eslClient *esl.FSInboundClient, appSvc IApplicationService, cfgProvider FreeswitchOutboundConfigProvider, logger *logrus.Logger) *CallService {
	return &CallService{
		db:         db,
		eslClient:  eslClient,
		appService: appSvc,
		logger:     logger.WithField("service", "call"),
		config:     cfgProvider,
	}
}

func (s *CallService) OriginateCall(accountSid string, appSidOrNil *string, fromNum string, toNum string, answerURL string, timeoutSeconds *int) (*domain.Call, error) {
	s.logger.Debugf("OriginateCall for AccountSID: %s, From: %s, To: %s, AnswerURL: %s, AppSID: %v",
		accountSid, fromNum, toNum, answerURL, appSidOrNil)

	if s.eslClient == nil {
		s.logger.Error("ESL Inbound Client is not initialized in CallService")
		return nil, ErrESLClientNotAvailable_CS
	}

	finalAnswerURL := answerURL
	effectiveApplicationSID := appSidOrNil

	if appSidOrNil != nil && *appSidOrNil != "" {
		if strings.TrimSpace(finalAnswerURL) == "" {
			app, err := s.appService.GetApplicationBySID(accountSid, *appSidOrNil)
			if err != nil {
				s.logger.Warnf("Failed to fetch application %s for call origination: %v", *appSidOrNil, err)
				return nil, fmt.Errorf("%w: invalid application_sid: %v", ErrAppLogicError_CS, err)
			}
			if app.VoiceURL == "" {
				s.logger.Warnf("Application %s has no VoiceURL configured", *appSidOrNil)
				return nil, fmt.Errorf("%w: application %s has no VoiceURL", ErrAppLogicError_CS, *appSidOrNil)
			}
			finalAnswerURL = app.VoiceURL
		}
	} else {
		effectiveApplicationSID = nil 
	}
	
	if strings.TrimSpace(finalAnswerURL) == "" {
		return nil, fmt.Errorf("%w: answer_url or valid application_sid is required", ErrCallValidationFailed_CS)
	}

	now := time.Now().UTC()
	call := &domain.Call{
		AccountSID:     accountSid, ApplicationSID: effectiveApplicationSID, FromNum: fromNum, ToNum: toNum,
		AnswerURL:      finalAnswerURL, Status: domain.CallStatusQueued, Direction: domain.CallDirectionOutboundAPI,
		StartTime:      &now, TimeoutSeconds: timeoutSeconds,
	}
	
	if err := s.db.Create(call).Error; err != nil {
		s.logger.Errorf("Error creating call record: %v", err); return nil, fmt.Errorf("%w: %v", ErrCallCreationFailed_CS, err)
	}
	s.logger.Infof("Call record %s created, status %s", call.SID, call.Status)
	
	escapedAnswerURL := url.QueryEscape(finalAnswerURL)
	vars := fmt.Sprintf("origination_caller_id_number=%s,origination_uuid=%s,agbara_account_sid=%s,agbara_call_sid=%s,agbara_answer_url=%s",
		fromNum, call.SID, accountSid, call.SID, escapedAnswerURL)
	if call.TimeoutSeconds != nil && *call.TimeoutSeconds > 0 { vars += fmt.Sprintf(",originate_timeout=%d", *call.TimeoutSeconds) }
	
	targetDialstring := fmt.Sprintf("user/%s", toNum) 
	eslOutboundApp := fmt.Sprintf("&socket(%s async full)", s.config.GetESLOutboundServerListenAddress())
	originateCmd := fmt.Sprintf("originate {%s}%s %s", vars, targetDialstring, eslOutboundApp)
	
	s.logger.Infof("Sending originate: bgapi %s", originateCmd)
	_, err := s.eslClient.SendCommand(fmt.Sprintf("bgapi %s", originateCmd))
	
	callUpdate := map[string]interface{}{}
	if err != nil {
		s.logger.Errorf("Originate failed for Call SID %s: %v", call.SID, err)
		callUpdate["status"] = domain.CallStatusFailed; callUpdate["hangup_cause"] = "ORIGINATE_ESL_ERROR"; callUpdate["end_time"] = time.Now().UTC()
		if errDb := s.db.Model(call).Updates(callUpdate).Error; errDb != nil { s.logger.Errorf("Failed to update call %s to failed: %v", call.SID, errDb) }
		return call, fmt.Errorf("%w: %v", ErrESLCommandFailed_CS, err)
	}
	callUpdate["status"] = domain.CallStatusInitiated 
	if errDb := s.db.Model(call).Updates(callUpdate).Error; errDb != nil { s.logger.Errorf("Failed to update call %s to initiated: %v", call.SID, errDb) }
	s.logger.Infof("Originate sent for Call SID %s. Status: %s.", call.SID, call.Status)
	return call, nil
}

func (s *CallService) GetCallBySID(accountSid string, callSid string) (*domain.Call, error) {
	var call domain.Call
	if err := s.db.Where("sid = ? AND account_sid = ?", callSid, accountSid).First(&call).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrCallNotFound_CS }
		return nil, fmt.Errorf("db error: %w", err)
	}
	return &call, nil
}

func (s *CallService) ListCalls(accountSid string, filters map[string]interface{}) ([]*domain.Call, error) {
	var calls []*domain.Call; query := s.db.Where("account_sid = ?", accountSid)
	for key, value := range filters {
		if strVal, ok := value.(string); ok && strVal != "" {
			switch key { // Use known DB column names directly
			case "status", "from_num", "to_num", "direction": query = query.Where(fmt.Sprintf("%s = ?", key), strVal)
			default: s.logger.Warnf("Unsupported filter key: %s", key)
			}
		}
	}
	if err := query.Order("created_at desc").Find(&calls).Error; err != nil { return nil, fmt.Errorf("db error: %w", err) }
	return calls, nil
}


