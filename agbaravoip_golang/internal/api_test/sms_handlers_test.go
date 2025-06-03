package api_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time" // For CallServicerForESL method signatures if needed

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/api"
	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/domain"
	// We need a mock for services.CallServicerForESL
	// Defining a local one for this test file for methods used by SMSHandler
)

// Local Mock for services.CallServicerForESL focusing on methods used by SMSHandler
type MockSmsHandlerService struct {
	mock.Mock
}

func (m *MockSmsHandlerService) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Application), args.Error(1)
}

func (m *MockSmsHandlerService) RecordInboundSMS(ctx context.Context, accountSid, to, from, body, gwSid string) (*domain.SMSMessage, error) {
	args := m.Called(ctx, accountSid, to, from, body, gwSid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.SMSMessage), args.Error(1)
}

// Add other methods from CallServicerForESL if SMSHandler starts using them (e.g., for XML reply via <Sms>)
// For now, these are the primary ones.
// To make this a complete mock for CallServicerForESL, all methods would be needed.
// For this test, we only need what SMSHandler directly calls.
func (m *MockSmsHandlerService) UpdateCallStatus(ctx domain.MinimalCallContext, status string, hangupCause string) error { return nil }
func (m *MockSmsHandlerService) CreateRecording(ctx domain.MinimalCallContext, callSid *string, recordingSid, filePath string, duration uint32, format string, sizeBytes int64) error { return nil }
func (m *MockSmsHandlerService) GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error) { return nil, nil }
func (m *MockSmsHandlerService) GetConferenceByName(ctx context.Context, accountSid, name string) (*domain.Conference, error) { return nil, nil }
func (m *MockSmsHandlerService) CreateConference(ctx context.Context, accountSid, name, sid string) (*domain.Conference, error) { return nil, nil }
func (m *MockSmsHandlerService) GetOrCreateConference(ctx context.Context, accountSid, name string) (*domain.Conference, error) { return nil, nil }
func (m *MockSmsHandlerService) UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error { return nil }
func (m *MockSmsHandlerService) EndConference(ctx context.Context, sid string, endTime time.Time) error { return nil }
func (m *MockSmsHandlerService) AddParticipant(ctx context.Context, confSid, callSid, pSid, accountSid string, isMuted, isModerator bool) (*domain.ConferenceParticipant, error) { return nil, nil }
func (m *MockSmsHandlerService) GetParticipant(ctx context.Context, pSid string) (*domain.ConferenceParticipant, error) { return nil, nil }
func (m *MockSmsHandlerService) GetParticipantByCallSID(ctx context.Context, callSid string) (*domain.ConferenceParticipant, error) { return nil, nil }
func (m *MockSmsHandlerService) UpdateParticipantMuteStatus(ctx context.Context, pSid string, isMuted bool) error { return nil }
func (m *MockSmsHandlerService) UpdateParticipantModeratorStatus(ctx context.Context, pSid string, isModerator bool) error { return nil }
func (m *MockSmsHandlerService) RemoveParticipant(ctx context.Context, pSid string, leaveTime time.Time) error { return nil }
func (m *MockSmsHandlerService) ListParticipants(ctx context.Context, confSid string) ([]*domain.ConferenceParticipant, error) { return nil, nil }
func (m *MockSmsHandlerService) SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error) { return nil, nil }
func (m *MockSmsHandlerService) GetSMSBySID(ctx context.Context, sid string) (*domain.SMSMessage, error) { return nil, nil }
func (m *MockSmsHandlerService) UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error { return nil }


func setupSMSHandlerTest(t *testing.T) (*api.SMSHandler, *MockSmsHandlerService, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()

	logger := logrus.New(); logger.SetOutput(io.Discard)
	mockService := new(MockSmsHandlerService)

	// XMLProcessor might not be used if not testing SmsURL XML fetching part.
	// For now, provide a dummy one.
	dummyXMLProcessor := callcontrol.NewXMLProcessor(nil)

	handler := api.NewSMSHandler(mockService, dummyXMLProcessor, logger)
	return handler, mockService, w
}

func TestInboundSMSEntrypoint_Success_NoSmsURL(t *testing.T) {
	handler, mockService, w := setupSMSHandlerTest(t)

	mockApp := &domain.Application{SID: "AP123", AccountSID: "AC123", SmsURL: ""} // No SmsURL
	mockSMS := &domain.SMSMessage{SID: "SMnew"}

	mockService.On("GetApplicationByIncomingDID", mock.Anything, "1555000TO").Return(mockApp, nil).Once()
	mockService.On("RecordInboundSMS", mock.Anything, "AC123", "1555000TO", "1555000FROM", "Hello", "GatewaySID1").Return(mockSMS, nil).Once()

	formData := url.Values{}
	formData.Set("From", "1555000FROM")
	formData.Set("To", "1555000TO")
	formData.Set("Body", "Hello")
	formData.Set("MessageSid", "GatewaySID1")

	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New() // Use a new router for each test or a shared one with c.reset()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "<Response></Response>", w.Body.String())
	mockService.AssertExpectations(t)
}

func TestInboundSMSEntrypoint_Success_WithSmsURL_NoXMLProcessing(t *testing.T) {
	handler, mockService, w := setupSMSHandlerTest(t)

	mockApp := &domain.Application{SID: "AP456", AccountSID: "AC456", SmsURL: "http://example.com/sms", SmsMethod: "POST"}
	mockSMS := &domain.SMSMessage{SID: "SMnew2"}

	mockService.On("GetApplicationByIncomingDID", mock.Anything, "1555001TO").Return(mockApp, nil).Once()
	mockService.On("RecordInboundSMS", mock.Anything, "AC456", "1555001TO", "1555001FROM", "With URL", "GatewaySID2").Return(mockSMS, nil).Once()

	formData := url.Values{}
	formData.Set("From", "1555001FROM")
	formData.Set("To", "1555001TO")
	formData.Set("Body", "With URL")
	formData.Set("MessageSid", "GatewaySID2")

	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "<Response></Response>", w.Body.String()) // Still empty as XML processing is TODO
	mockService.AssertExpectations(t)
}

func TestInboundSMSEntrypoint_MissingParams(t *testing.T) {
	handler, _, w := setupSMSHandlerTest(t) // mockService not strictly needed here as it shouldn't be called

	formData := url.Values{}
	// formData.Set("From", "1555000FROM") // From is missing
	formData.Set("To", "1555000TO")
	formData.Set("Body", "Hello")

	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing required parameters")
}

func TestInboundSMSEntrypoint_AppNotFound(t *testing.T) {
	handler, mockService, w := setupSMSHandlerTest(t)

	mockService.On("GetApplicationByIncomingDID", mock.Anything, "1555UNKNOWN").Return(nil, domain.ErrNotFound).Once()

	formData := url.Values{"From":{"valid"}, "To":{"1555UNKNOWN"}, "Body":{"valid"}}
	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // Returns 200 OK to gateway as per handler logic
	assert.Contains(t, w.Body.String(), "No application configured")
	mockService.AssertExpectations(t)
}

func TestInboundSMSEntrypoint_RecordSMSError(t *testing.T) {
	handler, mockService, w := setupSMSHandlerTest(t)

	mockApp := &domain.Application{SID: "AP789", AccountSID: "AC789", SmsURL: ""}
	mockService.On("GetApplicationByIncomingDID", mock.Anything, "1555ERRORTO").Return(mockApp, nil).Once()
	mockService.On("RecordInboundSMS", mock.Anything, "AC789", "1555ERRORTO", "1555ERRORFROM", "Error Body", "GatewaySIDError").Return(nil, errors.New("DB error recording SMS")).Once()

	formData := url.Values{"From":{"1555ERRORFROM"}, "To":{"1555ERRORTO"}, "Body":{"Error Body"}, "MessageSid":{"GatewaySIDError"}}
	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to record SMS")
	mockService.AssertExpectations(t)
}

func TestInboundSMSEntrypoint_GetAppError(t *testing.T) {
	handler, mockService, w := setupSMSHandlerTest(t)

	mockService.On("GetApplicationByIncomingDID", mock.Anything, "1555APPERROR").Return(nil, errors.New("some internal app service error")).Once()

	formData := url.Values{"From":{"valid"}, "To":{"1555APPERROR"}, "Body":{"valid"}}
	req, _ := http.NewRequest("POST", "/v1/sms/inbound", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	router := gin.New()
	router.POST("/v1/sms/inbound", handler.InboundSMSEntrypoint)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Error processing application lookup")
	mockService.AssertExpectations(t)
}

func TestMain(m *testing.M) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)
	m.Run()
}
