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
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockConferenceService is a mock type for IConferenceService
type MockConferenceService struct {
	mock.Mock
}

// Implement all IConferenceService methods for the mock
func (m *MockConferenceService) ListConferences(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Conference, error) {
	args := m.Called(ctx, accountSid, filters)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]*domain.Conference), args.Error(1)
}
func (m *MockConferenceService) GetConferenceBySID(ctx context.Context, accountSid string, confSid string) (*domain.Conference, error) {
	args := m.Called(ctx, accountSid, confSid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Conference), args.Error(1)
}
func (m *MockConferenceService) GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) {
    args := m.Called(ctx, accountSid, name); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Conference), args.Error(1)
}
func (m *MockConferenceService) ListParticipants(ctx context.Context, accountSid string, confSid string) ([]*domain.ConferenceParticipant, error) {
	args := m.Called(ctx, accountSid, confSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).([]*domain.ConferenceParticipant), args.Error(1)
}
func (m *MockConferenceService) GetParticipant(ctx context.Context, accountSid string, confSid string, participantSid string) (*domain.ConferenceParticipant, error) {
	args := m.Called(ctx, accountSid, confSid, participantSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.ConferenceParticipant), args.Error(1)
}
func (m *MockConferenceService) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) {
    args := m.Called(ctx, callSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.ConferenceParticipant), args.Error(1)
}
func (m *MockConferenceService) UpdateConferenceStatus(ctx context.Context, accountSid string, confSid string, status domain.ConferenceStatus) error {
    args := m.Called(ctx, accountSid, confSid, status); return args.Error(0)
}
func (m *MockConferenceService) EndConference(ctx context.Context, accountSid string, confSid string, endTime time.Time) error {
    args := m.Called(ctx, accountSid, confSid, endTime); return args.Error(0)
}
func (m *MockConferenceService) UpdateParticipantMuteStatus(ctx context.Context, accountSid string, confSid string, pSid string, isMuted bool) error {
    args := m.Called(ctx, accountSid, confSid, pSid, isMuted); return args.Error(0)
}
func (m *MockConferenceService) UpdateParticipantModeratorStatus(ctx context.Context, accountSid string, confSid string, pSid string, isModerator bool) error {
    args := m.Called(ctx, accountSid, confSid, pSid, isModerator); return args.Error(0)
}
func (m *MockConferenceService) RemoveParticipant(ctx context.Context, accountSid string, confSid string, pSid string, leaveTime time.Time) error {
    args := m.Called(ctx, accountSid, confSid, pSid, leaveTime); return args.Error(0)
}
func (m *MockConferenceService) PlayAudioInConference(ctx context.Context, accountSid string, confSid string, playURL string, loop int) (string, error) {
	args := m.Called(ctx, accountSid, confSid, playURL, loop); return args.String(0), args.Error(1)
}
func (m *MockConferenceService) SayTextInConference(ctx context.Context, accountSid string, confSid string, text string, language *string, voice *string) (string, error) {
	args := m.Called(ctx, accountSid, confSid, text, language, voice); return args.String(0), args.Error(1)
}
func (m *MockConferenceService) StartRecordingConference(ctx context.Context, accountSid string, confSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (string, string, error) {
	args := m.Called(ctx, accountSid, confSid, fileName, maxDurationSec, format, playBeep); return args.String(0), args.String(1), args.Error(2)
}
func (m *MockConferenceService) StopRecordingConference(ctx context.Context, accountSid string, confSid string, recordingNameOrUUID string) (string, error) {
	args := m.Called(ctx, accountSid, confSid, recordingNameOrUUID); return args.String(0), args.Error(1)
}
func (m *MockConferenceService) MuteParticipantInConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string, mute bool) (string, error) {
	args := m.Called(ctx, accountSid, confSid, participantCallSidOrMemberID, mute); return args.String(0), args.Error(1)
}
func (m *MockConferenceService) KickParticipantFromConference(ctx context.Context, accountSid string, confSid string, participantCallSidOrMemberID string) (string, error) {
	args := m.Called(ctx, accountSid, confSid, participantCallSidOrMemberID); return args.String(0), args.Error(1)
}

// MockCallServiceForConfHandler is a mock type for ICallService for ConferenceHandler dependency
type MockCallServiceForConfHandler struct {
	mock.Mock
}
func (m *MockCallServiceForConfHandler) OriginateCall(accountSid string, appSidOrNil *string, fromNum string, toNum string, answerURL string, timeoutSeconds *int) (*domain.Call, error) {
	args := m.Called(accountSid, appSidOrNil, fromNum, toNum, answerURL, timeoutSeconds); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Call), args.Error(1)
}
func (m *MockCallServiceForConfHandler) GetCallBySID(accountSid string, callSid string) (*domain.Call, error) {
	args := m.Called(accountSid, callSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Call), args.Error(1)
}
func (m *MockCallServiceForConfHandler) ListCalls(accountSid string, filters map[string]interface{}) ([]*domain.Call, error) {
	args := m.Called(accountSid, filters); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).([]*domain.Call), args.Error(1)
}
func (m *MockCallServiceForConfHandler) PlayAudioOnCall(ctx context.Context, accountSid, callSid string, playURL string, loop int, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, playURL, loop, legs); return args.String(0), args.Error(1)
}
func (m *MockCallServiceForConfHandler) SayTextOnCall(ctx context.Context, accountSid, callSid string, text string, language *string, voice *string, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, text, language, voice, legs); return args.String(0), args.Error(1)
}
func (m *MockCallServiceForConfHandler) SendDTMFOnCall(ctx context.Context, accountSid, callSid string, digits string, durationMs *int, legs string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, digits, durationMs, legs); return args.String(0), args.Error(1)
}
func (m *MockCallServiceForConfHandler) StartRecordingCall(ctx context.Context, accountSid, callSid string, fileName *string, maxDurationSec *int, format *string, playBeep *bool) (string, string, error) {
	args := m.Called(ctx, accountSid, callSid, fileName, maxDurationSec, format, playBeep); return args.String(0), args.String(1), args.Error(2)
}
func (m *MockCallServiceForConfHandler) StopRecordingCall(ctx context.Context, accountSid, callSid string, recordingNameOrUUID string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, recordingNameOrUUID); return args.String(0), args.Error(1)
}
func (m *MockCallServiceForConfHandler) HangupCall(ctx context.Context, accountSid, callSid string, cause string) (string, error) {
	args := m.Called(ctx, accountSid, callSid, cause); return args.String(0), args.Error(1)
}


func setupConferenceHandlerTestRouter(confService services.IConferenceService, callSvc services.ICallService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	handler := NewConferenceHandler(confService, callSvc, logger)

	router := gin.New() // Using gin.New() for a clean router for tests
	router.Use(func(c *gin.Context) { // Mock JWT middleware setting account SID
		c.Set(string(ContextKeyAccountSID), "ACtestconfSID")
		c.Next()
	})

	// Define routes for testing
	// The :account_sid in the path is for Gin to match the route,
	// but the handler uses c.GetString(ContextKeyAccountSID) which is set by the mock middleware.
	accRoutes := router.Group("/api/v1/accounts/:account_sid_in_path_ignored")
	confRoutes := accRoutes.Group("/conferences")
	{
		confRoutes.GET("", handler.ListConferences)
		confRoutes.GET("/:conf_sid", handler.GetConference)
		confRoutes.POST("/:conf_sid/play", handler.ConferencePlayAudio)
		confRoutes.POST("/:conf_sid/say", handler.ConferenceSayText)
		confRoutes.POST("/:conf_sid/record", handler.ConferenceRecordAction)

		pRoutes := confRoutes.Group("/:conf_sid/participants")
		{
			pRoutes.GET("", handler.ListParticipants)
			pRoutes.GET("/:participant_sid", handler.GetParticipant)
			pRoutes.PUT("/:participant_call_sid/mute", handler.ParticipantMute)
			pRoutes.POST("/:participant_call_sid/kick", handler.ParticipantKick)
		}
	}
	return router
}

func TestConferenceHandler_ListConferences_Success(t *testing.T) {
	mockConfService := new(MockConferenceService)
	mockCallService := new(MockCallServiceForConfHandler)
	router := setupConferenceHandlerTestRouter(mockConfService, mockCallService)

	accountSid := "ACtestconfSID" // Must match the one set in middleware
	expectedConfs := []*domain.Conference{
		{SID: "CF1", AccountSID: accountSid, FriendlyName: "Conf Room 1"},
	}
	mockConfService.On("ListConferences", mock.AnythingOfType("*gin.Context"), accountSid, mock.AnythingOfType("map[string]interface {}")).Return(expectedConfs, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/conferences", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var respList []ConferenceResponse
	err := json.Unmarshal(rr.Body.Bytes(), &respList)
	assert.NoError(t, err)
	assert.Len(t, respList, 1)
	assert.Equal(t, "CF1", respList[0].SID)
	mockConfService.AssertExpectations(t)
}

func TestConferenceHandler_GetConference_Success(t *testing.T) {
	mockConfService := new(MockConferenceService)
	mockCallService := new(MockCallServiceForConfHandler)
	router := setupConferenceHandlerTestRouter(mockConfService, mockCallService)

	accountSid := "ACtestconfSID"
	confSid := "CF123"
	expectedConf := &domain.Conference{ SID: confSid, AccountSID: accountSid, FriendlyName: "Test Conf" }
	mockConfService.On("GetConferenceBySID", mock.AnythingOfType("*gin.Context"), accountSid, confSid).Return(expectedConf, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/conferences/"+confSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp ConferenceResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedConf.SID, resp.SID)
	mockConfService.AssertExpectations(t)
}

func TestConferenceHandler_GetConference_NotFound(t *testing.T) {
	mockConfService := new(MockConferenceService)
	mockCallService := new(MockCallServiceForConfHandler)
	router := setupConferenceHandlerTestRouter(mockConfService, mockCallService)

	accountSid := "ACtestconfSID"
	confSid := "CFNotFound"
	mockConfService.On("GetConferenceBySID", mock.AnythingOfType("*gin.Context"), accountSid, confSid).Return(nil, domain.ErrNotFound)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/conferences/"+confSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockConfService.AssertExpectations(t)
}


func TestConferenceHandler_ConferencePlayAudio_Success(t *testing.T) {
	mockConfService := new(MockConferenceService)
	mockCallService := new(MockCallServiceForConfHandler)
	router := setupConferenceHandlerTestRouter(mockConfService, mockCallService)

	accountSid := "ACtestconfSID"
	confSid := "CF123"
	// Default loop for handler is 1 if req.Loop is nil, or actual value if provided (0 becomes 1)
	reqBody := ConferenceControlPlayRequest{URL: "http://example.com/audio.mp3"}
	expectedJobID := "job-play-123"

	mockConfService.On("PlayAudioInConference", mock.AnythingOfType("*gin.Context"), accountSid, confSid, reqBody.URL, 1).Return(expectedJobID, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/accounts/"+accountSid+"/conferences/"+confSid+"/play", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp CallActionResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, expectedJobID, resp.JobID)
	// Note: The handler currently puts confSid in resp.CallSID. This test reflects that.
	assert.Equal(t, confSid, resp.CallSID)
	mockConfService.AssertExpectations(t)
}

func TestConferenceHandler_ParticipantMute_Success(t *testing.T) {
	mockConfService := new(MockConferenceService)
	mockCallService := new(MockCallServiceForConfHandler)
	router := setupConferenceHandlerTestRouter(mockConfService, mockCallService)

	accountSid := "ACtestconfSID"
	confSid := "CF123"
	participantCallSid := "CAparticipant1"
	muteState := true
	reqBody := ParticipantMuteRequest{Mute: &muteState}
	expectedJobID := "job-mute-123"

	mockConfService.On("MuteParticipantInConference", mock.AnythingOfType("*gin.Context"), accountSid, confSid, participantCallSid, muteState).Return(expectedJobID, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/accounts/"+accountSid+"/conferences/"+confSid+"/participants/"+participantCallSid+"/mute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
	var resp CallActionResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, expectedJobID, resp.JobID)
	mockConfService.AssertExpectations(t)
}


// TODO: Add more tests for:
// - ListParticipants, GetParticipant (Success, NotFound)
// - Other live control: SayText, RecordAction (start/stop), ParticipantKick
// - Binding errors for all POST/PUT handlers
// - Service errors (e.g., ESL error, conference invalid state) for live control handlers.
// - Filtering for ListConferences.
// - Test specific values for optional parameters in live control DTOs.
// - Test error mapping in mapConferenceControlError helper.
