package api

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"io" // Required for io.Discard
)

// MockFreeswitchServerService is a mock type for IFreeswitchServerService
type MockFreeswitchServerService struct {
	mock.Mock
}

func (m *MockFreeswitchServerService) CreateFreeswitchServer(host string, port int, password string, outboundAddress string, isActive *bool) (*domain.FreeswitchServer, error) {
	args := m.Called(host, port, password, outboundAddress, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FreeswitchServer), args.Error(1)
}

func (m *MockFreeswitchServerService) GetFreeswitchServerBySID(sid string) (*domain.FreeswitchServer, error) {
	args := m.Called(sid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FreeswitchServer), args.Error(1)
}

func (m *MockFreeswitchServerService) ListFreeswitchServers(filters map[string]interface{}) ([]*domain.FreeswitchServer, error) {
	args := m.Called(filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.FreeswitchServer), args.Error(1)
}

func (m *MockFreeswitchServerService) UpdateFreeswitchServer(sid string, updates map[string]interface{}) (*domain.FreeswitchServer, error) {
	args := m.Called(sid, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.FreeswitchServer), args.Error(1)
}

func (m *MockFreeswitchServerService) DeleteFreeswitchServer(sid string) error {
	args := m.Called(sid)
	return args.Error(0)
}

func setupFreeswitchHandlerTestRouter(service services.IFreeswitchServerService) (*gin.Engine, *logrus.Logger) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs during testing

	handler := NewAdminFreeswitchHandler(service, logger)

	router := gin.Default()
	// Define routes for testing (matching those in server.go for this handler)
	adminRoutes := router.Group("/api/v1/admin/freeswitch-servers")
	{
		adminRoutes.POST("", handler.CreateFreeswitchServer)
		adminRoutes.GET("/:fs_sid", handler.GetFreeswitchServer)
		adminRoutes.GET("", handler.ListFreeswitchServers)
		adminRoutes.PUT("/:fs_sid", handler.UpdateFreeswitchServer)
		adminRoutes.DELETE("/:fs_sid", handler.DeleteFreeswitchServer)
	}
	return router, logger
}

func TestAdminCreateFreeswitchServer_Success(t *testing.T) {
	mockService := new(MockFreeswitchServerService)
	router, _ := setupFreeswitchHandlerTestRouter(mockService)

	isActive := true
	reqPayload := CreateFreeswitchServerRequest{
		Host:            "fs.test.com",
		Port:            8021,
		Password:        "testpass",
		OutboundAddress: "1.2.3.4:5060",
		IsActive:        &isActive,
	}
	expectedServer := &domain.FreeswitchServer{
		SID:      "FSGeneratedSID",
		Host:     reqPayload.Host,
		Port:     reqPayload.Port,
		IsActive: isActive,
		// ... other fields set by service
	}

	mockService.On("CreateFreeswitchServer", reqPayload.Host, reqPayload.Port, reqPayload.Password, reqPayload.OutboundAddress, &isActive).Return(expectedServer, nil)

	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/freeswitch-servers", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var response FreeswitchServerResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedServer.SID, response.SID)
	assert.Equal(t, expectedServer.Host, response.Host)

	mockService.AssertExpectations(t)
}

func TestAdminCreateFreeswitchServer_ValidationFailure(t *testing.T) {
	mockService := new(MockFreeswitchServerService) // Service won't be called
	router, _ := setupFreeswitchHandlerTestRouter(mockService)

	reqPayload := CreateFreeswitchServerRequest{Host: ""} // Invalid payload
	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/freeswitch-servers", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	// Add more assertions for error response body if needed
}

func TestAdminCreateFreeswitchServer_ServiceError(t *testing.T) {
	mockService := new(MockFreeswitchServerService)
	router, _ := setupFreeswitchHandlerTestRouter(mockService)

	isActive := false
	reqPayload := CreateFreeswitchServerRequest{
		Host:     "fs.error.com",
		Port:     8021,
		Password: "err",
		IsActive: &isActive,
	}

	mockService.On("CreateFreeswitchServer", reqPayload.Host, reqPayload.Port, reqPayload.Password, "", &isActive).Return(nil, errors.New("service internal error"))

	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/freeswitch-servers", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}


func TestAdminGetFreeswitchServer_Success(t *testing.T) {
	mockService := new(MockFreeswitchServerService)
	router, _ := setupFreeswitchHandlerTestRouter(mockService)

	fsSID := "FS123"
	expectedServer := &domain.FreeswitchServer{
		SID:       fsSID,
		Host:      "fs.get.com",
		Port:      8021,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockService.On("GetFreeswitchServerBySID", fsSID).Return(expectedServer, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/freeswitch-servers/"+fsSID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response FreeswitchServerResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedServer.SID, response.SID)

	mockService.AssertExpectations(t)
}

func TestAdminGetFreeswitchServer_NotFound(t *testing.T) {
	mockService := new(MockFreeswitchServerService)
	router, _ := setupFreeswitchHandlerTestRouter(mockService)

	fsSID := "FSNotFound"
	mockService.On("GetFreeswitchServerBySID", fsSID).Return(nil, services.ErrFreeswitchServerNotFound)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/freeswitch-servers/"+fsSID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

// TODO: Add tests for List, Update, Delete handlers and other error cases.
// For List, test with and without query parameters.
// For Update, test successful update and validation errors.
