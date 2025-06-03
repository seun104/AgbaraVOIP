package esl_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest" // For mocking http client if XMLProcessor is tested more deeply here
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"


	"github.com/fiorix/go-eventsocket/eventsocket" // For creating dummy events
	"github.com/google/uuid"                      // For SIDs
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/esl" // To access FSOutboundServer and its methods (if made public/testable)
)

// --- Mocks ---
type MockMinimalCallContext struct {
	mock.Mock
	// Store actual pending info to test clearing/getting
	pendingRec        *callcontrol.PendingRecordInfo
	pendingDials      map[string]*callcontrol.PendingDialInfo
	pendingDialsMutex sync.RWMutex
	nextElementsChan  chan []domain.CallControlElement

	// Fields to return for getter methods
	ReturnFsUUID        string
	ReturnAgbaraCallSID string
	ReturnAccountSID    string
	ReturnAppSID        string
	ReturnAnswerURL     string
	TestLogger          *logrus.Entry
	IsHangup            bool // To control IsHangupInitiated
}

func NewMockMinimalCallContext(t *testing.T, fsUUID, agbaraCallSID, accountSID string) *MockMinimalCallContext {
	testLogger := logrus.New()
	testLogger.SetOutput(io.Discard)
	entry := logrus.NewEntry(testLogger)

	mcc := &MockMinimalCallContext{
		pendingDials:     make(map[string]*callcontrol.PendingDialInfo),
		nextElementsChan: make(chan []domain.CallControlElement, 5), // Buffered for tests
		ReturnFsUUID:     fsUUID,
		ReturnAgbaraCallSID: agbaraCallSID,
		ReturnAccountSID: accountSID,
		TestLogger:       entry,
	}
	// Default behaviors
	mcc.On("Log").Maybe().Return(entry)
	mcc.On("GetUuid").Maybe().Return(agbaraCallSID)
	mcc.On("GetFreeswitchUUID").Maybe().Return(fsUUID)
	mcc.On("GetAccountSid").Maybe().Return(accountSID)
	mcc.On("GetApplicationSid").Maybe().Return("") // Default
	mcc.On("GetAnswerURL").Maybe().Return("")    // Default
	mcc.On("IsHangupInitiated").Maybe().Return(mcc.IsHangup)
	mcc.On("SetHangupInitiated").Maybe().Run(func(args mock.Arguments) { mcc.IsHangup = true })
	mcc.On("GetNextElementsChannel").Maybe().Return((<-chan []domain.CallControlElement)(mcc.nextElementsChan))

	return mcc
}

func (m *MockMinimalCallContext) Log() *logrus.Entry { m.Called(); return m.TestLogger }
func (m *MockMinimalCallContext) GetUuid() string { args := m.Called(); return args.String(0) }
func (m *MockMinimalCallContext) GetFreeswitchUUID() string { args := m.Called(); return args.String(0) }
func (m *MockMinimalCallContext) GetAccountSid() string { args := m.Called(); return args.String(0) }
func (m *MockMinimalCallContext) GetApplicationSid() string { args := m.Called(); return args.String(0) }
func (m *MockMinimalCallContext) GetAnswerURL() string { args := m.Called(); return args.String(0) }
func (m *MockMinimalCallContext) GetVariable(v string) string { args := m.Called(v); return args.String(0) }
func (m *MockMinimalCallContext) IsHangupInitiated() bool { args := m.Called(); return args.Bool(0) }
func (m *MockMinimalCallContext) SetHangupInitiated() { m.Called() }

func (m *MockMinimalCallContext) SetPendingRecording(info interface{}) {
	m.Called(info)
	if pRecInfo, ok := info.(callcontrol.PendingRecordInfo); ok {
		m.pendingRec = &pRecInfo
	} else if pRecInfoPtr, ok := info.(*callcontrol.PendingRecordInfo); ok {
		m.pendingRec = pRecInfoPtr
	} else if info == nil {
        m.pendingRec = nil
    }
}
func (m *MockMinimalCallContext) GetPendingRecording() (interface{}, bool) {
	m.Called()
	if m.pendingRec == nil { return nil, false }
	return m.pendingRec, true
}
func (m *MockMinimalCallContext) ClearPendingRecording() { m.Called(); m.pendingRec = nil }

func (m *MockMinimalCallContext) AddPendingDial(childChannelUUID string, info interface{}) {
	m.Called(childChannelUUID, info)
	m.pendingDialsMutex.Lock()
	defer m.pendingDialsMutex.Unlock()
	if pDialInfo, ok := info.(callcontrol.PendingDialInfo); ok {
		m.pendingDials[childChannelUUID] = &pDialInfo
	} else if pDialInfoPtr, ok := info.(*callcontrol.PendingDialInfo); ok {
		m.pendingDials[childChannelUUID] = pDialInfoPtr
	}
}
func (m *MockMinimalCallContext) GetPendingDial(childChannelUUID string) (interface{}, bool) {
	m.Called(childChannelUUID)
	m.pendingDialsMutex.RLock()
	defer m.pendingDialsMutex.RUnlock()
	info, exists := m.pendingDials[childChannelUUID]
	return info, exists
}
func (m *MockMinimalCallContext) RemovePendingDial(childChannelUUID string) {
	m.Called(childChannelUUID)
	m.pendingDialsMutex.Lock()
	defer m.pendingDialsMutex.Unlock()
	delete(m.pendingDials, childChannelUUID)
}
func (m *MockMinimalCallContext) GetAllPendingDials() map[string]interface{} {
	m.Called()
	m.pendingDialsMutex.RLock()
	defer m.pendingDialsMutex.RUnlock()
	res := make(map[string]interface{})
	for k, v := range m.pendingDials { res[k] = v }
	return res
}
func (m *MockMinimalCallContext) SendNextElements(elements []domain.CallControlElement) error {
	args := m.Called(elements)
	if args.Error(0) == nil {
		select {
		case m.nextElementsChan <- elements:
			m.TestLogger.Debugf("MockMinimalCallContext: Sent %d elements to nextElementsChan", len(elements))
		case <-time.After(50 * time.Millisecond): // Short timeout for test
			m.TestLogger.Error("MockMinimalCallContext: SendNextElements timeout, channel full or no listener.")
			return errors.New("mock SendNextElements: channel full or no listener")
		}
	}
	return args.Error(0)
}
func (m *MockMinimalCallContext) GetNextElementsChannel() <-chan []domain.CallControlElement {
	m.Called()
	return m.nextElementsChan
}
func (m *MockMinimalCallContext) GetCallSID() string { // This is AgbaraCallSID
    return m.ReturnAgbaraCallSID
}
// Add this to satisfy the interface if it's there, otherwise CallContext needs it.
// For tests, we might not need it if we don't test hangup loop logic deeply.
func (m *MockMinimalCallContext) HangupChan() <-chan struct{} {
	args := m.Called()
	if args.Get(0) == nil {
		dummyCh := make(chan struct{})
		// close(dummyCh) // Closing immediately might not be what tests expect
		return dummyCh
	}
	return args.Get(0).(<-chan struct{})
}


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


type MockCallServicerForESL struct{ mock.Mock }
func (m *MockCallServicerForESL) UpdateCallStatus(ctx domain.MinimalCallContext,s string,hC string) error { return m.Called(ctx,s,hC).Error(0) }
func (m *MockCallServicerForESL) UpdateCallRecording(ctx domain.MinimalCallContext,rP string,dS int,f string) error { return m.Called(ctx,rP,dS,f).Error(0) }
func (m *MockCallServicerForESL) CreateRecording(ctx domain.MinimalCallContext,cS *string,rS,fP string,dur uint32,fS string,sB int64) error { return m.Called(ctx,cS,rS,fP,dur,fS,sB).Error(0) }

type MockXMLProcessor struct{ mock.Mock }
func (m *MockXMLProcessor) FetchAndParseXML(ctx context.Context,callCtx domain.MinimalCallContext,urlStr,method string,params url.Values) ([]domain.CallControlElement,error) {
	args := m.Called(ctx,callCtx,urlStr,method,params)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]domain.CallControlElement), args.Error(1)
}

// TestHarness to hold mocks and server instance
type TestHarness struct {
	T                *testing.T
	Cfg              config.Config
	Logger           *logrus.Logger
	LogEntry         *logrus.Entry
	MockCallSvc      *MockCallServicerForESL
	MockXmlProcessor *MockXMLProcessor
	MockEslExecutor  *MockEslConnectionExecutor
	Server           *esl.FSOutboundServer // The actual server instance
}

func setupIntegrationTest(t *testing.T) *TestHarness {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	h := &TestHarness{
		T:                t,
		Cfg:              config.Config{Freeswitch: config.FreeswitchConfig{FSOutboundListenAddress: "127.0.0.1:0"}}, // Use :0 for dynamic port
		Logger:           logger,
		LogEntry:         logEntry,
		MockCallSvc:      new(MockCallServicerForESL),
		MockXmlProcessor: new(MockXMLProcessor),
		MockEslExecutor:  new(MockEslConnectionExecutor),
	}
	// We need a way to inject the mocked EslConnectionExecutor into the handleOutboundConnection scope.
	// The actual FSOutboundServer creates NewESLConnectionAdapter(rawEslConn).
	// For these tests, we might need to test handleEslEvents more directly,
	// or modify FSOutboundServer to allow injection of EslConnectionExecutor for testing.

	// For now, FSOutboundServer is not started, and handleEslEvents is tested by direct call if possible or conceptually.
	// server, err := esl.NewFSOutboundServer(h.Cfg, h.Logger, h.MockCallSvc, h.MockXmlProcessor)
	// assert.NoError(t, err)
	// h.Server = server
	// t.Cleanup(func() { h.Server.Shutdown() })
	return h
}


func TestHandleEslEvents_RECORD_STOP_NoActionURL(t *testing.T) {
	h := setupIntegrationTest(t)

	fsUUID := "fs-uuid-record-test"
	agbaraSID := "CA-record-test"
	accSID := "AC-record-test"
	mockCallCtx := NewMockMinimalCallContext(t, fsUUID, agbaraSID, accSID)

	recordElem := &domain.RecordElement{ActionURL: "", FileFormat: "wav"}
	pendingRec := callcontrol.PendingRecordInfo{
		OriginalElement:  recordElem,
		ExpectedFilePath: "/tmp/test_rec.wav",
	}
	mockCallCtx.SetPendingRecording(pendingRec) // Manually set for test

	// Setup CallContext mock calls for this specific test
	mockCallCtx.On("GetPendingRecording").Return(&pendingRec, true).Once()
	mockCallCtx.On("ClearPendingRecording").Return().Once()
	mockCallCtx.On("GetCallSID").Return(agbaraSID).Once() // For CreateRecording
	mockCallCtx.On("GetAccountSid").Return(accSID).Once() // For CreateRecording
	mockCallCtx.On("GetUuid").Return(agbaraSID).Once() // For logging or ActionURL params (not used here)


	// Setup mock for CallService.CreateRecording
	h.MockCallSvc.On("CreateRecording", mockCallCtx, &agbaraSID, mock.AnythingOfType("string"), "/tmp/test_rec.wav", uint32(10), "wav", int64(0)).Return(nil).Once()

	eventData := make(map[string]string)
	eventData["Event-Name"] = "RECORD_STOP"
	eventData["variable_record_file_path"] = "/tmp/test_rec.wav"
	eventData["variable_record_seconds"] = "10"
	event := eventsocket.Event{Header: eventData}

	// This test requires `handleEslEvents` to be callable/testable.
	// We are calling a conceptual method on a test helper or refactored esl.FSOutboundServer.
	// For now, this test is more of a specification of how it *should* behave.
	// If FSOutboundServer had a method `ProcessEventForTest(callCtx, event, eslExecutor)`

	// Create a minimal FSOutboundServer instance for the method receiver
	// This is still problematic as NewFSOutboundServer starts listener.
	// We need a way to test handleEslEvents without starting the server.
	// One way: make handleEslEvents a public function (e.g. esl.HandleEvent(...))
	// Another way: use a constructor that doesn't start the listener.

	// Let's assume we can get an instance for testing or `handleEslEvents` is made public.
	// For this example, we'll skip the actual execution and focus on setup.
	t.Skip("Skipping handleEslEvents test: requires FSOutboundServer method HandleSpecificEvent(ctx, event, executor) or complex ESL event simulation for full loop.")

	// Conceptual assertions if the event processing logic was directly invoked:
	// serverInstance.HandleSpecificEvent(mockCallCtx, &event, h.MockEslExecutor)
	// h.MockCallSvc.AssertExpectations(t)
	// mockCallCtx.AssertExpectations(t)
}

// TODO: TestHandleEslEvents_RECORD_STOP_WithActionURL_Success
// TODO: TestHandleEslEvents_CHANNEL_ANSWER_Bleg_WithActionURL
// TODO: TestHandleEslEvents_CHANNEL_HANGUP_Bleg_WithActionURL
// TODO: TestHandleEslEvents_DTMF_FinishOnKey_Recording

func TestMain(m *testing.M) {
	m.Run()
}
