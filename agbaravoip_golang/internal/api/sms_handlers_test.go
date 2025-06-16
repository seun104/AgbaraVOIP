package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	// "net/url" // Not used in this snippet, but might be for filter tests
	"testing"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSMSService is a mock type for ISMSService
type MockSMSService struct {
	mock.Mock
}

func (m *MockSMSService) SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, to, from, body, msgSID, actionURL, actionMethod)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.SMSMessage), args.Error(1)
}

func (m *MockSMSService) SendSMSViaAPI(ctx context.Context, accountSid, from, to, body string, statusCallbackURL *string) (*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, from, to, body, statusCallbackURL)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.SMSMessage), args.Error(1)
}

func (m *MockSMSService) GetSMSBySID(ctx context.Context, accountSid string, sid string) (*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, sid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.SMSMessage), args.Error(1)
}

func (m *MockSMSService) ListSMSMessages(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, filters)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).([]*domain.SMSMessage), args.Error(1)
}

func (m *MockSMSService) UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error {
	args := m.Called(ctx, agbaraSid, gatewaySid, status, errorCode, errorMessage, eventTime)
	return args.Error(0)
}

func (m *MockSMSService) RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, to, from, body, inboundGatewayMsgSid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.SMSMessage), args.Error(1)
}


func setupSMSSHandlerTestRouter(service services.ISMSService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Suppress logs

	handler := NewAccountSMSHandler(service, logger)

	router := gin.New() // Use gin.New() for a clean router for tests
	// Mock middleware to set AccountSID, similar to other handler tests
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyAccountSID), "ACtestsmartsid") // Mock authenticated user
		c.Next()
	})

	// Define routes for testing
	// The :account_sid in the path is for Gin to match the route,
	// but the handler uses c.GetString(ContextKeyAccountSID) which is set by the mock middleware.
	accountRoutes := router.Group("/api/v1/accounts/:account_sid_in_path_ignored")
	smsMessagesRoutes := accountRoutes.Group("/sms/messages")
	{
		smsMessagesRoutes.POST("", handler.SendSMS)
		smsMessagesRoutes.GET("", handler.ListSMSMessages)
		smsMessagesRoutes.GET("/:sms_sid", handler.GetSMSMessage)
	}
	return router
}

func TestAccountSMSHandler_SendSMS_Success(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)

	accountSid := "ACtestsmartsid" // This must match what's set in middleware for c.GetString
	reqPayload := SendSMSRequest{
		From: "+15005550006",
		To:   "+15005550007",
		Body: "Hello from TestSendSMS_Success",
	}
	expectedSMS := &domain.SMSMessage{
		SID:        "SMGeneratedSID",
		AccountSID: accountSid,
		From:       reqPayload.From,
		To:         reqPayload.To,
		Body:       reqPayload.Body,
		Status:     domain.SMSStatusQueued,
		Direction:  domain.SMSDirectionOutboundAPI,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockService.On("SendSMSViaAPI", mock.AnythingOfType("*gin.Context"), accountSid, reqPayload.From, reqPayload.To, reqPayload.Body, reqPayload.StatusCallbackURL).Return(expectedSMS, nil)

	// The path for the request needs to fill path parameters, even if one is ignored by test setup.
	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages"
	jsonPayload, _ := json.Marshal(reqPayload)
	req, _ := http.NewRequest(http.MethodPost, requestPath, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp SMSMessageResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedSMS.SID, resp.SID)
	assert.Equal(t, expectedSMS.Body, resp.Body)
	mockService.AssertExpectations(t)
}

func TestAccountSMSHandler_SendSMS_BindingError(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)
	accountSid := "ACtestsmartsid"

	// Missing 'To' field which is required by SendSMSRequest binding:"required"
	reqPayload := map[string]string{"from": "+123", "body": "test"}
	jsonPayload, _ := json.Marshal(reqPayload)
	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages"
	req, _ := http.NewRequest(http.MethodPost, requestPath, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestAccountSMSHandler_SendSMS_ServiceError(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)
	accountSid := "ACtestsmartsid"
	reqPayload := SendSMSRequest{ From: "+1", To: "+2", Body: "fail test" }

	mockService.On("SendSMSViaAPI", mock.AnythingOfType("*gin.Context"), accountSid, reqPayload.From, reqPayload.To, reqPayload.Body, reqPayload.StatusCallbackURL).Return(nil, errors.New("gateway unavailable"))

	jsonPayload, _ := json.Marshal(reqPayload)
	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages"
	req, _ := http.NewRequest(http.MethodPost, requestPath, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	mockService.AssertExpectations(t)
}


func TestAccountSMSHandler_GetSMSMessage_Success(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)

	accountSid := "ACtestsmartsid"
	smsSid := "SM123"
	expectedSMS := &domain.SMSMessage{
		SID:        smsSid,
		AccountSID: accountSid,
		From:       "+1from", To: "+1to", Body: "Test message",
		Status: domain.SMSStatusSent, Direction: domain.SMSDirectionOutboundAPI,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	mockService.On("GetSMSBySID", mock.AnythingOfType("*gin.Context"), accountSid, smsSid).Return(expectedSMS, nil)

	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages/" + smsSid
	req, _ := http.NewRequest(http.MethodGet, requestPath, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp SMSMessageResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedSMS.SID, resp.SID)
	mockService.AssertExpectations(t)
}

func TestAccountSMSHandler_GetSMSMessage_NotFound(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)
	accountSid := "ACtestsmartsid"
	smsSid := "SMNotFound"

	mockService.On("GetSMSBySID", mock.AnythingOfType("*gin.Context"), accountSid, smsSid).Return(nil, domain.ErrNotFound)

	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages/" + smsSid
	req, _ := http.NewRequest(http.MethodGet, requestPath, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	mockService.AssertExpectations(t)
}

func TestAccountSMSHandler_ListSMSMessages_Success(t *testing.T) {
	mockService := new(MockSMSService)
	router := setupSMSSHandlerTestRouter(mockService)
	accountSid := "ACtestsmartsid"

	expectedSMSList := []*domain.SMSMessage{
		{SID: "SM1", AccountSID: accountSid, Body: "msg1", CreatedAt: time.Now()},
		{SID: "SM2", AccountSID: accountSid, Body: "msg2", CreatedAt: time.Now().Add(-time.Hour)},
	}

	expectedFilters := map[string]interface{}{} // For call with no query params

	mockService.On("ListSMSMessages", mock.AnythingOfType("*gin.Context"), accountSid, expectedFilters).Return(expectedSMSList, nil)

	requestPath := "/api/v1/accounts/" + accountSid + "/sms/messages"
	req, _ := http.NewRequest(http.MethodGet, requestPath, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var respList []SMSMessageResponse
	err := json.Unmarshal(rr.Body.Bytes(), &respList)
	assert.NoError(t, err)
	assert.Len(t, respList, 2)
	assert.Equal(t, "SM1", respList[0].SID)
	mockService.AssertExpectations(t)
}


// TODO: Add more tests for ListSMSMessages with filters.
// TODO: Add tests for service returning other types of errors for all handlers.
