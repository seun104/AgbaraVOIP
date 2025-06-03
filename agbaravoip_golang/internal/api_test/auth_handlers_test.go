package api_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/api"       // Package containing the handler
	"github.com/user/agbaravoip_golang/internal/auth"      // For JWT utils & Init
	"github.com/user/agbaravoip_golang/internal/domain"
)

// Mock for CallServicerForESL focusing on methods used by AuthHandler
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) ValidateCredentials(ctx context.Context, accountSid string, plainToken string) (*domain.Account, error) {
	args := m.Called(ctx, accountSid, plainToken)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Account), args.Error(1)
}

func (m *MockAuthService) GetAccountBySID(ctx context.Context, sid string) (*domain.Account, error) {
	args := m.Called(ctx, sid)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Account), args.Error(1)
}
// Add stubs for other CallServicerForESL methods to satisfy the interface fully
func (m *MockAuthService) UpdateCallStatus(ctx domain.MinimalCallContext, status string, hangupCause string) error { return nil }
func (m *MockAuthService) CreateRecording(ctx domain.MinimalCallContext, callSid *string, rSID, fp string, dur uint32, fS string, sB int64) error { return nil }
func (m *MockAuthService) GetConferenceBySID(ctx context.Context, sid string) (*domain.Conference, error) { return nil, nil }
func (m *MockAuthService) GetConferenceByName(ctx context.Context, accSID, name string) (*domain.Conference, error) { return nil, nil }
func (m *MockAuthService) CreateConference(ctx context.Context, accSID, name, sid string) (*domain.Conference, error) { return nil, nil }
func (m *MockAuthService) GetOrCreateConference(ctx context.Context, accSID, name string) (*domain.Conference, error) { return nil, nil }
func (m *MockAuthService) UpdateConferenceStatus(ctx context.Context, sid string, status domain.ConferenceStatus) error { return nil }
func (m *MockAuthService) EndConference(ctx context.Context, sid string, endTime time.Time) error { return nil }
func (m *MockAuthService) AddParticipant(ctx context.Context, confSID, callSID, pSID, accSID string,isMuted,isMod bool) (*domain.ConferenceParticipant,error) { return nil,nil }
func (m *MockAuthService) GetParticipant(ctx context.Context, pSID string) (*domain.ConferenceParticipant,error) { return nil,nil }
func (m *MockAuthService) GetParticipantByCallSID(ctx context.Context, cSID string) (*domain.ConferenceParticipant,error) { return nil,nil }
func (m *MockAuthService) UpdateParticipantMuteStatus(ctx context.Context, pSID string, isMuted bool) error { return nil }
func (m *MockAuthService) UpdateParticipantModeratorStatus(ctx context.Context, pSID string, isMod bool) error { return nil }
func (m *MockAuthService) RemoveParticipant(ctx context.Context, pSID string, lTime time.Time) error { return nil }
func (m *MockAuthService) ListParticipants(ctx context.Context, confSID string) ([]*domain.ConferenceParticipant,error) { return nil,nil }
func (m *MockAuthService) SendSMS(ctx context.Context, accSID,to,from,body,mSID,aURL,aMethod string) (*domain.SMSMessage,error) { return nil,nil }
func (m *MockAuthService) GetSMSBySID(ctx context.Context, sid string) (*domain.SMSMessage,error) { return nil,nil }
func (m *MockAuthService) UpdateSMSStatus(ctx context.Context, agSID string, gwSID *string, stat domain.SMSStatus, errC *int32, errM *string, evTime *time.Time) error { return nil }
func (m *MockAuthService) RecordInboundSMS(ctx context.Context, accSID,to,from,body,inGwSID string) (*domain.SMSMessage,error) { return nil,nil }
func (m *MockAuthService) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application,error) { return nil,nil }


const testAuthHandlerSecret = "testsecrethandlerkey1234567890123"
const testAuthTokenDuration = 15 * time.Minute

func setupAuthHandlerTest(t *testing.T) (*gin.Engine, *MockAuthService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	auth.InitJWTSecret(testAuthHandlerSecret) // Initialize JWT for tests

	mockService := new(MockAuthService)
	authHandler := api.NewAuthHandler(mockService, logger, testAuthHandlerSecret, testAuthTokenDuration)

	router.POST("/auth/token", authHandler.GenerateTokenHandler)
	return router, mockService
}

func TestGenerateTokenHandler_Success_BasicAuth(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)

	accountSID := "AC123"
	authToken := "test_token"
	mockAccount := &domain.Account{SID: accountSID, HashedAuthToken: authToken, Status: domain.AccountStatusActive, FriendlyName: "Test Acc"}

	// For this test, GetAccountBySID is called by the handler.
	// The handler then does a direct comparison. For real hash, ValidateCredentials would be better.
	// The prompt used GetAccountBySID and direct compare as a placeholder.
	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(mockAccount, nil).Once()
	// If ValidateCredentials was used:
	// mockService.On("ValidateCredentials", mock.Anything, accountSID, authToken).Return(mockAccount, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accountSID+":"+authToken)))

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response["token"])
	assert.NotEmpty(t, response["expires_at"])
	assert.Equal(t, "Bearer", response["type"])

	// Verify token content (optional but good)
	claims, err := auth.ValidateToken(response["token"])
	assert.NoError(t, err)
	assert.Equal(t, accountSID, claims.AccountSID)
	assert.Equal(t, "user", claims.Role) // Default role

	mockService.AssertExpectations(t)
}

func TestGenerateTokenHandler_Success_JSONBody(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)

	accountSID := "AC456"
	authToken := "json_token"
	mockAccount := &domain.Account{SID: accountSID, HashedAuthToken: authToken, Status: domain.AccountStatusActive}

	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(mockAccount, nil).Once()

	payload := api.TokenRequestPayload{AccountSID: accountSID, AuthToken: authToken} // Use exported type if defined in api
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["token"])
	mockService.AssertExpectations(t)
}

func TestGenerateTokenHandler_InvalidCredentials_NotFound(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)
	accountSID := "ACunknown"
	authToken := "bad_token"

	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(nil, domain.ErrNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accountSID+":"+authToken)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
	mockService.AssertExpectations(t)
}

func TestGenerateTokenHandler_InvalidCredentials_WrongToken(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)
	accountSID := "AC789"
	correctAuthToken := "correct_token"
	wrongAuthToken := "wrong_token"
	// Assuming AuthToken in domain.Account is the raw token for this test, not hashed
	mockAccount := &domain.Account{SID: accountSID, HashedAuthToken: correctAuthToken, Status: domain.AccountStatusActive}


	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(mockAccount, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accountSID+":"+wrongAuthToken)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid credentials")
	mockService.AssertExpectations(t)
}

func TestGenerateTokenHandler_AccountNotActive(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)
	accountSID := "ACinactive"
	authToken := "inactive_token"
	mockAccount := &domain.Account{SID: accountSID, HashedAuthToken: authToken, Status: domain.AccountStatusSuspended}

	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(mockAccount, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accountSID+":"+authToken)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Account is not active")
	mockService.AssertExpectations(t)
}

func TestGenerateTokenHandler_MissingCredentials_JSON(t *testing.T) {
	router, _ := setupAuthHandlerTest(t) // mockService not called

	payload := map[string]string{"account_sid": "ACmissing"} // AuthToken missing
	jsonPayload, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", bytes.NewBuffer(jsonPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "AccountSID and AuthToken are required")
}

func TestGenerateTokenHandler_ServiceError_GetAccount(t *testing.T) {
	router, mockService := setupAuthHandlerTest(t)
	accountSID := "ACserviceErr"
	authToken := "anytoken"
	serviceErr := errors.New("internal service error")

	mockService.On("GetAccountBySID", mock.Anything, accountSID).Return(nil, serviceErr).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/token", nil)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(accountSID+":"+authToken)))
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Server error during authentication")
	mockService.AssertExpectations(t)
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}
