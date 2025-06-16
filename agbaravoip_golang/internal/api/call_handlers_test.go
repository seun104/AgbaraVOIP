package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	// "time" // Not directly used in the initial set of tests, but good for future

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock IApplicationService ---
type MockApplicationService struct {
	mock.Mock
}

func (m *MockApplicationService) CreateApplication(accountSid string, app *domain.Application) (*domain.Application, error) {
	args := m.Called(accountSid, app)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *MockApplicationService) GetApplicationBySID(accountSid string, appSid string) (*domain.Application, error) {
	args := m.Called(accountSid, appSid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *MockApplicationService) ListApplications(accountSid string) ([]*domain.Application, error) {
	args := m.Called(accountSid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Application), args.Error(1)
}

func (m *MockApplicationService) UpdateApplication(accountSid string, appSid string, updates map[string]interface{}) (*domain.Application, error) {
	args := m.Called(accountSid, appSid, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *MockApplicationService) DeleteApplication(accountSid string, appSid string) error {
	args := m.Called(accountSid, appSid)
	return args.Error(0)
}
func (m *MockApplicationService) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Application), args.Error(1)
}


// --- Mock ICallService ---
type MockCallService struct {
	mock.Mock
}

func (m *MockCallService) OriginateCall(accountSid string, appSidOrNil *string, fromNum string, toNum string, answerURL string, timeoutSeconds *int) (*domain.Call, error) {
	args := m.Called(accountSid, appSidOrNil, fromNum, toNum, answerURL, timeoutSeconds)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Call), args.Error(1)
}

func (m *MockCallService) GetCallBySID(accountSid string, callSid string) (*domain.Call, error) {
	args := m.Called(accountSid, callSid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Call), args.Error(1)
}

func (m *MockCallService) ListCalls(accountSid string, filters map[string]interface{}) ([]*domain.Call, error) {
	args := m.Called(accountSid, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Call), args.Error(1)
}

func (m *MockCallService) PlayAudioOnCall(ctx context.Context, accountSid, callSid string, playURL string, loop int, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, playURL, loop, legs)
	return args.String(0), args.Error(1)
}

func (m *MockCallService) SayTextOnCall(ctx context.Context, accountSid, callSid string, text string, language *string, voice *string, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, text, language, voice, legs)
	return args.String(0), args.Error(1)
}

func (m *MockCallService) SendDTMFOnCall(ctx context.Context, accountSid, callSid string, digits string, durationMs *int, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, digits, durationMs, legs)
	return args.String(0), args.Error(1)
}

func (m *MockCallService) StartRecordingCall(ctx context.Context, accountSid, callSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (string, string, error) {
	args := m.Called(ctx, accountSid, callSid, fileName, maxDurationSec, format, playBeep)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockCallService) StopRecordingCall(ctx context.Context, accountSid, callSid string, recordingNameOrUUID string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, recordingNameOrUUID)
	return args.String(0), args.Error(1)
}

func (m *MockCallService) HangupCall(ctx context.Context, accountSid, callSid string, cause string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, cause)
	return args.String(0), args.Error(1)
}


// --- Test Router Setup ---
func setupCallHandlerTestRouter(callService services.ICallService, appService services.IApplicationService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := NewCallHandler(callService, appService, logger)

	router := gin.New() // Use gin.New() for a clean router for tests
	router.Use(func(c *gin.Context) { // Mock JWT and AccountAccess middleware
		c.Set(string(ContextKeyAccountSID), "ACtestmockaccountSID") // Set AccountSID from JWT
		// If AccountAccessMiddleware logic needs c.Param("account_sid") to match, set it here too
		// For these live call control tests, the :account_sid is part of the path.
		c.Next()
	})

	// Mimic server.go route setup for call control handlers
	// Base path: /api/v1/accounts/:account_sid/calls/:call_sid/
	// We need to ensure the :account_sid param is available if AccountAccessMiddleware relies on it,
	// but these handlers use c.GetString(ContextKeyAccountSID) which we set above.
	// The :account_sid in the path is mainly for routing structure.
	basePath := "/api/v1/accounts/:account_sid/calls/:call_sid"

	router.POST(basePath+"/play", handler.PlayAudio)
	router.POST(basePath+"/say", handler.SayText)
	router.POST(basePath+"/dtmf", handler.SendDTMF)
	router.POST(basePath+"/record", handler.RecordAction)
	router.POST(basePath+"/hangup", handler.HangupLiveCall)

	return router
}

// --- PlayAudio Tests ---
func TestCallHandler_PlayAudio_Success(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService) // Needed for NewCallHandler
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)

	accountSid := "ACtestmockaccountSID" // This must match what's set in middleware
	callSid := "CAtestcallSID"
	playURL := "http://example.com/audio.wav"
	expectedJobID := "job-uuid-123"

	reqBody := CallPlayRequest{URL: playURL, Loop: new(int)} // Default loop 0
	*reqBody.Loop = 1 // Explicitly play once
	jsonBody, _ := json.Marshal(reqBody)

	// accountSid, callSid, playURL, loop int, legs string
	mockCallSvc.On("PlayAudioOnCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, playURL, 1, "aleg").Return(expectedJobID, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/play", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp CallActionResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, expectedJobID, resp.JobID)
	assert.Equal(t, callSid, resp.CallSID)
	mockCallSvc.AssertExpectations(t)
}

func TestCallHandler_PlayAudio_BindingError(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService)
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)
	accountSid := "ACtestmockaccountSID"
	callSid := "CAtestcallSID"

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/play", bytes.NewBufferString(`{"url": "not_a_url", "loop": "not_an_int"}`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCallHandler_PlayAudio_CallNotFound(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService)
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)
	accountSid := "ACtestmockaccountSID"
	callSid := "CAtestcallSID"
	playURL := "http://example.com/audio.wav"
	reqBody := CallPlayRequest{URL: playURL}
	jsonBody, _ := json.Marshal(reqBody)

	mockCallSvc.On("PlayAudioOnCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, playURL, 1, "aleg").Return("", services.ErrCallNotFound_CS)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/play", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockCallSvc.AssertExpectations(t)
}

func TestCallHandler_PlayAudio_ESLError(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService)
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)
	accountSid := "ACtestmockaccountSID"
	callSid := "CAtestcallSID"
	playURL := "http://example.com/audio.wav"
	reqBody := CallPlayRequest{URL: playURL}
	jsonBody, _ := json.Marshal(reqBody)

	mockCallSvc.On("PlayAudioOnCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, playURL, 1, "aleg").Return("", services.ErrESLCommandFailed_CS)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/play", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	mockCallSvc.AssertExpectations(t)
}


// --- HangupLiveCall Tests ---
func TestCallHandler_HangupLiveCall_Success(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService)
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)
	accountSid := "ACtestmockaccountSID"
	callSid := "CAtestcallSIDtohangup"
	expectedJobID := "job-hangup-456"

	mockCallSvc.On("HangupCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, "").Return(expectedJobID, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/hangup", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp CallActionResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, expectedJobID, resp.JobID)
	mockCallSvc.AssertExpectations(t)
}

func TestCallHandler_HangupLiveCall_NotFound(t *testing.T) {
	mockCallSvc := new(MockCallService)
	mockAppSvc := new(MockApplicationService)
	router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)
	accountSid := "ACtestmockaccountSID"
	callSid := "CAcallnotfound"

	mockCallSvc.On("HangupCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, "").Return("", services.ErrCallNotFound_CS)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/hangup", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockCallSvc.AssertExpectations(t)
}

// --- RecordAction Tests (Start) ---
func TestCallHandler_RecordAction_Start_Success(t *testing.T) {
    mockCallSvc := new(MockCallService)
    mockAppSvc := new(MockApplicationService)
    router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)

    accountSid := "ACtestmockaccountSID"
    callSid := "CArecordStart"
    expectedJobID := "job-record-start-789"
    expectedRecordingName := "recfile.wav"

    reqBody := CallRecordRequest{Action: CallRecordActionStart, Format: new(string)}
    *reqBody.Format = "wav" // Example optional param
    jsonBody, _ := json.Marshal(reqBody)

    mockCallSvc.On("StartRecordingCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, reqBody.FileName, reqBody.MaxDurationSeconds, reqBody.Format, reqBody.PlayBeep).Return(expectedRecordingName, expectedJobID, nil)

    req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/record", bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")

    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusAccepted, rr.Code)
    var resp CallActionResponse
    err := json.Unmarshal(rr.Body.Bytes(), &resp)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
    assert.Equal(t, expectedJobID, resp.JobID)
    assert.Contains(t, resp.Message, expectedRecordingName)
    mockCallSvc.AssertExpectations(t)
}

// --- RecordAction Tests (Stop) ---
func TestCallHandler_RecordAction_Stop_Success(t *testing.T) {
    mockCallSvc := new(MockCallService)
    mockAppSvc := new(MockApplicationService)
    router := setupCallHandlerTestRouter(mockCallSvc, mockAppSvc)

    accountSid := "ACtestmockaccountSID"
    callSid := "CArecordStop"
    recordingToStop := callSid // Default name used by handler if FileName is nil
    expectedJobID := "job-record-stop-012"

    reqBody := CallRecordRequest{Action: CallRecordActionStop}
    jsonBody, _ := json.Marshal(reqBody)

    mockCallSvc.On("StopRecordingCall", mock.AnythingOfType("*gin.Context"), accountSid, callSid, recordingToStop).Return(expectedJobID, nil)

    req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/calls/"+callSid+"/record", bytes.NewBuffer(jsonBody))
    req.Header.Set("Content-Type", "application/json")

    rr := httptest.NewRecorder()
    router.ServeHTTP(rr, req)

    assert.Equal(t, http.StatusAccepted, rr.Code)
    var resp CallActionResponse
    err := json.Unmarshal(rr.Body.Bytes(), &resp)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
    assert.Equal(t, expectedJobID, resp.JobID)
    mockCallSvc.AssertExpectations(t)
}


// TODO: Add more tests for SayText, SendDTMF handlers.
// Add more error path tests for RecordAction (e.g. invalid action, service errors).
// Consider testing default values for optional parameters in requests.
// Test with specific values for legs, language, voice etc.
