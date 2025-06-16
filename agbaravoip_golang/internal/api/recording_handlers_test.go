package api

import (
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

// MockRecordingService is a mock type for IRecordingService
type MockRecordingService struct {
	mock.Mock
}

func (m *MockRecordingService) GetRecordingBySID(ctx context.Context, accountSid string, recordingSid string) (*domain.Recording, error) {
	args := m.Called(ctx, accountSid, recordingSid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Recording), args.Error(1)
}

func (m *MockRecordingService) ListRecordings(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.Recording, error) {
	args := m.Called(ctx, accountSid, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Recording), args.Error(1)
}

func (m *MockRecordingService) DeleteRecording(ctx context.Context, accountSid string, recordingSid string) error {
	args := m.Called(ctx, accountSid, recordingSid)
	return args.Error(0)
}

func setupRecordingHandlerTestRouter(service services.IRecordingService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs

	handler := NewRecordingHandler(service, logger)

	router := gin.New() // Use gin.New() for a clean router for tests
	// Mock middleware to set AccountSID
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyAccountSID), "ACtestrecordersid")
		c.Next()
	})

	// Define routes for testing
	// The :account_sid in the path is for Gin to match the route,
	// but the handler uses c.GetString(ContextKeyAccountSID) which is set by the mock middleware.
	accountRoutes := router.Group("/api/v1/accounts/:account_sid_in_path_ignored")
	recordingsRoutes := accountRoutes.Group("/recordings")
	{
		recordingsRoutes.GET("", handler.ListRecordings)
		recordingsRoutes.GET("/:recording_sid", handler.GetRecording)
		recordingsRoutes.DELETE("/:recording_sid", handler.DeleteRecording)
	}
	return router
}

func TestRecordingHandler_GetRecording_Success(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)

	accountSid := "ACtestrecordersid" // Must match the one set in middleware
	recordingSid := "RE123"
	expectedRec := &domain.Recording{
		SID:        recordingSid,
		AccountSID: accountSid,
		FilePath:   "/recs/rec1.wav",
		Format:     "wav",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	mockService.On("GetRecordingBySID", mock.AnythingOfType("*gin.Context"), accountSid, recordingSid).Return(expectedRec, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/recordings/"+recordingSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp RecordingResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedRec.SID, resp.SID)
	mockService.AssertExpectations(t)
}

func TestRecordingHandler_GetRecording_NotFound(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)
	accountSid := "ACtestrecordersid"
	recordingSid := "RENotFound"

	mockService.On("GetRecordingBySID", mock.AnythingOfType("*gin.Context"), accountSid, recordingSid).Return(nil, services.ErrRecordingNotFound)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/recordings/"+recordingSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

func TestRecordingHandler_ListRecordings_Success(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)
	accountSid := "ACtestrecordersid"

	expectedRecs := []*domain.Recording{
		{SID: "RE1", AccountSID: accountSid, FilePath: "path1.wav"},
		{SID: "RE2", AccountSID: accountSid, FilePath: "path2.mp3"},
	}
	expectedFilters := map[string]interface{}{}
	mockService.On("ListRecordings", mock.AnythingOfType("*gin.Context"), accountSid, expectedFilters).Return(expectedRecs, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/accounts/"+accountSid+"/recordings", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var respList []RecordingResponse
	err := json.Unmarshal(rr.Body.Bytes(), &respList)
	assert.NoError(t, err)
	assert.Len(t, respList, 2)
	mockService.AssertExpectations(t)
}

func TestRecordingHandler_ListRecordings_WithFilter(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)
	accountSid := "ACtestrecordersid"
	callSidFilter := "CAfilter123"

	expectedRecs := []*domain.Recording{
		{SID: "RE1", AccountSID: accountSid, CallSID: &callSidFilter, FilePath: "path1.wav"},
	}

	mockService.On("ListRecordings", mock.AnythingOfType("*gin.Context"), accountSid, mock.MatchedBy(func(filters map[string]interface{}) bool {
		return filters["call_sid"] == callSidFilter
	})).Return(expectedRecs, nil)

	reqUrl := "/api/v1/accounts/"+accountSid+"/recordings?call_sid="+callSidFilter
	req, _ := http.NewRequest(http.MethodGet, reqUrl, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var respList []RecordingResponse
	err := json.Unmarshal(rr.Body.Bytes(), &respList)
	assert.NoError(t, err)
	assert.Len(t, respList, 1)
	assert.NotNil(t, respList[0].CallSID) // Ensure CallSID is present
	assert.Equal(t, callSidFilter, *respList[0].CallSID) // Dereference pointer for assertion
	mockService.AssertExpectations(t)
}


func TestRecordingHandler_DeleteRecording_Success(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)
	accountSid := "ACtestrecordersid"
	recordingSid := "REtoDelete"

	mockService.On("DeleteRecording", mock.AnythingOfType("*gin.Context"), accountSid, recordingSid).Return(nil)

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/accounts/"+accountSid+"/recordings/"+recordingSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	mockService.AssertExpectations(t)
}

func TestRecordingHandler_DeleteRecording_NotFound(t *testing.T) {
	mockService := new(MockRecordingService)
	router := setupRecordingHandlerTestRouter(mockService)
	accountSid := "ACtestrecordersid"
	recordingSid := "RENotFoundDelete"

	mockService.On("DeleteRecording", mock.AnythingOfType("*gin.Context"), accountSid, recordingSid).Return(services.ErrRecordingNotFound)

	req, _ := http.NewRequest(http.MethodDelete, "/api/v1/accounts/"+accountSid+"/recordings/"+recordingSid, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

// TODO: Add tests for service returning other types of errors for all handlers.
// (e.g. services.ErrRecordingDeletionFailed for DeleteRecording)
