package esl_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	// "net/http/httptest" // Not directly used in these specific integration tests, but useful for sendConferenceCallback if it were tested here
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/esl"
	"github.com/user/agbaravoip_golang/internal/utils"
)

// --- Mocks ---
type MockMinimalCallContext struct {
	mock.Mock
	pendingRec        *callcontrol.PendingRecordInfo
	pendingDials      map[string]*callcontrol.PendingDialInfo
	pendingDialsMutex sync.RWMutex
	nextElementsChan  chan []domain.CallControlElement

	ReturnFsUUID        string
	ReturnAgbaraCallSID string
	ReturnAccountSID    string
	TestLogger          *logrus.Entry
	IsHangup            bool

	currentConfSID         *string
	currentConfName        *string
	currentConfCallbackURL *string
	currentConfCallbackMethod *string
	currentConfParticipantSID *string
	confMutex               sync.RWMutex
	hangupChan				chan struct{}
}

func NewMockMinimalCallContext(t *testing.T, fsUUID, agbaraCallSID, accountSID string) *MockMinimalCallContext {
	testLogger := logrus.New(); testLogger.SetOutput(io.Discard); entry := logrus.NewEntry(testLogger)
	mcc := &MockMinimalCallContext{
		pendingDials:     make(map[string]*callcontrol.PendingDialInfo),
		nextElementsChan: make(chan []domain.CallControlElement, 5),
		hangupChan:       make(chan struct{}),
		ReturnFsUUID:     fsUUID, ReturnAgbaraCallSID: agbaraCallSID, ReturnAccountSID: accountSID, TestLogger: entry,
	}
	mcc.On("Log").Maybe().Return(entry)
	mcc.On("GetUuid").Maybe().Return(agbaraCallSID)
	mcc.On("GetFreeswitchUUID").Maybe().Return(fsUUID)
	mcc.On("GetAccountSid").Maybe().Return(accountSID)
	mcc.On("GetApplicationSid").Maybe().Return("APmockAppSid")
	mcc.On("GetAnswerURL").Maybe().Return("http://mock/answer")
	mcc.On("IsHangupInitiated").Maybe().Run(func(args mock.Arguments){}).Return(func() bool { return mcc.IsHangup })
	mcc.On("SetHangupInitiated").Maybe().Run(func(args mock.Arguments) {
		if !mcc.IsHangup { // Prevent closing already closed channel
			mcc.IsHangup = true
			close(mcc.hangupChan)
		}
	})
	mcc.On("GetNextElementsChannel").Maybe().Return((<-chan []domain.CallControlElement)(mcc.nextElementsChan))
	mcc.On("HangupChan").Maybe().Return((<-chan struct{})(mcc.hangupChan))
	mcc.On("EnterConference", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Run(func(args mock.Arguments){
		mcc.confMutex.Lock(); defer mcc.confMutex.Unlock()
		s1,s2,s3,s4 := args.String(0),args.String(1),args.String(2),args.String(3)
		mcc.currentConfSID = &s1; mcc.currentConfName = &s2; mcc.currentConfCallbackURL = &s3; mcc.currentConfCallbackMethod = &s4
	})
	mcc.On("LeaveConference").Maybe().Run(func(args mock.Arguments){
		mcc.confMutex.Lock(); defer mcc.confMutex.Unlock()
		mcc.currentConfSID, mcc.currentConfName, mcc.currentConfCallbackURL, mcc.currentConfCallbackMethod, mcc.currentConfParticipantSID = nil,nil,nil,nil,nil
	})
	mcc.On("IsInConference").Maybe().Return(func() bool { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); return mcc.currentConfSID != nil })
	mcc.On("GetCurrentConferenceSID").Maybe().Return(func() (string, bool) { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); if mcc.currentConfSID == nil {return "", false}; return *mcc.currentConfSID, true })
	mcc.On("GetCurrentConferenceName").Maybe().Return(func() (string, bool) { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); if mcc.currentConfName == nil {return "", false}; return *mcc.currentConfName, true })
	mcc.On("GetCurrentConferenceCallbackURL").Maybe().Return(func() (string, bool) { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); if mcc.currentConfCallbackURL == nil {return "", false}; return *mcc.currentConfCallbackURL, true })
	mcc.On("GetCurrentConferenceCallbackMethod").Maybe().Return(func() (string, bool) { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); if mcc.currentConfCallbackMethod == nil {return "", false}; return *mcc.currentConfCallbackMethod, true })
	mcc.On("SetCurrentConferenceParticipantSID", mock.AnythingOfType("string")).Maybe().Run(func(args mock.Arguments){ mcc.confMutex.Lock(); defer mcc.confMutex.Unlock(); s:=args.String(0); mcc.currentConfParticipantSID = &s })
	mcc.On("GetCurrentConferenceParticipantSID").Maybe().Return(func() (string, bool) { mcc.confMutex.RLock(); defer mcc.confMutex.RUnlock(); if mcc.currentConfParticipantSID == nil {return "", false}; return *mcc.currentConfParticipantSID, true })
	mcc.On("SetPendingRecording", mock.Anything).Maybe().Run(func(args mock.Arguments){ if args.Get(0) == nil { mcc.pendingRec = nil; return}; if pi,ok := args.Get(0).(callcontrol.PendingRecordInfo); ok { mcc.pendingRec = &pi } else if piPtr, ok := args.Get(0).(*callcontrol.PendingRecordInfo); ok { mcc.pendingRec = piPtr }})
	mcc.On("GetPendingRecording").Maybe().Return(func() (interface{}, bool) { if mcc.pendingRec == nil {return nil,false}; return mcc.pendingRec, true })
	mcc.On("ClearPendingRecording").Maybe().Run(func(args mock.Arguments){ mcc.pendingRec = nil })
	mcc.On("AddPendingDial", mock.AnythingOfType("string"), mock.Anything).Maybe().Run(func(args mock.Arguments){ cUUID := args.String(0); if args.Get(1) == nil {return}; if pi,ok := args.Get(1).(callcontrol.PendingDialInfo); ok { mcc.pendingDialsMutex.Lock(); mcc.pendingDials[cUUID]=&pi; mcc.pendingDialsMutex.Unlock() } else if piPtr, ok := args.Get(1).(*callcontrol.PendingDialInfo); ok { mcc.pendingDialsMutex.Lock(); mcc.pendingDials[cUUID]=piPtr; mcc.pendingDialsMutex.Unlock() }})
	mcc.On("GetPendingDial", mock.AnythingOfType("string")).Maybe().Return(func(cUUID string) (interface{}, bool) { mcc.pendingDialsMutex.RLock(); defer mcc.pendingDialsMutex.RUnlock(); pi,ex := mcc.pendingDials[cUUID]; return pi,ex })
	mcc.On("RemovePendingDial", mock.AnythingOfType("string")).Maybe().Run(func(args mock.Arguments){ cUUID := args.String(0); mcc.pendingDialsMutex.Lock(); delete(mcc.pendingDials, cUUID); mcc.pendingDialsMutex.Unlock() })
	mcc.On("GetCallSID").Maybe().Return(func() *string { s := mcc.ReturnAgbaraCallSID; return &s }())
	mcc.On("GetVariable", mock.AnythingOfType("string")).Maybe().Return("") // Default for GetVariable

	return mcc
}
type MockEslConnectionExecutor struct{ mock.Mock }
func (m *MockEslConnectionExecutor) Execute(cmd string, cmdArgs ...string) (string,error) { callArgs := make([]interface{}, len(cmdArgs)+1); callArgs[0] = cmd; for i, arg := range cmdArgs { callArgs[i+1] = arg }; args := m.Called(callArgs...); return args.String(0),args.Error(1) }
func (m *MockEslConnectionExecutor) ExecuteSofia(cmd string, cmdArgs ...string) (string,error) { callArgs := make([]interface{}, len(cmdArgs)+1); callArgs[0] = cmd; for i, arg := range cmdArgs { callArgs[i+1] = arg }; args := m.Called(callArgs...); return args.String(0),args.Error(1) }
func (m *MockEslConnectionExecutor) SendMsg(msg map[string]string) (string,error) { cArgs:=m.Called(msg); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) GetVar(vName string) (string,error) { cArgs:=m.Called(vName); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Answer() (string,error) { cArgs:=m.Called(); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Hangup(r string) (string,error) { cArgs:=m.Called(r); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) RecordSession(fp string,mDS uint32,sT uint,sH uint) (string,error) { cArgs:=m.Called(fp,mDS,sT,sH); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Originate(dS string,vars map[string]string) (string,error) { cArgs:=m.Called(dS,vars); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) PlayAndGetDigits(min,max,maxAtt int,tO uint32,term,af,iaf,vN string) (string,error) { cArgs:=m.Called(min,max,maxAtt,tO,term,af,iaf,vN); return cArgs.String(0),cArgs.Error(1) }

type MockCallServicerForESL struct{ mock.Mock }
func (m *MockCallServicerForESL) UpdateCallStatus(ctx domain.MinimalCallContext,s string,hC string) error { return m.Called(ctx,s,hC).Error(0) }
func (m *MockCallServicerForESL) CreateRecording(ctx domain.MinimalCallContext,cS *string,rS,fP string,dur uint32,fS string,sB int64) error { return m.Called(ctx,cS,rS,fP,dur,fS,sB).Error(0) }
func (m *MockCallServicerForESL) GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error) { args := m.Called(ctx, sid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Conference), args.Error(1) }
func (m *MockCallServicerForESL) GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error) { args := m.Called(ctx, accountSid, name); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Conference), args.Error(1) }
func (m *MockCallServicerForESL) CreateConference(ctx context.Context, accountSid, name, sid string) (*domain.Conference, error) { args := m.Called(ctx, accountSid, name, sid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Conference), args.Error(1) }
func (m *MockCallServicerForESL) GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) { args := m.Called(ctx, accountSid, name); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Conference), args.Error(1) }
func (m *MockCallServicerForESL) UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error { return m.Called(ctx, sid, status).Error(0) }
func (m *MockCallServicerForESL) EndConference(ctx context.Context, sid string, endTime time.Time) error { return m.Called(ctx, sid, endTime).Error(0) }
func (m *MockCallServicerForESL) AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*domain.ConferenceParticipant, error) { args := m.Called(ctx, confSid, callSid, pSid, accountSid, isMuted, isModerator); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.ConferenceParticipant), args.Error(1) }
func (m *MockCallServicerForESL) GetParticipant(ctx context.Context, pSid string) (*domain.ConferenceParticipant, error) { args := m.Called(ctx, pSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.ConferenceParticipant), args.Error(1) }
func (m *MockCallServicerForESL) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) { args := m.Called(ctx, callSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.ConferenceParticipant), args.Error(1) }
func (m *MockCallServicerForESL) UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error { return m.Called(ctx, pSid, isMuted).Error(0) }
func (m *MockCallServicerForESL) UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error { return m.Called(ctx, pSid, isModerator).Error(0) }
func (m *MockCallServicerForESL) RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error { return m.Called(ctx, pSid, leaveTime).Error(0) }
func (m *MockCallServicerForESL) ListParticipants(ctx context.Context, confSid string) ([]*domain.ConferenceParticipant, error) { args := m.Called(ctx, confSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).([]*domain.ConferenceParticipant), args.Error(1) }

type MockXMLProcessor struct{ mock.Mock }
func (m *MockXMLProcessor) FetchAndParseXML(ctx context.Context,callCtx domain.MinimalCallContext,urlStr,method string,params url.Values) ([]domain.CallControlElement,error) {
	args := m.Called(ctx,callCtx,urlStr,method,params); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).([]domain.CallControlElement), args.Error(1)
}

type TestHarness struct {
	T                *testing.T; Cfg config.Config; LogEntry *logrus.Entry
	MockCallSvc      *MockCallServicerForESL; MockXmlProcessor *MockXMLProcessor
	MockEslExecutor  *MockEslConnectionExecutor
}
func setupIntegrationTest(t *testing.T) *TestHarness {
	logger := logrus.New(); logger.SetOutput(io.Discard); logEntry := logrus.NewEntry(logger)
	return &TestHarness{
		T: t, Cfg: config.Config{}, LogEntry: logEntry,
		MockCallSvc: new(MockCallServicerForESL), MockXmlProcessor: new(MockXMLProcessor), MockEslExecutor: new(MockEslConnectionExecutor),
	}
}

type TestableEventProcessor struct { // Used to test event handling logic from outbound.go
	CallService  domain.CallServicerForESL; XmlProcessor *MockXMLProcessor
	Logger       *logrus.Entry
	// This struct can be expanded to hold other dependencies needed by handleEslEvents logic
}
func (tep *TestableEventProcessor) ProcessEvent(event *eventsocket.Event, callCtx *MockMinimalCallContext, eslExecutor domain.EslConnectionExecutor) {
	// This is a simplified version of the switch from outbound.go's handleEslEvents
	// It directly calls the logic that would be inside the cases.
	eventName := event.Get("Event-Name")
	eventUUID := event.Get("Unique-ID") // FS UUID of the channel event pertains to
	callCtx.Log().Debugf("TestableEventProcessor: Processing ESL Event: %s for UUID: %s", eventName, eventUUID)

	localFreeswitchHangupCauseToDialStatus := map[string]string{
		"NORMAL_CLEARING": "completed", "USER_BUSY": "busy", "NO_ANSWER": "no-answer", "CALL_REJECTED": "failed",
	}

	switch eventName {
	case "CONFERENCE_MAINTENANCE":
		confName := event.Get("Conference-Name"); participantCallFsUUID := event.Get("Caller-Channel-UUID"); eventSubclass := event.Get("Event-Subclass")
		currentConfSID, inConf := callCtx.GetCurrentConferenceSID(); currentConfName, _ := callCtx.GetCurrentConferenceName()
		if !inConf || currentConfName != confName { callCtx.Log().Debugf("Ignoring CONFERENCE_MAINTENANCE for conf '%s', current context conf is '%s'", confName, currentConfName); return }
		var participantCallAgbaraSID string
		if participantCallFsUUID == callCtx.ReturnFsUUID { participantCallAgbaraSID = callCtx.ReturnAgbaraCallSID } else {
			participantCallAgbaraSID = event.Get("Caller-Caller-ID-Number"); if participantCallAgbaraSID == "" { participantCallAgbaraSID = "unknown:" + participantCallFsUUID }
		}
		dbConf, err := tep.CallService.GetConferenceBySID(context.Background(), currentConfSID)
		if err != nil { callCtx.Log().Errorf("ConfEvent: Failed to get conf %s from DB: %v", currentConfSID, err); return }
		params := url.Values{}; params.Set("ConferenceSid", dbConf.SID); params.Set("ConferenceFriendlyName", dbConf.FriendlyName); params.Set("Timestamp", time.Now().UTC().Format(time.RFC3339)); params.Set("EventMemberID", event.Get("Member-ID"))
		switch eventSubclass {
		case "conference::maintenance::add-member":
			params.Set("Event", "participant-join"); var pSID string
			if participantCallFsUUID == callCtx.ReturnFsUUID { pSID, _ = callCtx.GetCurrentConferenceParticipantSID()
			} else {
				part, pErr := tep.CallService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
				if pErr == nil && part != nil { pSID = part.SID } else { newPSID := utils.GenerateSID("CP_test_"); _, addErr := tep.CallService.AddParticipant(context.Background(), dbConf.SID, participantCallAgbaraSID, newPSID, callCtx.ReturnAccountSID, false, false); if addErr == nil { pSID = newPSID } else { pSID = "unknown_add_err" }}
			}
			params.Set("ParticipantSid", pSID); params.Set("CallSid", participantCallAgbaraSID)
		case "conference::maintenance::del-member":
			params.Set("Event", "participant-leave")
			part, pErr := tep.CallService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
			if pErr == nil && part != nil { _ = tep.CallService.RemoveParticipant(context.Background(), part.SID, time.Now().UTC()); params.Set("ParticipantSid", part.SID) } else { params.Set("ParticipantSid", "unknown") }
			params.Set("CallSid", participantCallAgbaraSID); if participantCallFsUUID == callCtx.ReturnFsUUID { callCtx.LeaveConference() }
		case "conference::maintenance::mute-member", "conference::maintenance::unmute-member":
			params.Set("Event", "participant-mute-update")
			part, pErr := tep.CallService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
			if pErr == nil && part != nil { newMuteState := eventSubclass == "conference::maintenance::mute-member"; _ = tep.CallService.UpdateParticipantMuteStatus(context.Background(), part.SID, newMuteState); params.Set("ParticipantSid", part.SID); params.Set("Muted", fmt.Sprintf("%t", newMuteState)) } else {params.Set("ParticipantSid", "unknown")}
			params.Set("CallSid", participantCallAgbaraSID)
		case "conference::maintenance::start-talking", "conference::maintenance::stop-talking":
			params.Set("Event", "participant-talk-status"); part, pErr := tep.CallService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
			if pErr == nil && part != nil { params.Set("ParticipantSid", part.SID) } else { params.Set("ParticipantSid", "unknown") }
			params.Set("CallSid", participantCallAgbaraSID); params.Set("Talking", fmt.Sprintf("%t", eventSubclass == "conference::maintenance::start-talking"))
		case "conference::maintenance::end":
			params.Set("Event", "conference-end"); if tep.CallService != nil { _ = tep.CallService.EndConference(context.Background(), dbConf.SID, time.Now().UTC()) }; callCtx.LeaveConference()
		default: callCtx.Log().Debugf("Unhandled conference event subclass: %s", eventSubclass); return
		}
		cbURL, hasCB := callCtx.GetCurrentConferenceCallbackURL(); cbMethod, _ := callCtx.GetCurrentConferenceCallbackMethod()
		if hasCB && cbURL != "" { callCtx.Log().Infof("Simulating conference callback to %s with params %v", cbURL, params) /* Actual sendConferenceCallback call omitted for test simplicity */ }

	case "DTMF":
		digit := event.Get("DTMF-Digit")
		if recInfoInter, recExists := callCtx.GetPendingRecording(); recExists && recInfoInter != nil {
			recInfo := recInfoInter.(*callcontrol.PendingRecordInfo)
			if recInfo.OriginalElement.FinishOnKey != "" && strings.Contains(recInfo.OriginalElement.FinishOnKey, digit) {
				callCtx.Log().Infof("FinishOnKey '%s' received. Stopping recording %s.", digit, recInfo.ExpectedFilePath)
				_, err := eslExecutor.Execute("uuid_record", callCtx.GetFreeswitchUUID(), "stop", recInfo.ExpectedFilePath)
				if err != nil { callCtx.Log().Errorf("Error stopping recording %s on FinishOnKey: %v", recInfo.ExpectedFilePath, err) }
			}
		}
	// Add cases for RECORD_STOP, CHANNEL_ANSWER (B-leg), CHANNEL_HANGUP (B-leg) here if testing them via this helper
	}
}

func TestHandleEslEvents_Conference_AddMember(t *testing.T) {
	h := setupIntegrationTest(t)
	confSID := "CFtestconf1"; confName := "MyTestRoom"; participantAgbaraSID := "CAparticipant1"; participantFsUUID := "fs-uuid-participant1"
	otherMemberAgbaraSID := "CAothermember"; otherMemberFsUUID := "fs-uuid-othermember"; otherMemberPSID := "CPothermember"; accSID := "ACintegration"; callbackURL := "http://example.com/conf_events"
	mockCallCtx := NewMockMinimalCallContext(t, participantFsUUID, participantAgbaraSID, accSID)
	mockCallCtx.EnterConference(confSID, confName, callbackURL, "POST")
	mockConference := &domain.Conference{SID: confSID, FriendlyName: confName, AccountSID: accSID}
	h.MockCallSvc.On("GetConferenceBySID", mock.Anything, confSID).Return(mockConference, nil)
	h.MockCallSvc.On("GetParticipantByCallSID", mock.Anything, otherMemberAgbaraSID).Return(nil, domain.ErrNotFound).Once()
	h.MockCallSvc.On("AddParticipant", mock.Anything, confSID, otherMemberAgbaraSID, mock.AnythingOfType("string"), accSID, false, false).Return(&domain.ConferenceParticipant{SID: otherMemberPSID}, nil).Once()
	eventData := map[string]string{ "Event-Name": "CONFERENCE_MAINTENANCE", "Event-Subclass": "conference::maintenance::add-member", "Conference-Name": confName, "Caller-Channel-UUID": otherMemberFsUUID, "Caller-Caller-ID-Number": otherMemberAgbaraSID }
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService:  h.MockCallSvc, Logger: h.LogEntry }
	mockCallCtx.On("GetCurrentConferenceSID").Return(confSID, true); mockCallCtx.On("GetCurrentConferenceName").Return(confName, true)
	mockCallCtx.On("GetCurrentConferenceCallbackURL").Return(callbackURL, true); mockCallCtx.On("GetCurrentConferenceCallbackMethod").Return("POST", true)
	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)
	h.MockCallSvc.AssertExpectations(t); mockCallCtx.AssertExpectations(t)
}

func TestHandleEslEvents_Conference_DelMember_CurrentLeg(t *testing.T) {
	h := setupIntegrationTest(t); confSID := "CFtestconfDel"; confName := "MyRoomToLeave"; participantAgbaraSID := "CAparticipantCurrent"; participantFsUUID := "fs-uuid-currentLeg"; participantPSID := "CPcurrentLeg"; accSID := "ACintegrationDel"; callbackURL := "http://example.com/conf_events_del"
	mockCallCtx := NewMockMinimalCallContext(t, participantFsUUID, participantAgbaraSID, accSID)
	mockCallCtx.EnterConference(confSID, confName, callbackURL, "GET"); mockCallCtx.SetCurrentConferenceParticipantSID(participantPSID)
	mockConference := &domain.Conference{SID: confSID, FriendlyName: confName, AccountSID: accSID}
	h.MockCallSvc.On("GetConferenceBySID", mock.Anything, confSID).Return(mockConference, nil)
	mockedParticipant := &domain.ConferenceParticipant{SID: participantPSID, CallSID: participantAgbaraSID, ConferenceSID: confSID}
	h.MockCallSvc.On("GetParticipantByCallSID", mock.Anything, participantAgbaraSID).Return(mockedParticipant, nil).Once()
	h.MockCallSvc.On("RemoveParticipant", mock.Anything, participantPSID, mock.AnythingOfType("time.Time")).Return(nil).Once()
	mockCallCtx.On("GetCurrentConferenceSID").Return(confSID, true); mockCallCtx.On("GetCurrentConferenceName").Return(confName, true)
	mockCallCtx.On("GetCurrentConferenceCallbackURL").Return(callbackURL, true); mockCallCtx.On("GetCurrentConferenceCallbackMethod").Return("GET", true)
	mockCallCtx.On("LeaveConference").Return().Once()
	eventData := map[string]string{ "Event-Name": "CONFERENCE_MAINTENANCE", "Event-Subclass": "conference::maintenance::del-member", "Conference-Name": confName, "Caller-Channel-UUID": participantFsUUID, "Caller-Caller-ID-Number": participantAgbaraSID }
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService:  h.MockCallSvc, Logger: h.LogEntry }
	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)
	h.MockCallSvc.AssertExpectations(t); mockCallCtx.AssertExpectations(t)
}

func TestHandleEslEvents_Conference_MuteUnmute(t *testing.T) {
	h := setupIntegrationTest(t); confSID := "CFtestconfMute"; confName := "MyRoomToMute"; participantAgbaraSID := "CAparticipantMute"; participantFsUUID := "fs-uuid-muteLeg"; participantPSID := "CPmuteLeg"; accSID := "ACintegrationMute"; callbackURL := "http://example.com/conf_events_mute"
	mockCallCtx := NewMockMinimalCallContext(t, participantFsUUID, participantAgbaraSID, accSID)
	mockCallCtx.EnterConference(confSID, confName, callbackURL, "POST"); mockCallCtx.SetCurrentConferenceParticipantSID(participantPSID)
	mockConference := &domain.Conference{SID: confSID, FriendlyName: confName, AccountSID: accSID}
	h.MockCallSvc.On("GetConferenceBySID", mock.Anything, confSID).Return(mockConference, nil).Times(2)
	mockedParticipant := &domain.ConferenceParticipant{SID: participantPSID, CallSID: participantAgbaraSID, ConferenceSID: confSID, IsMuted: false}
	h.MockCallSvc.On("GetParticipantByCallSID", mock.Anything, participantAgbaraSID).Return(mockedParticipant, nil).Times(2)
	h.MockCallSvc.On("UpdateParticipantMuteStatus", mock.Anything, participantPSID, true).Return(nil).Once()
	h.MockCallSvc.On("UpdateParticipantMuteStatus", mock.Anything, participantPSID, false).Return(nil).Once()
	mockCallCtx.On("GetCurrentConferenceSID").Return(confSID, true).Times(2); mockCallCtx.On("GetCurrentConferenceName").Return(confName, true).Times(2)
	mockCallCtx.On("GetCurrentConferenceCallbackURL").Return(callbackURL, true).Times(2); mockCallCtx.On("GetCurrentConferenceCallbackMethod").Return("POST", true).Times(2)
	eventProcessor := &TestableEventProcessor{ CallService:  h.MockCallSvc, Logger: h.LogEntry }
	muteEventData := map[string]string{ "Event-Name": "CONFERENCE_MAINTENANCE", "Event-Subclass": "conference::maintenance::mute-member", "Conference-Name": confName, "Member-ID": "1", "Caller-Channel-UUID": participantFsUUID, "Caller-Caller-ID-Number": participantAgbaraSID }
	eventProcessor.ProcessEvent(&eventsocket.Event{Header: muteEventData}, mockCallCtx, h.MockEslExecutor)
	unmuteEventData := map[string]string{ "Event-Name": "CONFERENCE_MAINTENANCE", "Event-Subclass": "conference::maintenance::unmute-member", "Conference-Name": confName, "Member-ID": "1", "Caller-Channel-UUID": participantFsUUID, "Caller-Caller-ID-Number": participantAgbaraSID }
	eventProcessor.ProcessEvent(&eventsocket.Event{Header: unmuteEventData}, mockCallCtx, h.MockEslExecutor)
	h.MockCallSvc.AssertExpectations(t); mockCallCtx.AssertExpectations(t)
}

func TestHandleEslEvents_Conference_End(t *testing.T) {
	h := setupIntegrationTest(t); confSID := "CFtestconfEnd"; confName := "MyRoomToEnd"; participantAgbaraSID := "CAparticipantEnd"; participantFsUUID := "fs-uuid-endLeg"; accSID := "ACintegrationEnd"; callbackURL := "http://example.com/conf_events_end"
	mockCallCtx := NewMockMinimalCallContext(t, participantFsUUID, participantAgbaraSID, accSID)
	mockCallCtx.EnterConference(confSID, confName, callbackURL, "POST")
	mockConference := &domain.Conference{SID: confSID, FriendlyName: confName, AccountSID: accSID}
	h.MockCallSvc.On("GetConferenceBySID", mock.Anything, confSID).Return(mockConference, nil).Once()
	h.MockCallSvc.On("EndConference", mock.Anything, confSID, mock.AnythingOfType("time.Time")).Return(nil).Once()
	mockCallCtx.On("GetCurrentConferenceSID").Return(confSID, true).Once(); mockCallCtx.On("GetCurrentConferenceName").Return(confName, true).Once()
	mockCallCtx.On("GetCurrentConferenceCallbackURL").Return(callbackURL, true).Once(); mockCallCtx.On("GetCurrentConferenceCallbackMethod").Return("POST", true).Once()
	mockCallCtx.On("LeaveConference").Return().Once()
	eventData := map[string]string{ "Event-Name": "CONFERENCE_MAINTENANCE", "Event-Subclass": "conference::maintenance::end", "Conference-Name": confName }
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService:  h.MockCallSvc, Logger: h.LogEntry }
	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)
	h.MockCallSvc.AssertExpectations(t); mockCallCtx.AssertExpectations(t)
}

// Placeholder for other integration tests
func TestHandleEslEvents_RECORD_STOP_NoActionURL(t *testing.T) {
	h := setupIntegrationTest(t)
	fsUUID := "fs-uuid-recordstop-noaction"
	agbaraSID := "CA-recordstop-noaction"
	accSID := "AC-recordstop-noaction"
	mockCallCtx := NewMockMinimalCallContext(t, fsUUID, agbaraSID, accSID)

	recordElem := &domain.RecordElement{ActionURL: "", FileFormat: "mp3"} // No ActionURL
	pendingRec := callcontrol.PendingRecordInfo{
		OriginalElement:  recordElem,
		ExpectedFilePath: "/tmp/rec_no_action.mp3",
	}
	mockCallCtx.pendingRec = &pendingRec

	mockCallCtx.On("GetPendingRecording").Return(mockCallCtx.pendingRec, true).Once()
	mockCallCtx.On("ClearPendingRecording").Return().Once()
	agbaraSIDPtr := agbaraSID // Need a pointer for *string arg
	h.MockCallSvc.On("CreateRecording", mockCallCtx, &agbaraSIDPtr, mock.AnythingOfType("string"), "/tmp/rec_no_action.mp3", uint32(15), "mp3", int64(0)).Return(nil).Once()
	// No call to xmlProcessor expected

	eventData := map[string]string{
		"Event-Name":                "RECORD_STOP",
		"variable_record_file_path": "/tmp/rec_no_action.mp3",
		"variable_record_seconds":   "15",
	}
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService: h.MockCallSvc, Logger: h.LogEntry, XmlProcessor: h.MockXmlProcessor }

	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)

	h.MockCallSvc.AssertExpectations(t)
	mockCallCtx.AssertExpectations(t)
	h.MockXmlProcessor.AssertNotCalled(t, "FetchAndParseXML", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleEslEvents_RECORD_STOP_WithActionURL_Success(t *testing.T) {
	h := setupIntegrationTest(t)
	fsUUID := "fs-uuid-recordstop-action"
	agbaraSID := "CA-recordstop-action"
	accSID := "AC-recordstop-action"
	actionURL := "http://example.com/record_callback"
	mockCallCtx := NewMockMinimalCallContext(t, fsUUID, agbaraSID, accSID)

	recordElem := &domain.RecordElement{ActionURL: actionURL, Method: "POST", FileFormat: "wav"}
	pendingRec := callcontrol.PendingRecordInfo{
		OriginalElement:  recordElem,
		ExpectedFilePath: "/tmp/rec_with_action.wav",
	}
	mockCallCtx.pendingRec = &pendingRec

	mockCallCtx.On("GetPendingRecording").Return(mockCallCtx.pendingRec, true).Once()
	mockCallCtx.On("ClearPendingRecording").Return().Once()
	agbaraSIDPtr := agbaraSID
	h.MockCallSvc.On("CreateRecording", mockCallCtx, &agbaraSIDPtr, mock.AnythingOfType("string"), "/tmp/rec_with_action.wav", uint32(20), "wav", int64(0)).Return(nil).Once()

	expectedParams := url.Values{}
	expectedParams.Set("RecordingSid", mock.Anything) // SID is generated, cannot predict exact value
	expectedParams.Set("RecordingDuration", "20")
	expectedParams.Set("RecordingUrl", "/tmp/rec_with_action.wav")
	expectedParams.Set("CallSid", agbaraSID)
	expectedParams.Set("AccountSid", accSID)

	dummyElements := []domain.CallControlElement{&domain.Say{Text: "Callback received"}}
	h.MockXmlProcessor.On("FetchAndParseXML", mock.Anything, mockCallCtx, actionURL, "POST", mock.MatchedBy(func(v url.Values) bool {
		return v.Get("RecordingDuration") == "20" && v.Get("CallSid") == agbaraSID // Check a few key params
	})).Return(dummyElements, nil).Once()

	mockCallCtx.On("SendNextElements", dummyElements).Return(nil).Once()


	eventData := map[string]string{
		"Event-Name":                "RECORD_STOP",
		"variable_record_file_path": "/tmp/rec_with_action.wav",
		"variable_record_seconds":   "20",
	}
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService: h.MockCallSvc, Logger: h.LogEntry, XmlProcessor: h.MockXmlProcessor }

	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)

	h.MockCallSvc.AssertExpectations(t)
	h.MockXmlProcessor.AssertExpectations(t)
	mockCallCtx.AssertExpectations(t)
}

func TestHandleEslEvents_CHANNEL_ANSWER_Bleg_WithActionURL(t *testing.T) {
	h := setupIntegrationTest(t)
	fsUUID_Aleg := "fs-uuid-aleg-ans"
	agbaraSID_Aleg := "CA-aleg-ans"
	accSID := "AC-ans"

	bLegFSUUID := "fs-uuid-bleg-ans"
	actionURL := "http://example.com/dial_answered"

	mockCallCtx := NewMockMinimalCallContext(t, fsUUID_Aleg, agbaraSID_Aleg, accSID)

	// Setup PendingDialInfo for the B-leg
	dialElem := &domain.DialElement{ActionURL: actionURL, Method: "POST"}
	pendingDial := callcontrol.PendingDialInfo{
		OriginalElement:     dialElem,
		ParentAgbaraCallSID: agbaraSID_Aleg,
	}
	mockCallCtx.pendingDials[bLegFSUUID] = &pendingDial // Manually set for test

	mockCallCtx.On("GetPendingDial", bLegFSUUID).Return(&pendingDial, true).Once()
	mockCallCtx.On("GetFreeswitchUUID").Return(fsUUID_Aleg).Once() // For uuid_bridge
	mockCallCtx.On("GetVariable", "agbara_dial_send_digits_on_answer").Return("123ww").Once() // Simulate sendDigits

	h.MockEslExecutor.On("Execute", "uuid_bridge", fsUUID_Aleg, bLegFSUUID).Return("OK", nil).Once()
	h.MockEslExecutor.On("Execute", "uuid_send_dtmf", bLegFSUUID, "123ww").Return("OK", nil).Once()

	dummyElements := []domain.CallControlElement{&domain.Hangup{}} // Example callback XML
	h.MockXmlProcessor.On("FetchAndParseXML", mock.Anything, mockCallCtx, actionURL, "POST", mock.MatchedBy(func(v url.Values) bool {
		return v.Get("DialCallStatus") == "answered" && v.Get("DialBlegUuid") == bLegFSUUID
	})).Return(dummyElements, nil).Once()
	mockCallCtx.On("SendNextElements", dummyElements).Return(nil).Once()

	eventData := map[string]string{
		"Event-Name":  "CHANNEL_ANSWER",
		"Unique-ID":   bLegFSUUID, // Event is for the B-leg
		// variable_agbara_dial_send_digits_on_answer is retrieved from A-leg context, not event for B-leg here.
		// However, the event might carry its own variables if set during originate.
		// For this test, GetVariable on mockCallCtx will provide it.
	}
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService: h.MockCallSvc, Logger: h.LogEntry, XmlProcessor: h.MockXmlProcessor }

	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)

	h.MockEslExecutor.AssertExpectations(t)
	h.MockXmlProcessor.AssertExpectations(t)
	mockCallCtx.AssertExpectations(t)
}

func TestHandleEslEvents_CHANNEL_HANGUP_Bleg_WithActionURL(t *testing.T) {
	h := setupIntegrationTest(t)
	fsUUID_Aleg := "fs-uuid-aleg-hup"
	agbaraSID_Aleg := "CA-aleg-hup"
	accSID := "AC-hup"
	bLegFSUUID := "fs-uuid-bleg-hup"
	actionURL := "http://example.com/dial_hangup"

	mockCallCtx := NewMockMinimalCallContext(t, fsUUID_Aleg, agbaraSID_Aleg, accSID)

	dialElem := &domain.DialElement{ActionURL: actionURL, Method: "GET"}
	pendingDial := callcontrol.PendingDialInfo{
		OriginalElement:     dialElem,
		ParentAgbaraCallSID: agbaraSID_Aleg,
	}
	mockCallCtx.pendingDials[bLegFSUUID] = &pendingDial

	mockCallCtx.On("GetPendingDial", bLegFSUUID).Return(&pendingDial, true).Once()
	mockCallCtx.On("RemovePendingDial", bLegFSUUID).Return().Once()

	dummyElements := []domain.CallControlElement{&domain.Log{Message: "Dial hangup processed"}}
	h.MockXmlProcessor.On("FetchAndParseXML", mock.Anything, mockCallCtx, actionURL, "GET", mock.MatchedBy(func(v url.Values) bool {
		return v.Get("DialCallStatus") == "busy" && v.Get("DialBlegUuid") == bLegFSUUID && v.Get("DialHangupCause") == "USER_BUSY"
	})).Return(dummyElements, nil).Once()
	mockCallCtx.On("SendNextElements", dummyElements).Return(nil).Once()

	eventData := map[string]string{
		"Event-Name":   "CHANNEL_HANGUP",
		"Unique-ID":    bLegFSUUID,
		"Hangup-Cause": "USER_BUSY",
	}
	event := eventsocket.Event{Header: eventData}
	eventProcessor := &TestableEventProcessor{ CallService: h.MockCallSvc, Logger: h.LogEntry, XmlProcessor: h.MockXmlProcessor }

	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)

	h.MockXmlProcessor.AssertExpectations(t)
	mockCallCtx.AssertExpectations(t)
}


func TestHandleEslEvents_DTMF_FinishOnKey_Recording(t *testing.T) {
	h := setupIntegrationTest(t)

	fsUUID := "fs-uuid-dtmf-rec"
	agbaraSID := "CA-dtmf-rec"
	accSID := "AC-dtmf-rec"
	mockCallCtx := NewMockMinimalCallContext(t, fsUUID, agbaraSID, accSID)

	mockCallCtx.On("GetFreeswitchUUID").Return(fsUUID) // Needed by the DTMF handler logic in TestableEventProcessor

	recordElem := &domain.RecordElement{FinishOnKey: "#", FileFormat: "wav"}
	pendingRec := callcontrol.PendingRecordInfo{
		OriginalElement:  recordElem,
		ExpectedFilePath: "/tmp/dtmf_rec.wav",
	}
	mockCallCtx.pendingRec = &pendingRec // Directly set for test simplicity

	mockCallCtx.On("GetPendingRecording").Return(mockCallCtx.pendingRec, true).Once()
	h.MockEslExecutor.On("Execute", "uuid_record", fsUUID, "stop", "/tmp/dtmf_rec.wav").Return("OK", nil).Once()

	eventData := map[string]string{
		"Event-Name":      "DTMF",
		"DTMF-Digit":      "#",
		"Unique-ID":       fsUUID,
	}
	event := eventsocket.Event{Header: eventData}

	eventProcessor := &TestableEventProcessor{
		CallService:  h.MockCallSvc,
		XmlProcessor: h.MockXmlProcessor,
		Logger:       h.LogEntry,
	}

	eventProcessor.ProcessEvent(&event, mockCallCtx, h.MockEslExecutor)

	h.MockEslExecutor.AssertExpectations(t)
	mockCallCtx.AssertExpectations(t)
}

func TestMain(m *testing.M) {
	m.Run()
}
