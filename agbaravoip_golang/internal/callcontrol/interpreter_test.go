package callcontrol_test

import (
	"errors"
	"io"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/domain"
)

// --- Mocks needed for interpreter tests ---

// MockMinimalCallContext for interpreter tests
type MockMinimalCallContext struct {
	mock.Mock
}

func (m *MockMinimalCallContext) Log() *logrus.Entry {
	args := m.Called()
	if args.Get(0) == nil {
		entry := logrus.NewEntry(logrus.New())
		entry.Logger.SetOutput(io.Discard)
		return entry
	}
	return args.Get(0).(*logrus.Entry)
}
func (m *MockMinimalCallContext) GetUuid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAccountSid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetApplicationSid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAnswerURL() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetVariable(varName string) string { return m.Called(varName).String(0) }
func (m *MockMinimalCallContext) IsHangupInitiated() bool { return m.Called().Bool(0) }
func (m *MockMinimalCallContext) SetHangupInitiated() { m.Called() }

// MockEslConnectionExecutor for interpreter tests
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
func (m *MockEslConnectionExecutor) ExecuteSofia(command string, cmdArgs ...string) (string, error) { return "", nil } // Stub
func (m *MockEslConnectionExecutor) SendMsg(msg map[string]string) (string, error) { return "", nil } // Stub
func (m *MockEslConnectionExecutor) GetVar(varName string) (string, error) { return "", nil } // Stub
func (m *MockEslConnectionExecutor) Answer() (string, error) { return "", nil } // Stub
func (m *MockEslConnectionExecutor) Hangup(reason string) (string, error) { return "", nil } // Stub

// MockCallServicerForESL for interpreter tests
type MockCallServicerForESL struct {
	mock.Mock
}

func (m *MockCallServicerForESL) UpdateCallStatus(ctx domain.MinimalCallContext, status string, hangupCause string) error {
	return m.Called(ctx, status, hangupCause).Error(0)
}
func (m *MockCallServicerForESL) UpdateCallRecording(ctx domain.MinimalCallContext, recordingPath string, durationSec int, format string) error {
	return m.Called(ctx, recordingPath, durationSec, format).Error(0)
}

// MockCallControlElement for interpreter tests
type MockCallControlElement struct {
	mock.Mock
	TypeName string // To help identify the element in logs/tests
}

func (m *MockCallControlElement) GetType() domain.CallControlAction {
	args := m.Called()
	// Return a string representation that can be converted to CallControlAction if needed,
	// or directly the CallControlAction type. For simplicity, using a fixed one or from TypeName.
	// This might need adjustment based on how deeply GetType is inspected by the interpreter.
	// The interpreter uses it for logging: log.Infof("Executing XML element: %T", element)
	// So, the actual type matters more than this mock method's return for that specific log line.
	// For behavior, it's the Execute mock that's critical.
	return domain.CallControlAction(m.TypeName) // Example, assuming TypeName matches an action
}
func (m *MockCallControlElement) GetActionURL() string { return m.Called().String(0) }
func (m *MockCallControlElement) GetMethod() string    { return m.Called().String(0) }
func (m *MockCallControlElement) Execute(ctx domain.MinimalCallContext, eslConn domain.EslConnectionExecutor, callSvc domain.CallServicerForESL) domain.CallControlResult {
	args := m.Called(ctx, eslConn, callSvc)
	return args.Get(0).(domain.CallControlResult)
}

// --- Test Setup Helper ---
func setupInterpreterTest(t *testing.T) (*MockMinimalCallContext, *MockEslConnectionExecutor, *MockCallServicerForESL, *logrus.Entry) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetOutput(io.Discard)

	mockCtx := new(MockMinimalCallContext)
	mockEsl := new(MockEslConnectionExecutor)
	mockCallSvc := new(MockCallServicerForESL)

	mockCtx.On("Log").Return(logger)
	mockCtx.On("GetUuid").Return("test-interpreter-uuid")
	// Default: not hangup initiated
	mockCtx.On("IsHangupInitiated").Return(false)

	return mockCtx, mockEsl, mockCallSvc, logger
}

// --- ExecuteAgbaraXML Tests ---

func TestExecuteAgbaraXML_EmptyElements(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)
	elements := []domain.CallControlElement{}

	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_SingleElement_Continue(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement := new(MockCallControlElement)
	mockElement.TypeName = "MockSay" // For GetType if it were more sophisticated
	elementResult := domain.CallControlResult{Action: domain.ActionContinue, Err: nil}
	mockElement.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(elementResult).Once()

	elements := []domain.CallControlElement{mockElement}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockElement.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_Sequence_AllContinue(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement1 := new(MockCallControlElement); mockElement1.TypeName = "MockSay"
	mockElement2 := new(MockCallControlElement); mockElement2.TypeName = "MockPlay"
	elementResult := domain.CallControlResult{Action: domain.ActionContinue, Err: nil}

	mockElement1.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(elementResult).Once()
	mockElement2.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(elementResult).Once()

	elements := []domain.CallControlElement{mockElement1, mockElement2}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockElement1.AssertExpectations(t)
	mockElement2.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_RedirectActionStopsProcessing(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement1 := new(MockCallControlElement); mockElement1.TypeName = "MockRedirect"
	mockElement2 := new(MockCallControlElement); mockElement2.TypeName = "MockSayShouldNotExecute" // This one should not be called

	redirectResult := domain.CallControlResult{Action: domain.ActionRedirect, RedirectURL: "http://new.url", RedirectMethod: "POST"}
	mockElement1.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(redirectResult).Once()
	// mockElement2.On("Execute", ...) should not be called

	elements := []domain.CallControlElement{mockElement1, mockElement2}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionRedirect, result.Action)
	assert.Equal(t, "http://new.url", result.RedirectURL)
	assert.Equal(t, "POST", result.RedirectMethod)
	assert.NoError(t, result.Err)
	mockElement1.AssertExpectations(t)
	mockElement2.AssertNotCalled(t, "Execute", mockCtx, mockEsl, mockCallSvc) // Verify second element not executed
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_HangupActionStopsProcessing(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement1 := new(MockCallControlElement); mockElement1.TypeName = "MockHangup"
	mockElement2 := new(MockCallControlElement); mockElement2.TypeName = "MockSayShouldNotExecute"

	hangupResult := domain.CallControlResult{Action: domain.ActionHangup}
	mockElement1.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(hangupResult).Once()

	elements := []domain.CallControlElement{mockElement1, mockElement2}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionHangup, result.Action)
	assert.NoError(t, result.Err)
	mockElement1.AssertExpectations(t)
	mockElement2.AssertNotCalled(t, "Execute", mockCtx, mockEsl, mockCallSvc)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_ElementErrorLeadsToHangup(t *testing.T) {
	// This test reflects the current interpreter.go logic:
	// if result.Err != nil { ... currentResult.Action = domain.ActionHangup; break }
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement1 := new(MockCallControlElement); mockElement1.TypeName = "MockFailingElement"
	mockElement2 := new(MockCallControlElement); mockElement2.TypeName = "MockShouldNotRun"

	expectedError := errors.New("element execution failed")
	errorResult := domain.CallControlResult{Action: domain.ActionContinue, Err: expectedError} // Element itself might return Continue

	mockElement1.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(errorResult).Once()

	elements := []domain.CallControlElement{mockElement1, mockElement2}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionHangup, result.Action) // Interpreter changes action to Hangup on any error
	assert.Equal(t, expectedError, result.Err)
	mockElement1.AssertExpectations(t)
	mockElement2.AssertNotCalled(t, "Execute", mockCtx, mockEsl, mockCallSvc)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_HangupInitiatedMidway(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElement1 := new(MockCallControlElement); mockElement1.TypeName = "MockSay"
	mockElement2 := new(MockCallControlElement); mockElement2.TypeName = "MockPlayShouldNotExecute"

	continueResult := domain.CallControlResult{Action: domain.ActionContinue}

	// First element executes fine
	mockElement1.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(continueResult).Once()

	// After first element, IsHangupInitiated will return true
	mockCtx.On("IsHangupInitiated").Return(false).Once() // For first check
	mockCtx.On("IsHangupInitiated").Return(true).Once()  // For second check (before mockElement2)

	elements := []domain.CallControlElement{mockElement1, mockElement2}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionHangup, result.Action) // Should switch to Hangup
	assert.NoError(t, result.Err) // No error from elements, hangup is due to external signal

	mockElement1.AssertExpectations(t)
	mockElement2.AssertNotCalled(t, "Execute", mockCtx, mockEsl, mockCallSvc) // Second element should not run
	// IsHangupInitiated was called twice as expected by the On().Return() sequence
	// Log and GetUuid are also asserted by mockCtx.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_NilElementInSlice(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

	mockElementValid := new(MockCallControlElement); mockElementValid.TypeName = "MockSay"
	continueResult := domain.CallControlResult{Action: domain.ActionContinue}
	mockElementValid.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(continueResult).Once()

	elements := []domain.CallControlElement{nil, mockElementValid} // Nil element first
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	// The interpreter.go has a nil check:
	// if element == nil { callCtx.Log().Warn(...); continue }
	// So, it should skip the nil and execute the valid one.
	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockElementValid.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_AllNilElements(t *testing.T) {
	mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)
	elements := []domain.CallControlElement{nil, nil}
	result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

	assert.Equal(t, domain.ActionContinue, result.Action)
	assert.NoError(t, result.Err)
	mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_HangupInitiatedExternallyAfterLoop(t *testing.T) {
    mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

    mockElement := new(MockCallControlElement); mockElement.TypeName = "MockSay"
    continueResult := domain.CallControlResult{Action: domain.ActionContinue}
    mockElement.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(continueResult).Once()

    // IsHangupInitiated is false during the loop, but true for the final check in ExecuteAgbaraXML
    mockCtx.On("IsHangupInitiated").Return(false).Once() // During loop
    mockCtx.On("IsHangupInitiated").Return(true).Once()  // After loop in ExecuteAgbaraXML's final check

    elements := []domain.CallControlElement{mockElement}
    result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

    assert.Equal(t, domain.ActionHangup, result.Action)
    assert.NoError(t, result.Err) // Error should be nilled by the external hangup logic
    mockElement.AssertExpectations(t)
    mockCtx.AssertExpectations(t)
}

func TestExecuteAgbaraXML_ElementErrorAndExternalHangup(t *testing.T) {
    // Scenario: Element returns error (which interpreter converts to ActionHangup),
    // AND IsHangupInitiated also becomes true.
    mockCtx, mockEsl, mockCallSvc, _ := setupInterpreterTest(t)

    mockElement := new(MockCallControlElement); mockElement.TypeName = "MockFailingElement"
    expectedError := errors.New("element failed")
    errorResult := domain.CallControlResult{Action: domain.ActionContinue, Err: expectedError}
    mockElement.On("Execute", mockCtx, mockEsl, mockCallSvc).Return(errorResult).Once()

    // IsHangupInitiated is false during the loop (so element error is processed first),
    // but true for the final check.
    mockCtx.On("IsHangupInitiated").Return(false).Once() // During loop (before element error processing)
    mockCtx.On("IsHangupInitiated").Return(true).Once()  // After loop in ExecuteAgbaraXML's final check

    elements := []domain.CallControlElement{mockElement}
    result := callcontrol.ExecuteAgbaraXML(mockCtx, elements, mockEsl, mockCallSvc)

    // The interpreter's logic: element error sets ActionHangup and the error.
    // The final check for IsHangupInitiated will see ActionHangup is already set.
    // It will then nil the error.
    assert.Equal(t, domain.ActionHangup, result.Action)
    assert.NoError(t, result.Err) // Error is nilled by the final IsHangupInitiated block
    mockElement.AssertExpectations(t)
    mockCtx.AssertExpectations(t)
}
