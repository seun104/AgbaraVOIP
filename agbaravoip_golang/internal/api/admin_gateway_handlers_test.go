package api

import (
	"bytes"
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

// MockGatewayService is a mock type for IGatewayService
type MockGatewayService struct {
	mock.Mock
}

func (m *MockGatewayService) CreateGateway(accountSID string, fsServerSID *string, friendlyName string, gatewayString string, codecs []string, retryCount *int, timeoutSeconds *int, routes domain.GatewayRoutes, isEnabled *bool) (*domain.Gateway, error) {
	args := m.Called(accountSID, fsServerSID, friendlyName, gatewayString, codecs, retryCount, timeoutSeconds, routes, isEnabled)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) GetGatewayBySID(accountSID string, sid string) (*domain.Gateway, error) {
	args := m.Called(accountSID, sid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) ListGateways(accountSID string, filters map[string]interface{}) ([]*domain.Gateway, error) {
	args := m.Called(accountSID, filters)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) UpdateGateway(accountSID string, sid string, updates map[string]interface{}) (*domain.Gateway, error) {
	args := m.Called(accountSID, sid, updates)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) DeleteGateway(accountSID string, sid string) error {
	args := m.Called(accountSID, sid)
	return args.Error(0)
}

func (m *MockGatewayService) ListGlobalGateways(filters map[string]interface{}) ([]*domain.Gateway, error) {
	args := m.Called(filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) GetGlobalGatewayBySID(sid string) (*domain.Gateway, error) {
	args := m.Called(sid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) UpdateGlobalGateway(sid string, updates map[string]interface{}) (*domain.Gateway, error) {
	args := m.Called(sid, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Gateway), args.Error(1)
}

func (m *MockGatewayService) DeleteGlobalGateway(sid string) error {
	args := m.Called(sid)
	return args.Error(0)
}


func setupGatewayHandlerTestRouter(service services.IGatewayService) (*gin.Engine, *logrus.Logger) {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs

	handler := NewAdminGatewayHandler(service, logger)

	router := gin.Default()
	adminRoutes := router.Group("/api/v1/admin/gateways")
	{
		adminRoutes.POST("", handler.CreateGateway)
		adminRoutes.GET("/:gw_sid", handler.GetGateway)
		adminRoutes.GET("", handler.ListGateways)
		adminRoutes.PUT("/:gw_sid", handler.UpdateGateway)
		adminRoutes.DELETE("/:gw_sid", handler.DeleteGateway)
	}
	return router, logger
}

func TestAdminCreateGateway_Success(t *testing.T) {
	mockService := new(MockGatewayService)
	router, _ := setupGatewayHandlerTestRouter(mockService)

	isEnabled := true
	reqPayload := CreateGatewayRequest{
		AccountSID:    "ACAdmin123",
		FriendlyName:  "Admin Test GW",
		GatewayString: "sofia/test/admin",
		IsEnabled:     &isEnabled,
	}
	expectedGateway := &domain.Gateway{
		SID:           "GWGeneratedSID",
		AccountSID:    reqPayload.AccountSID,
		FriendlyName:  reqPayload.FriendlyName,
		GatewayString: reqPayload.GatewayString,
		IsEnabled:     isEnabled,
		// ... other fields
	}

	mockService.On("CreateGateway", reqPayload.AccountSID, reqPayload.FreeswitchServerSID, reqPayload.FriendlyName, reqPayload.GatewayString, reqPayload.Codecs, reqPayload.RetryCount, reqPayload.TimeoutSeconds, reqPayload.Routes, reqPayload.IsEnabled).Return(expectedGateway, nil)

	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/gateways", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var response GatewayResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedGateway.SID, response.SID)
	assert.Equal(t, expectedGateway.FriendlyName, response.FriendlyName)

	mockService.AssertExpectations(t)
}

func TestAdminCreateGateway_ValidationFailure_NoAccountSID(t *testing.T) {
	mockService := new(MockGatewayService) // Service might not be called if binding fails early
	router, _ := setupGatewayHandlerTestRouter(mockService)

	// AccountSID is required by the handler logic, even if not by struct binding tag here
	reqPayload := CreateGatewayRequest{FriendlyName: "No Acc SID GW"}
	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/admin/gateways", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	// Optionally assert error message if it's consistent
}

func TestAdminGetGateway_Success(t *testing.T) {
	mockService := new(MockGatewayService)
	router, _ := setupGatewayHandlerTestRouter(mockService)

	gwSID := "GW123"
	expectedGateway := &domain.Gateway{
		SID:          gwSID,
		AccountSID:   "ACSomeAcc",
		FriendlyName: "Fetched GW",
		IsEnabled:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	// Admin handler uses GetGlobalGatewayBySID
	mockService.On("GetGlobalGatewayBySID", gwSID).Return(expectedGateway, nil)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/gateways/"+gwSID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var response GatewayResponse
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedGateway.SID, response.SID)

	mockService.AssertExpectations(t)
}

func TestAdminGetGateway_NotFound(t *testing.T) {
	mockService := new(MockGatewayService)
	router, _ := setupGatewayHandlerTestRouter(mockService)

	gwSID := "GWNotFound"
	mockService.On("GetGlobalGatewayBySID", gwSID).Return(nil, services.ErrGatewayNotFound)

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/admin/gateways/"+gwSID, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}


// TODO: Add tests for ListGateways, UpdateGateway, DeleteGateway handlers.
// Test filtering for ListGateways.
// Test validation and service errors for all endpoints.
