package domain_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/domain"
)

// --- MockMinimalCallContext ---
type MockMinimalCallContext struct {
	mock.Mock
}

func (m *MockMinimalCallContext) Log() *logrus.Entry {
	args := m.Called()
	if args.Get(0) == nil { entry := logrus.NewEntry(logrus.New()); entry.Logger.SetOutput(io.Discard); return entry }
	return args.Get(0).(*logrus.Entry)
}
func (m *MockMinimalCallContext) GetUuid() string             { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAccountSid() string       { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetApplicationSid() string   { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAnswerURL() string        { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetVariable(v string) string { return m.Called(v).String(0) }
func (m *MockMinimalCallContext) IsHangupInitiated() bool     { return m.Called().Bool(0) }
func (m *MockMinimalCallContext) SetHangupInitiated()         { m.Called() }
func (m *MockMinimalCallContext) SetPendingRecording(i interface{}) { m.Called(i) }
func (m *MockMinimalCallContext) GetPendingRecording() (interface{}, bool) {
	args := m.Called(); if args.Get(0) == nil { return nil, args.Bool(1) }; return args.Get(0), args.Bool(1)
}
func (m *MockMinimalCallContext) ClearPendingRecording() { m.Called() }
func (m *MockMinimalCallContext) AddPendingDial(cUuid string, i interface{}) { m.Called(cUuid, i) }
func (m *MockMinimalCallContext) GetPendingDial(cUuid string) (interface{}, bool) {
	args := m.Called(cUuid); if args.Get(0) == nil { return nil, args.Bool(1) }; return args.Get(0), args.Bool(1)
}
func (m *MockMinimalCallContext) RemovePendingDial(cUuid string) { m.Called(cUuid) }
func (m *MockMinimalCallContext) GetAllPendingDials() map[string]interface{} {
	args := m.Called(); if args.Get(0) == nil { return nil }; return args.Get(0).(map[string]interface{})
}
func (m *MockMinimalCallContext) SendNextElements(e []domain.CallControlElement) error { return m.Called(e).Error(0) }
func (m *MockMinimalCallContext) GetNextElementsChannel() <-chan []domain.CallControlElement {
	args := m.Called(); if args.Get(0) == nil { ch := make(chan []domain.CallControlElement); close(ch); return ch }; return args.Get(0).(<-chan []domain.CallControlElement)
}
func (m *MockMinimalCallContext) HangupChan() <-chan struct{} {
	args := m.Called(); if args.Get(0) == nil { ch := make(chan struct{}); return ch }; return args.Get(0).(<-chan struct{})
}
func (m *MockMinimalCallContext) EnterConference(confSID, confName, cbURL, cbMethod string) { m.Called(confSID, confName, cbURL, cbMethod) }
func (m *MockMinimalCallContext) LeaveConference()                                       { m.Called() }
func (m *MockMinimalCallContext) IsInConference() bool                                   { return m.Called().Bool(0) }
func (m *MockMinimalCallContext) GetCurrentConferenceSID() (string, bool) {
	args := m.Called(); return args.String(0), args.Bool(1)
}
func (m *MockMinimalCallContext) GetCurrentConferenceName() (string, bool) {
	args := m.Called(); return args.String(0), args.Bool(1)
}
func (m *MockMinimalCallContext) GetCurrentConferenceCallbackURL() (string, bool) {
	args := m.Called(); return args.String(0), args.Bool(1)
}
func (m *MockMinimalCallContext) GetCurrentConferenceCallbackMethod() (string, bool) {
	args := m.Called(); return args.String(0), args.Bool(1)
}
func (m *MockMinimalCallContext) SetCurrentConferenceParticipantSID(pSID string) { m.Called(pSID) }
func (m *MockMinimalCallContext) GetCurrentConferenceParticipantSID() (string, bool) {
	args := m.Called(); return args.String(0), args.Bool(1)
}
func (m *MockMinimalCallContext) GetFreeswitchUUID() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetCallSID() *string {
    args := m.Called(); if args.Get(0) == nil { return nil }; return args.Get(0).(*string)
}

// --- MockEslConnectionExecutor ---
type MockEslConnectionExecutor struct{ mock.Mock }
func (m *MockEslConnectionExecutor) Execute(cmd string, cmdArgs ...string) (string,error) {
	callArgs := make([]interface{}, len(cmdArgs)+1); callArgs[0] = cmd
	for i, arg := range cmdArgs { callArgs[i+1] = arg }
	args := m.Called(callArgs...); return args.String(0),args.Error(1)
}
func (m *MockEslConnectionExecutor) ExecuteSofia(cmd string, cmdArgs ...string) (string,error) { callArgs := make([]interface{}, len(cmdArgs)+1); callArgs[0] = cmd; for i, arg := range cmdArgs { callArgs[i+1] = arg }; args := m.Called(callArgs...); return args.String(0),args.Error(1) }
func (m *MockEslConnectionExecutor) SendMsg(msg map[string]string) (string,error) { cArgs:=m.Called(msg); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) GetVar(vName string) (string,error) { cArgs:=m.Called(vName); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Answer() (string,error) { cArgs:=m.Called(); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Hangup(r string) (string,error) { cArgs:=m.Called(r); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) RecordSession(fp string,mDS uint32,sT uint,sH uint) (string,error) { cArgs:=m.Called(fp,mDS,sT,sH); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) Originate(dS string,vars map[string]string) (string,error) { cArgs:=m.Called(dS,vars); return cArgs.String(0),cArgs.Error(1) }
func (m *MockEslConnectionExecutor) PlayAndGetDigits(min,max,maxAtt int,tO uint32,term,af,iaf,vN string) (string,error) { cArgs:=m.Called(min,max,maxAtt,tO,term,af,iaf,vN); return cArgs.String(0),cArgs.Error(1) }

// --- MockCallServicerForESL ---
type MockCallServicerForESL struct{ mock.Mock }
// Call methods
func (m *MockCallServicerForESL) UpdateCallStatus(ctx domain.MinimalCallContext,s string,hC string) error { return m.Called(ctx,s,hC).Error(0) }
// Recording methods
func (m *MockCallServicerForESL) CreateRecording(ctx domain.MinimalCallContext,cS *string,rS,fP string,dur uint32,fS string,sB int64) error { return m.Called(ctx,cS,rS,fP,dur,fS,sB).Error(0) }
// Conference methods
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
// SMS methods
func (m *MockCallServicerForESL) SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error) { args := m.Called(ctx, accountSid, to, from, body, msgSID, actionURL, actionMethod); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.SMSMessage), args.Error(1) }
func (m *MockCallServicerForESL) GetSMSBySID(ctx context.Context, sid string) (*domain.SMSMessage, error) { args := m.Called(ctx, sid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.SMSMessage), args.Error(1) }
func (m *MockCallServicerForESL) UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error { return m.Called(ctx, agbaraSid, gatewaySid, status, errorCode, errorMessage, eventTime).Error(0) }
func (m *MockCallServicerForESL) RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error) { args := m.Called(ctx, accountSid, to, from, body, inboundGatewayMsgSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.SMSMessage), args.Error(1) }
// Application methods
func (m *MockCallServicerForESL) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error) { args := m.Called(ctx, did); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Application), args.Error(1) }


// --- setupTestMocks ---
func setupTestMocks(t *testing.T) (*logrus.Entry, *MockMinimalCallContext, *MockEslConnectionExecutor, *MockCallServicerForESL) {
	testLogger := logrus.NewEntry(logrus.New()); testLogger.Logger.SetOutput(io.Discard)
	mockCtx := new(MockMinimalCallContext)
	mockEsl := new(MockEslConnectionExecutor)
	mockCallSvc := new(MockCallServicerForESL)

	mockCtx.On("Log").Return(testLogger)
	mockCtx.On("GetUuid").Return("test-uuid-123")
	mockCtx.On("GetFreeswitchUUID").Return("test-fs-uuid-123")
	mockCtx.On("GetAccountSid").Return("test-account-sid")

	dummyElementsChan := make(chan []domain.CallControlElement); close(dummyElementsChan)
	mockCtx.On("GetNextElementsChannel").Maybe().Return((<-chan []domain.CallControlElement)(dummyElementsChan))
	dummyHangupChan := make(chan struct{});
	mockCtx.On("HangupChan").Maybe().Return((<-chan struct{})(dummyHangupChan))

	return testLogger, mockCtx, mockEsl, mockCallSvc
}

// --- Verb Tests ---
func TestSayElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	say := domain.SayElement{Text: "Hello", Loop: 1, Engine: "flite", Voice: "slt"}
	expectedSpeakArg := "flite|slt|Hello"
	mockEsl.On("Execute", "speak", expectedSpeakArg).Return("OK", nil).Once()
	result := say.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestPlayElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	play := domain.PlayElement{URL: "file://sound.wav", Loop: 1}
	mockEsl.On("Execute", "playback", "file://sound.wav").Return("OK", nil).Once()
	result := play.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}
func TestHangupElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	hangup := domain.HangupElement{Reason: "USER_BUSY"}
	mockEsl.On("Hangup", "USER_BUSY").Return("OK", nil).Once()
	result := hangup.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionHangup, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}
func TestPauseElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	pause := domain.PauseElement{Length: 5}
	expectedSleepArg := fmt.Sprintf("%d", 5*1000)
	mockEsl.On("Execute", "sleep", expectedSleepArg).Return("OK", nil).Once()
	result := pause.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}
func TestRedirectElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	redirect := domain.RedirectElement{URL: "http://example.com/next", Method: "POST"}
	result := redirect.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionRedirect, result.Action); assert.NoError(t, result.Err)
	assert.Equal(t, "http://example.com/next", result.RedirectURL)
	assert.Equal(t, "POST", result.RedirectMethod)
}

// --- GatherElement Execute Tests ---
func TestGatherElement_Execute_Success_NoActionURL_NoInput(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{TimeoutSeconds: 5, FinishOnKey: "#"}
	mockEsl.On("PlayAndGetDigits", 0, 0, 0, uint32(5000), "#", "", "", "gathered_digits").Return("", nil).Once()
	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.Equal(t, "", result.Digits); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestGatherElement_Execute_Success_WithActionURL_WithDigits(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{ActionURL: "/gather_handler", Method: "POST", NumDigits: 4}
	mockEsl.On("PlayAndGetDigits", 0, 4, 0, uint32(0), "", "", "", "gathered_digits").Return("1234", nil).Once()
	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionRedirect, result.Action); assert.Equal(t, "1234", result.Digits)
	assert.Equal(t, "/gather_handler", result.RedirectURL); assert.Equal(t, "POST", result.RedirectMethod)
	assert.NoError(t, result.Err); mockEsl.AssertExpectations(t)
}
func TestGatherElement_Execute_PlayAndGetDigitsError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{}
	expectedError := errors.New("pgd failed")
	mockEsl.On("PlayAndGetDigits", 0,0,0,uint32(0),"","","","gathered_digits").Return("", expectedError).Once()
	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.Error(t, result.Err); assert.Contains(t, result.Err.Error(), "PlayAndGetDigits failed")
	mockEsl.AssertExpectations(t)
}
func TestGatherElement_Execute_NestedPlayError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{Play: &domain.PlayElement{URL: "prompt.wav"}}
	expectedError := errors.New("nested playback failed")
	mockEsl.On("Execute", "playback", "prompt.wav").Return("", expectedError).Once()
	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.Error(t, result.Err); assert.Contains(t, result.Err.Error(), "Nested Play failed")
	mockEsl.AssertExpectations(t)
}

// --- RecordElement Execute Tests ---
func TestRecordElement_Execute_Success(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{ActionURL: "/rec_notify", MaxLengthSeconds: 60, FileFormat: "mp3"}
	var capturedFilePath string
	mockEsl.On("RecordSession", mock.MatchedBy(func(fp string) bool { capturedFilePath=fp; return true }), uint32(60), uint(0), uint(0)).Return("OK", nil).Once()
	mockCtx.On("SetPendingRecording", mock.MatchedBy(func(info callcontrol.PendingRecordInfo) bool {
		return info.ExpectedFilePath == capturedFilePath && info.OriginalElement.ActionURL == "/rec_notify"
	})).Return().Once()
	result := record.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestRecordElement_Execute_PlayBeep(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{PlayBeep: true}
	mockEsl.On("Execute", "playback", "tone_stream://%(1000,0,640)").Return("OK", nil).Once()
	mockEsl.On("RecordSession", mock.Anything, uint32(0), uint(0), uint(0)).Return("OK", nil).Once()
	mockCtx.On("SetPendingRecording", mock.Anything).Return().Once()
	result := record.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestRecordElement_Execute_RecordSessionError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{}
	expectedError := errors.New("record failed")
	mockEsl.On("RecordSession", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", expectedError).Once()
	result := record.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.Error(t, result.Err); assert.Contains(t, result.Err.Error(), "failed to start recording")
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}

// --- DialElement Execute Tests (Focus on Nested Targets) ---
func TestDialElement_Execute_WithNumberTarget(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	numberElem := &domain.NumberElement{PhoneNumber: "5551234567", SendDigits: "ww123"}
	dialElem := domain.DialElement{Number: numberElem}
	mockCtx.On("GetVariable", "caller_id_number").Return("").Once()
	expectedVars := map[string]string{ "originate_timeout":"60", "agbara_parent_call_sid":"test-uuid-123", "agbara_dial_action_url":"", "agbara_dial_action_method":"POST", "agbara_dial_hangup_on_star":"false", "agbara_dial_send_digits_on_answer": "ww123"}
	mockEsl.On("Originate", "sofia/gateway/default/5551234567", expectedVars).Return("bleg-uuid", nil).Once()
	mockCtx.On("AddPendingDial", "bleg-uuid", mock.AnythingOfType("callcontrol.PendingDialInfo")).Return().Once()
	result := dialElem.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestDialElement_Execute_WithNestedConferenceTarget(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElem := &domain.NestedConferenceElement{RoomName: "dialToRoom", Muted: true}
	dialElem := domain.DialElement{NestedConference: confElem}
	mockCtx.On("GetVariable", "caller_id_number").Return("").Once()
	expectedVars := map[string]string{ "originate_timeout":"60", "agbara_parent_call_sid":"test-uuid-123", "agbara_dial_action_url":"", "agbara_dial_action_method":"POST", "agbara_dial_hangup_on_star":"false"}
	mockEsl.On("Originate", "conference:dialToRoom@default+mute", expectedVars).Return("bleg-uuid", nil).Once()
	mockCtx.On("AddPendingDial", "bleg-uuid", mock.AnythingOfType("callcontrol.PendingDialInfo")).Return().Once()
	result := dialElem.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestDialElement_Execute_WithSipTarget(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	sipElem := &domain.SipElement{URI: "sip:user@domain.com"}
	dialElem := domain.DialElement{Sip: sipElem}
	mockCtx.On("GetVariable", "caller_id_number").Return("").Once()
	expectedVars := map[string]string{ "originate_timeout":"60", "agbara_parent_call_sid":"test-uuid-123", "agbara_dial_action_url":"", "agbara_dial_action_method":"POST", "agbara_dial_hangup_on_star":"false"}
	mockEsl.On("Originate", "sip:user@domain.com", expectedVars).Return("bleg-uuid", nil).Once()
	mockCtx.On("AddPendingDial", "bleg-uuid", mock.AnythingOfType("callcontrol.PendingDialInfo")).Return().Once()
	result := dialElem.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t); mockCtx.AssertExpectations(t)
}
func TestDialElement_Execute_EmptyNestedTargetError(t *testing.T) {
	_, mockCtx, _, _ := setupTestMocks(t)
	dialNumEmpty := domain.DialElement{Number: &domain.NumberElement{PhoneNumber: " "}}
	resultNum := dialNumEmpty.Execute(mockCtx, nil, nil)
	assert.Equal(t, domain.ActionError, resultNum.Action); assert.EqualError(t, resultNum.Err, "Dial <Number> is empty")
	mockCtx.AssertExpectations(t)
}

// --- ConferenceElement Execute Tests ---
func TestConferenceElement_Execute_Success(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: "testRoom123", Muted: true, CallbackURL: "/conf_events"}
	mockConf := &domain.Conference{SID: "CFmockSID", FriendlyName: "testRoom123", AccountSID: "ACtestacc"}
	mockParticipant := &domain.ConferenceParticipant{SID: "CPmockPartSID"}
	mockCtx.On("GetAccountSid").Return("ACtestacc").Times(2); mockCtx.On("GetUuid").Return("CAtestcall").Times(2)
	mockCallSvc.On("GetOrCreateConference", mock.Anything, "ACtestacc", "testRoom123").Return(mockConf, nil).Once()
	mockCallSvc.On("AddParticipant", mock.Anything, "CFmockSID", "CAtestcall", mock.AnythingOfType("string"), "ACtestacc", true, false).Return(mockParticipant, nil).Once()
	mockCtx.On("SetCurrentConferenceParticipantSID", "CPmockPartSID").Return().Once()
	mockCtx.On("EnterConference", "CFmockSID", "testRoom123", "/conf_events", "POST").Return().Once()
	expectedConfDialString := "testRoom123@default+mute"
	mockEsl.On("Execute", "conference", expectedConfDialString).Return("OK", nil).Once()
	mockCtx.On("LeaveConference").Return().Once()
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockCtx.AssertExpectations(t); mockEsl.AssertExpectations(t); mockCallSvc.AssertExpectations(t)
}
func TestConferenceElement_Execute_EmptyRoomName(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: " "}
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.EqualError(t, result.Err, "Conference RoomName is empty")
	mockCtx.AssertExpectations(t)
}
func TestConferenceElement_Execute_GetOrCreateDBError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: "testRoomDbError"}
	dbError := errors.New("db error get_or_create")
	mockCtx.On("GetAccountSid").Return("ACtestdb").Once()
	mockCallSvc.On("GetOrCreateConference", mock.Anything, "ACtestdb", "testRoomDbError").Return(nil, dbError).Once()
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.Error(t, result.Err); assert.True(t, errors.Is(result.Err, dbError) || strings.Contains(result.Err.Error(), "DB GetOrCreateConference failed"))
	mockCtx.AssertExpectations(t); mockCallSvc.AssertExpectations(t)
}
func TestConferenceElement_Execute_AddParticipantDBError_Fatal(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: "testRoomAddPartError"}
	mockConf := &domain.Conference{SID: "CFmockSID2", FriendlyName: "testRoomAddPartError"}
	dbError := errors.New("db error add_participant")
	mockCtx.On("GetAccountSid").Return("ACtestadd").Times(2); mockCtx.On("GetUuid").Return("CAtestcall2").Times(1)
	mockCallSvc.On("GetOrCreateConference", mock.Anything, "ACtestadd", "testRoomAddPartError").Return(mockConf, nil).Once()
	mockCallSvc.On("AddParticipant", mock.Anything, mockConf.SID, "CAtestcall2", mock.AnythingOfType("string"), "ACtestadd", false, false).Return(nil, dbError).Once()
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action); assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "Failed to add participant")
	mockEsl.AssertNotCalled(t, "Execute", "conference", mock.Anything)
	mockCtx.AssertExpectations(t); mockCallSvc.AssertExpectations(t)
}
func TestConferenceElement_Execute_AddParticipantConflictAllowed(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: "testRoomConflict"}
	mockConf := &domain.Conference{SID: "CFmockSID3", FriendlyName: "testRoomConflict"}
	mockCtx.On("GetAccountSid").Return("ACtestconflict").Times(2); mockCtx.On("GetUuid").Return("CAtestcall3").Times(2)
	mockCallSvc.On("GetOrCreateConference", mock.Anything, "ACtestconflict", "testRoomConflict").Return(mockConf, nil).Once()
	mockCallSvc.On("AddParticipant", mock.Anything, mockConf.SID, "CAtestcall3", mock.AnythingOfType("string"), "ACtestconflict", false, false).Return(nil, domain.ErrConflict).Once()
	mockCtx.On("EnterConference", mockConf.SID, mockConf.FriendlyName, "", "POST").Return().Once()
	mockEsl.On("Execute", "conference", "testRoomConflict@default").Return("OK", nil).Once()
	mockCtx.On("LeaveConference").Return().Once()
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action); assert.NoError(t, result.Err)
	mockCtx.AssertExpectations(t); mockCallSvc.AssertExpectations(t); mockEsl.AssertExpectations(t)
}
func TestConferenceElement_Execute_EslError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	confElement := domain.ConferenceElement{RoomName: "testRoomESLError"}
	mockConf := &domain.Conference{SID: "CFmockSID4", FriendlyName: "testRoomESLError"}
	mockParticipant := &domain.ConferenceParticipant{SID: "CPmockPartSID4"}
	eslError := errors.New("esl conference failed")
	mockCtx.On("GetAccountSid").Return("ACtestesl").Times(2); mockCtx.On("GetUuid").Return("CAtestcall4").Times(2)
	mockCallSvc.On("GetOrCreateConference", mock.Anything, "ACtestesl", "testRoomESLError").Return(mockConf, nil).Once()
	mockCallSvc.On("AddParticipant", mock.Anything, mockConf.SID, "CAtestcall4", mock.AnythingOfType("string"), "ACtestesl", false, false).Return(mockParticipant, nil).Once()
	mockCtx.On("SetCurrentConferenceParticipantSID", "CPmockPartSID4").Return().Once()
	mockCtx.On("EnterConference", mockConf.SID, mockConf.FriendlyName, "", "POST").Return().Once()
	mockEsl.On("Execute", "conference", "testRoomESLError@default").Return("", eslError).Once()
	mockCtx.On("LeaveConference").Return().Once()
	result := confElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Contains(t, []domain.CallControlAction{domain.ActionError, domain.ActionHangup}, result.Action)
	assert.Error(t, result.Err); assert.True(t, errors.Is(result.Err, eslError) || strings.Contains(result.Err.Error(), eslError.Error()))
	mockCtx.AssertExpectations(t); mockCallSvc.AssertExpectations(t); mockEsl.AssertExpectations(t)
}

// --- SmsElement Execute Tests ---
func TestSmsElement_Execute_Success(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: "to_num", From: "from_num", Body: "Hello SMS"}

	mockCtx.On("GetAccountSid").Return("AC123").Once()

	mockCallSvc.On("SendSMS",
		mock.Anything, // context.Context
		"AC123",
		"to_num",
		"from_num",
		"Hello SMS",
		mock.MatchedBy(func(sid string) bool { return strings.HasPrefix(sid, "SM") }),
		"",
		"POST",
	).Return(&domain.SMSMessage{SID: "SMgeneratedsid"}, nil).Once()

	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockCallSvc.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestSmsElement_Execute_ServiceError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: "to_num", From: "from_num", Body: "Error SMS"}
	serviceErr := errors.New("sms service failure")

	mockCtx.On("GetAccountSid").Return("AC123").Once()
	mockCallSvc.On("SendSMS", mock.Anything, "AC123", "to_num", "from_num", "Error SMS", mock.AnythingOfType("string"), "", "POST").Return(nil, serviceErr).Once()

	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	assert.True(t, errors.Is(result.Err, serviceErr) || strings.Contains(result.Err.Error(), serviceErr.Error()))
	mockCallSvc.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestSmsElement_Execute_Validation_EmptyTo(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: " ", From: "from_num", Body: "Test Body"}
	mockCtx.On("GetAccountSid").Maybe().Return("AC123") // May not be called due to early exit
	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.EqualError(t, result.Err, "SmsElement: 'to', 'from', and message body cannot be empty.")
	mockCallSvc.AssertNotCalled(t, "SendSMS", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSmsElement_Execute_Validation_EmptyFrom(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: "to_num", From: " ", Body: "Test Body"}
	mockCtx.On("GetAccountSid").Maybe().Return("AC123")
	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.EqualError(t, result.Err, "SmsElement: 'to', 'from', and message body cannot be empty.")
}

func TestSmsElement_Execute_Validation_EmptyBody(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: "to_num", From: "from_num", Body: "   "}
	mockCtx.On("GetAccountSid").Maybe().Return("AC123")
	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.EqualError(t, result.Err, "SmsElement: 'to', 'from', and message body cannot be empty.")
}

func TestSmsElement_Execute_Validation_NoAccountSid(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	smsElement := domain.SmsElement{To: "to_num", From: "from_num", Body: "Valid body"}
	mockCtx.On("GetAccountSid").Return("").Once() // Empty Account SID
	result := smsElement.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.EqualError(t, result.Err, "SmsElement: AccountSID is missing from context.")
	mockCallSvc.AssertNotCalled(t, "SendSMS", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}


// Ensure all mocks are asserted
func TestMain(m *testing.M) {
	m.Run()
}

[end of agbaravoip_golang/internal/domain/call_control_test.go]
