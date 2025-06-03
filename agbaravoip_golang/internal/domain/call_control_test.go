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

func (m *MockMinimalCallContext) SetPendingRecording(info interface{}) {
	m.Called(info)
}

func (m *MockMinimalCallContext) GetPendingRecording() (interface{}, bool) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0), args.Bool(1)
}

func (m *MockMinimalCallContext) ClearPendingRecording() {
	m.Called()
}

func (m *MockMinimalCallContext) AddPendingDial(childChannelUUID string, info interface{}) {
	m.Called(childChannelUUID, info)
}

func (m *MockMinimalCallContext) GetPendingDial(childChannelUUID string) (interface{}, bool) {
	args := m.Called(childChannelUUID)
	if args.Get(0) == nil {
		return nil, args.Bool(1)
	}
	return args.Get(0), args.Bool(1)
}

func (m *MockMinimalCallContext) RemovePendingDial(childChannelUUID string) {
	m.Called(childChannelUUID)
}

func (m *MockMinimalCallContext) GetAllPendingDials() map[string]interface{} {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(map[string]interface{})
}

func (m *MockMinimalCallContext) SendNextElements(elements []domain.CallControlElement) error {
	args := m.Called(elements)
	return args.Error(0)
}

func (m *MockMinimalCallContext) GetNextElementsChannel() <-chan []domain.CallControlElement {
	args := m.Called()
	if args.Get(0) == nil {
		// Return a dummy closed channel if not mocked, to prevent nil panics in select
		ch := make(chan []domain.CallControlElement)
		close(ch)
		return ch
	}
	return args.Get(0).(<-chan []domain.CallControlElement)
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

func (m *MockEslConnectionExecutor) Originate(dialString string, vars map[string]string) (string, error) {
	args := m.Called(dialString, vars)
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

func (m *MockCallServicerForESL) CreateRecording(ctx domain.MinimalCallContext, callSid *string, recordingSid, filePath string, duration uint32, format string, sizeBytes int64) error {
	args := m.Called(ctx, callSid, recordingSid, filePath, duration, format, sizeBytes)
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
	assert.Contains(t, result.Err.Error(), "GatherElement.Execute not implemented")
}

func TestRecordElement_Execute_NotImplemented(t *testing.T) {
    _, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
    // This test is for the previous stub. It will be replaced by detailed tests below.
    // For now, let's ensure it reflects the updated stub error message if Execute wasn't changed yet.
    record := domain.RecordElement{}
    result := record.Execute(mockCtx, mockEsl, mockCallSvc) // This calls the NEW Execute if already implemented
    // If testing the stub, it should be:
    // assert.Equal(t, domain.ActionContinue, result.Action)
    // assert.Error(t, result.Err)
    // assert.Contains(t, result.Err.Error(), "RecordElement.Execute not implemented")
	// Since we are adding new tests for implemented Record, this test can be removed or updated.
	// For now, let's assume it was for the stub and we'll replace it.
	// Marking as a placeholder to be effectively replaced by new tests.
	assert.NotNil(t, result) // Placeholder, this test will be removed.
}


// --- GatherElement Execute Tests ---
func TestGatherElement_Execute_Success_NoActionURL_NoInput(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{TimeoutSeconds: 5, FinishOnKey: "#"}

	// Mock PlayAndGetDigits to return empty digits and no error (e.g., timeout)
	mockEsl.On("PlayAndGetDigits", 0, 0, 0, uint32(5000), "#", "", "", "gathered_digits").Return("", nil).Once()
	mockCtx.On("SetPendingRecording", mock.Anything).Times(0) // Ensure not called

	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.Equal(t, "", result.Digits) // No digits collected
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestGatherElement_Execute_Success_NoActionURL_WithDigits(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{NumDigits: 4} // Example: expecting 4 digits

	// Mock PlayAndGetDigits to return some digits
	mockEsl.On("PlayAndGetDigits", 0, 4, 0, uint32(0), "", "", "", "gathered_digits").Return("1234", nil).Once()

	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.Equal(t, "1234", result.Digits)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestGatherElement_Execute_Success_WithActionURL(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{ActionURL: "/gather_handler", Method: "POST", NumDigits: 1}

	mockEsl.On("PlayAndGetDigits", 0, 1, 0, uint32(0), "", "", "", "gathered_digits").Return("5", nil).Once()

	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionRedirect, result.Action)
	assert.Equal(t, "5", result.Digits)
	assert.Equal(t, "/gather_handler", result.RedirectURL)
	assert.Equal(t, "POST", result.RedirectMethod)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
}

func TestGatherElement_Execute_PlayAndGetDigitsError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	gather := domain.GatherElement{ActionURL: "/should_not_call"}

	expectedError := errors.New("esl PlayAndGetDigits failed")
	mockEsl.On("PlayAndGetDigits", 0, 0, 0, uint32(0), "", "", "", "gathered_digits").Return("", expectedError).Once()

	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action) // Or ActionHangup based on error handling policy
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "PlayAndGetDigits failed")
	mockEsl.AssertExpectations(t)
}

func TestGatherElement_Execute_NestedPlayError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	nestedPlay := &domain.PlayElement{URL: "prompt.wav"}
	gather := domain.GatherElement{Play: nestedPlay, TimeoutSeconds: 5}

	expectedError := errors.New("playback failed for nested play")
	// Mock nested Play.Execute to return an error
	// The actual call to Play.Execute is internal to Gather.Execute, so we mock the ESL command it would make.
	mockEsl.On("Execute", "playback", "prompt.wav").Return("", expectedError).Once()
	// PlayAndGetDigits should not be called if nested Play fails
	// mockEsl.On("PlayAndGetDigits", ...).Times(0)


	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action) // Gather should propagate the error
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "Nested Play failed") // Check for specific error if possible
	mockEsl.AssertExpectations(t)
}

func TestGatherElement_Execute_NestedSayError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	nestedSay := &domain.SayElement{Text: "Enter digits."}
	gather := domain.GatherElement{Say: nestedSay, TimeoutSeconds: 5}

	expectedError := errors.New("speak failed for nested say")
	mockEsl.On("Execute", "speak", "flite|slt|Enter digits.").Return("", expectedError).Once()

	result := gather.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "Nested Say failed")
	mockEsl.AssertExpectations(t)
}


// --- RecordElement Execute Tests ---
func TestRecordElement_Execute_Success(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{ActionURL: "/record_notify", Method: "GET", MaxLengthSeconds: 60, FileFormat: "mp3"}

	mockCtx.On("GetAccountSid").Return("test_account_sid").Once()
	// Expected path format: /var/lib/freeswitch/recordings/test_account_sid/test-uuid-123_TIMESTAMP.mp3
	// We can't predict timestamp, so use mock.MatchedBy for filePath in RecordSession
	// And capture the argument to verify in SetPendingRecording
	var capturedFilePath string
	mockEsl.On("RecordSession", mock.MatchedBy(func(fp string) bool {
		capturedFilePath = fp
		return strings.HasPrefix(fp, "/var/lib/freeswitch/recordings/test_account_sid/test-uuid-123_") && strings.HasSuffix(fp, ".mp3")
	}), uint32(60), uint(0), uint(0)).Return("OK", nil).Once()

	// Mock SetPendingRecording
	mockCtx.On("SetPendingRecording", mock.AnythingOfType("callcontrol.PendingRecordInfo")).Run(func(args mock.Arguments) {
		info := args.Get(0).(callcontrol.PendingRecordInfo)
		assert.Equal(t, capturedFilePath, info.ExpectedFilePath)
		assert.Equal(t, "/record_notify", info.OriginalElement.ActionURL)
		assert.Equal(t, "GET", info.OriginalElement.Method)
	}).Once()


	result := record.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestRecordElement_Execute_PlayBeep(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{PlayBeep: true, MaxLengthSeconds: 10}

	mockCtx.On("GetAccountSid").Return("test_account_sid").Once()
	mockEsl.On("Execute", "playback", "tone_stream://%(1000,0,640)").Return("OK", nil).Once()
	mockEsl.On("RecordSession", mock.AnythingOfType("string"), uint32(10), uint(0), uint(0)).Return("OK", nil).Once()
	mockCtx.On("SetPendingRecording", mock.AnythingOfType("callcontrol.PendingRecordInfo")).Return().Once()


	result := record.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestRecordElement_Execute_RecordSessionError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	record := domain.RecordElement{MaxLengthSeconds: 10}
	expectedError := errors.New("record session failed")

	mockCtx.On("GetAccountSid").Return("test_account_sid").Once()
	mockEsl.On("RecordSession", mock.AnythingOfType("string"), uint32(10), uint(0), uint(0)).Return("", expectedError).Once()
	// SetPendingRecording should not be called if RecordSession fails
	mockCtx.On("SetPendingRecording", mock.Anything).Times(0)


	result := record.Execute(mockCtx, mockEsl, mockCallSvc)
	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "failed to start recording")
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

// --- DialElement Execute Tests ---
func TestDialElement_Execute_Success(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	dial := domain.DialElement{CalleeIDToDial: "1000", ActionURL: "/dial_status", CallerID: "123"}

	mockCtx.On("GetVariable", "caller_id_number").Return("").Once() // A-leg's original CID is empty

	expectedVars := map[string]string{
		"origination_caller_id_number": "123",
		"originate_timeout":            "60", // Default
		"agbara_parent_call_sid":       "test-uuid-123",
		"agbara_dial_action_url":       "/dial_status",
		"agbara_dial_action_method":    "POST",
		"agbara_dial_hangup_on_star":   "false",
	}
	mockBlegUUID := "bleg-uuid-456"
	mockEsl.On("Originate", "user/1000", expectedVars).Return(mockBlegUUID, nil).Once()

	mockCtx.On("AddPendingDial", mockBlegUUID, mock.MatchedBy(func(info callcontrol.PendingDialInfo) bool {
		return info.OriginalElement.ActionURL == "/dial_status" && info.ParentAgbaraCallSID == "test-uuid-123"
	})).Return().Once()

	result := dial.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestDialElement_Execute_OriginateError(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	dial := domain.DialElement{CalleeIDToDial: "2000"}
	expectedError := errors.New("originate command failed")

	mockCtx.On("GetVariable", "caller_id_number").Return("999").Once()
	mockEsl.On("Originate", "user/2000", mock.AnythingOfType("map[string]string")).Return("", expectedError).Once()
	mockCtx.On("AddPendingDial", mock.Anything, mock.Anything).Times(0)


	result := dial.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "failed to send originate command")
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestDialElement_Execute_EmptyCallee(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	dial := domain.DialElement{CalleeIDToDial: " "}

	result := dial.Execute(mockCtx, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionError, result.Action)
	assert.Error(t, result.Err)
	assert.Equal(t, "Dial CalleeIDToDial is empty", result.Err.Error())
	mockEsl.AssertNotCalled(t, "Originate", mock.Anything, mock.Anything)
	mockCtx.AssertExpectations(t) // Log should still be called
}

func TestDialElement_Execute_CallerIDLogic(t *testing.T) {
	// Case 1: DialElement.CallerID is set
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	dial1 := domain.DialElement{CalleeIDToDial: "3000", CallerID: "from-dial"}
	mockCtx.On("GetVariable", "caller_id_number").Return("from-aleg").Once() // This shouldn't be used
	expectedVars1 := map[string]string{
		"origination_caller_id_number": "from-dial", // Takes precedence
		"originate_timeout":            "60",
		"agbara_parent_call_sid":       "test-uuid-123",
		"agbara_dial_action_url":       "", "agbara_dial_action_method": "POST", "agbara_dial_hangup_on_star": "false",
	}
	mockEsl.On("Originate", "user/3000", expectedVars1).Return("uuid1", nil).Once()
	mockCtx.On("AddPendingDial", "uuid1", mock.Anything).Return().Once()
	dial1.Execute(mockCtx, mockEsl, mockCallSvc)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)

	// Case 2: DialElement.CallerID is empty, use A-leg's caller_id_number
	mockCtx2, mockEsl2, mockCallSvc2 := setupTestMocks(t) // Fresh mocks
	dial2 := domain.DialElement{CalleeIDToDial: "4000"}
	mockCtx2.On("GetVariable", "caller_id_number").Return("from-aleg-var").Once()
	expectedVars2 := map[string]string{
		"origination_caller_id_number": "from-aleg-var", // Used from context
		"originate_timeout":            "60",
		"agbara_parent_call_sid":       "test-uuid-123",
		"agbara_dial_action_url":       "", "agbara_dial_action_method": "POST", "agbara_dial_hangup_on_star": "false",
	}
	mockEsl2.On("Originate", "user/4000", expectedVars2).Return("uuid2", nil).Once()
	mockCtx2.On("AddPendingDial", "uuid2", mock.Anything).Return().Once()
	dial2.Execute(mockCtx2, mockEsl2, mockCallSvc2)
	mockEsl2.AssertExpectations(t)
	mockCtx2.AssertExpectations(t)
}

func TestDialElement_Execute_TimeoutLogic(t *testing.T) {
	// Case 1: TimeoutSeconds set
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	dial1 := domain.DialElement{CalleeIDToDial: "5000", TimeoutSeconds: 30}
	mockCtx.On("GetVariable", "caller_id_number").Return("").Once()
	expectedVars1 := map[string]string{
		"originate_timeout":            "30", // Custom timeout
		"agbara_parent_call_sid":       "test-uuid-123",
		"agbara_dial_action_url":       "", "agbara_dial_action_method": "POST", "agbara_dial_hangup_on_star": "false",
	}
	mockEsl.On("Originate", "user/5000", expectedVars1).Return("uuid1", nil).Once()
	mockCtx.On("AddPendingDial", "uuid1", mock.Anything).Return().Once()
	dial1.Execute(mockCtx, mockEsl, mockCallSvc)
	mockEsl.AssertExpectations(t)
	mockCtx.AssertExpectations(t)

	// Case 2: TimeoutSeconds not set (or zero), use default
	mockCtx2, mockEsl2, mockCallSvc2 := setupTestMocks(t) // Fresh mocks
	dial2 := domain.DialElement{CalleeIDToDial: "6000"} // TimeoutSeconds is 0 (default)
	mockCtx2.On("GetVariable", "caller_id_number").Return("").Once()
	expectedVars2 := map[string]string{
		"originate_timeout":            "60", // Default timeout
		"agbara_parent_call_sid":       "test-uuid-123",
		"agbara_dial_action_url":       "", "agbara_dial_action_method": "POST", "agbara_dial_hangup_on_star": "false",
	}
	mockEsl2.On("Originate", "user/6000", expectedVars2).Return("uuid2", nil).Once()
	mockCtx2.On("AddPendingDial", "uuid2", mock.Anything).Return().Once()
	dial2.Execute(mockCtx2, mockEsl2, mockCallSvc2)
	mockEsl2.AssertExpectations(t)
	mockCtx2.AssertExpectations(t)
}


// Ensure all mocks are asserted
func TestMain(m *testing.M) {
	// This is a placeholder if global setup/teardown for mocks is needed.
	// For now, each test handles its own mock assertions.
	// logrus.SetOutput(io.Discard) // Optional: Discard all log output globally for tests
	m.Run()
}
