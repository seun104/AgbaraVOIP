package domain_test

import (
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/agbaravoip_golang/internal/domain"
)

// --- Logger Mock (if domain.Logger interface is used; otherwise direct logrus.Entry) ---
// For now, assuming MinimalCallContext.Log() returns *logrus.Entry directly as per call_control_interfaces.go

// --- MockMinimalCallContext ---
type MockMinimalCallContext struct {
	mock.Mock
}

func (m *MockMinimalCallContext) Log() *logrus.Entry {
	args := m.Called()
	if args.Get(0) == nil {
		// To prevent panic if Log isn't set up in a test but is called by underlying code (e.g. error path)
		// Return a default discard logger.
		entry := logrus.NewEntry(logrus.New())
		entry.Logger.SetOutput(io.Discard)
		return entry
	}
	return args.Get(0).(*logrus.Entry)
}

func (m *MockMinimalCallContext) GetUuid() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMinimalCallContext) GetAccountSid() string { // Added as per typical context needs
	args := m.Called()
	return args.String(0)
}

func (m *MockMinimalCallContext) GetApplicationSid() string { // Added as per typical context needs
	args := m.Called()
	return args.String(0)
}

func (m *MockMinimalCallContext) GetAnswerURL() string { // Added as per typical context needs
	args := m.Called()
	return args.String(0)
}
func (m *MockMinimalCallContext) GetVariable(varName string) string {
	args := m.Called(varName)
	return args.String(0)
}

func (m *MockMinimalCallContext) IsHangupInitiated() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockMinimalCallContext) SetHangupInitiated() {
	m.Called()
}

// --- MockEslConnectionExecutor ---
type MockEslConnectionExecutor struct {
	mock.Mock
}

func (m *MockEslConnectionExecutor) Execute(command string, cmdArgs ...string) (string, error) {
	callArgs := make([]interface{}, len(cmdArgs)+1)
	callArgs[0] = command
	for i, arg := range cmdArgs {
		callArgs[i+1] = arg
	}
	args := m.Called(callArgs...)
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) ExecuteSofia(command string, cmdArgs ...string) (string, error) {
	callArgs := make([]interface{}, len(cmdArgs)+1)
	callArgs[0] = command
	for i, arg := range cmdArgs {
		callArgs[i+1] = arg
	}
	args := m.Called(callArgs...)
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) SendMsg(msg map[string]string) (string, error) {
	args := m.Called(msg)
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) GetVar(varName string) (string, error) {
	args := m.Called(varName)
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) Answer() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) Hangup(reason string) (string, error) {
	args := m.Called(reason)
	return args.String(0), args.Error(1)
}

// Stubs for methods mentioned in prompt but not yet on EslConnectionExecutor interface
func (m *MockEslConnectionExecutor) PlayAndGetDigits(minDigits, maxDigits int, maxAttempts int, timeout uint32, terminators string, audioFile string, invalidAudioFile string, varName string) (string, error) {
	args := m.Called(minDigits, maxDigits, maxAttempts, timeout, terminators, audioFile, invalidAudioFile, varName)
	return args.String(0), args.Error(1)
}

func (m *MockEslConnectionExecutor) RecordSession(filePath string, maxDurationSec uint32, silenceThreshold uint, silenceHits uint) (string, error) {
	args := m.Called(filePath, maxDurationSec, silenceThreshold, silenceHits)
	return args.String(0), args.Error(1)
}

// --- MockCallServicerForESL ---
type MockCallServicerForESL struct {
	mock.Mock
}

func (m *MockCallServicerForESL) UpdateCallStatus(ctx domain.MinimalCallContext, status string, hangupCause string) error {
	args := m.Called(ctx, status, hangupCause)
	return args.Error(0)
}

func (m *MockCallServicerForESL) UpdateCallRecording(ctx domain.MinimalCallContext, recordingPath string, durationSec int, format string) error {
	args := m.Called(ctx, recordingPath, durationSec, format)
	return args.Error(0)
}

// --- setupTestMocks ---
func setupTestMocks(t *testing.T) (*logrus.Entry, *MockMinimalCallContext, *MockEslConnectionExecutor, *MockCallServicerForESL) {
	testLogger := logrus.NewEntry(logrus.New())
	testLogger.Logger.SetOutput(io.Discard)

	mockCtx := new(MockMinimalCallContext)
	mockEsl := new(MockEslConnectionExecutor)
	mockCallSvc := new(MockCallServicerForESL)

	// Default mock expectations
	mockCtx.On("Log").Return(testLogger)
	mockCtx.On("GetUuid").Return("test-uuid-123") // Default UUID
	// mockCtx.On("ESLConnection").Return(mockEsl) // Not part of MinimalCallContext interface

	return testLogger, mockCtx, mockEsl, mockCallSvc
}

// --- Verb Tests ---

func TestSayElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	say := domain.SayElement{Text: "Hello", Loop: 1, Engine: "flite", Voice: "slt"}

	// Expected argument for Execute("speak", arg)
	expectedSpeakArg := "flite|slt|Hello"
	mockEsl.On("Execute", "speak", expectedSpeakArg).Return("OK", nil).Once()

	result := say.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestSayElement_Execute_Loop(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	say := domain.SayElement{Text: "Loop me", Loop: 3} // Engine and Voice will use defaults

	expectedSpeakArg := "flite|slt|Loop me" // Default engine/voice
	mockEsl.On("Execute", "speak", expectedSpeakArg).Return("OK", nil).Times(3)

	result := say.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestSayElement_Execute_Error(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	say := domain.SayElement{Text: "Error test", Loop: 1}

	expectedSpeakArg := "flite|slt|Error test"
	mockEsl.On("Execute", "speak", expectedSpeakArg).Return("", errors.New("speak failed")).Once()

	result := say.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestPlayElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	play := domain.PlayElement{URL: "file://sound.wav", Loop: 1}

	mockEsl.On("Execute", "playback", "file://sound.wav").Return("OK", nil).Once()
	result := play.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestPlayElement_Execute_EmptyURL(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	play := domain.PlayElement{URL: "  ", Loop: 1} // Empty URL

	// No ESL call expected
	result := play.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action) // Should continue, not error out for empty URL in Play
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t) // Ensure no methods on mockEsl were called
}

func TestPlayElement_Execute_Error(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	play := domain.PlayElement{URL: "file://error.wav", Loop: 1}

	mockEsl.On("Execute", "playback", "file://error.wav").Return("", errors.New("playback failed")).Once()
	result := play.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestHangupElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	hangup := domain.HangupElement{Reason: "USER_BUSY"}

	mockEsl.On("Hangup", "USER_BUSY").Return("OK", nil).Once()
	result := hangup.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionHangup, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestHangupElement_Execute_Scheduled(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	scheduleTime := 60
	hangup := domain.HangupElement{Reason: "NORMAL_CLEARING", Schedule: &scheduleTime}

	mockEsl.On("Execute", "sched_hangup", "+60", "NORMAL_CLEARING").Return("OK", nil).Once()
	result := hangup.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionHangup, result.Action) // Still ActionHangup, but ESL handles the delay
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestHangupElement_Execute_Error(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	hangup := domain.HangupElement{Reason: "ERROR_CONDITION"}

	mockEsl.On("Hangup", "ERROR_CONDITION").Return("", errors.New("hangup failed")).Once()
	result := hangup.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionHangup, result.Action) // Action is still Hangup
	assert.Error(t, result.Err)                        // But error is reported
	mockEsl.AssertExpectations(t)
}


func TestPauseElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	pause := domain.PauseElement{Length: 5}
	expectedSleepArg := fmt.Sprintf("%d", 5*1000)

	mockEsl.On("Execute", "sleep", expectedSleepArg).Return("OK", nil).Once()
	result := pause.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestPauseElement_Execute_ZeroLength(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	pause := domain.PauseElement{Length: 0}

	// No ESL call expected
	result := pause.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t) // Verify no calls to ESL
}

func TestPauseElement_Execute_NegativeLength(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	pause := domain.PauseElement{Length: -5}

	// No ESL call expected
	result := pause.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t) // Verify no calls to ESL
}


func TestPauseElement_Execute_Error(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	pause := domain.PauseElement{Length: 3}
	expectedSleepArg := fmt.Sprintf("%d", 3*1000)

	mockEsl.On("Execute", "sleep", expectedSleepArg).Return("", errors.New("sleep failed")).Once()
	result := pause.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestRedirectElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	redirect := domain.RedirectElement{URL: "http://example.com/next", Method: "POST"}

	result := redirect.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionRedirect, result.Action)
	assert.Equal(t, "http://example.com/next", result.RedirectURL)
	assert.Equal(t, "POST", result.RedirectMethod)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t) // No ESL calls for Redirect
}

func TestRedirectElement_Execute_EmptyURL(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	redirect := domain.RedirectElement{URL: "  ", Method: "GET"}

	result := redirect.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action) // Changed from ActionHangup to ActionError
	assert.Error(t, result.Err)
	assert.Equal(t, "redirect URL empty", result.Err.Error())
	mockEsl.AssertExpectations(t)
}

func TestRedirectElement_Execute_DefaultMethod(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	redirect := domain.RedirectElement{URL: "http://example.com/default"} // Method is empty

	result := redirect.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionRedirect, result.Action)
	assert.Equal(t, "http://example.com/default", result.RedirectURL)
	assert.Equal(t, "POST", result.RedirectMethod) // Should default to POST
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

// TODO: Add tests for GatherElement and RecordElement once their Execute methods are implemented.
// For now, they return "Not Implemented".
func TestGatherElement_Execute_NotImplemented(t *testing.T) {
    _, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
    gather := domain.GatherElement{} // Fill with minimal required fields if any for GetType etc.
    result := gather.Execute(mockCtx, mockEsl, mockCallSvc)
    assert.Equal(t, domain.ActionContinue, result.Action) // Or ActionError as per actual stub
    assert.Error(t, result.Err)
    assert.Contains(t, result.Err.Error(), "Gather not implemented")
}

func TestRecordElement_Execute_NotImplemented(t *testing.T) {
    _, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
    record := domain.RecordElement{} // Fill with minimal required fields if any
    result := record.Execute(mockCtx, mockEsl, mockCallSvc)
    assert.Equal(t, domain.ActionContinue, result.Action) // Or ActionError
    assert.Error(t, result.Err)
    assert.Contains(t, result.Err.Error(), "Record not implemented")
}

// Ensure all mocks are asserted
func TestMain(m *testing.M) {
	// This is a placeholder if global setup/teardown for mocks is needed.
	// For now, each test handles its own mock assertions.
	// logrus.SetOutput(io.Discard) // Optional: Discard all log output globally for tests
	m.Run()
}
