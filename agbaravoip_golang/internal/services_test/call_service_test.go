package services_test

import (
	"errors"
	"testing"
	"strings" // For mock.MatchedBy
	// "time" // Not used in this simplified version

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/esl" 
	"github.com/user/agbaravoip_golang/internal/services"
	"github.com/sirupsen/logrus"
	"context" // For sqlmock/GORM context
	"time"    // For time.Now() in tests

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/esl"
	"github.com/user/agbaravoip_golang/internal/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger" as gormlogger // Alias to avoid conflict with logrus
)

// --- MockMinimalCallContext for service tests ---
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
func (m *MockMinimalCallContext) GetUuid() string             { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAccountSid() string       { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetApplicationSid() string   { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAnswerURL() string        { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetVariable(v string) string { return m.Called(v).String(0) }
func (m *MockMinimalCallContext) IsHangupInitiated() bool     { return m.Called().Bool(0) }
func (m *MockMinimalCallContext) SetHangupInitiated()         { m.Called() }
func (m *MockMinimalCallContext) SetPendingRecording(i interface{}) { m.Called(i) }
func (m *MockMinimalCallContext) GetPendingRecording() (interface{}, bool) {
	args := m.Called()
	return args.Get(0), args.Bool(1)
}
func (m *MockMinimalCallContext) ClearPendingRecording() { m.Called() }
func (m *MockMinimalCallContext) AddPendingDial(cUuid string, i interface{}) { m.Called(cUuid, i) }
func (m *MockMinimalCallContext) GetPendingDial(cUuid string) (interface{}, bool) {
	args := m.Called(cUuid)
	return args.Get(0), args.Bool(1)
}
func (m *MockMinimalCallContext) RemovePendingDial(cUuid string)       { m.Called(cUuid) }
func (m *MockMinimalCallContext) GetAllPendingDials() map[string]interface{} { args := m.Called(); return args.Get(0).(map[string]interface{}) }
func (m *MockMinimalCallContext) SendNextElements(e []domain.CallControlElement) error { return m.Called(e).Error(0) }
func (m *MockMinimalCallContext) GetNextElementsChannel() <-chan []domain.CallControlElement {
	args := m.Called()
	return args.Get(0).(<-chan []domain.CallControlElement)
}


// MockFSInboundClient is a mock for esl.FSInboundClient
type MockFSInboundClient struct { mock.Mock }
func (m *MockFSInboundClient) SendCommand(cmd string) (string, error) { args := m.Called(cmd); return args.String(0), args.Error(1) }
func (m *MockFSInboundClient) GetFSStatus() (string, error) { args := m.Called(); return args.String(0), args.Error(1) }
func (m *MockFSInboundClient) Close() {} // Add Close method

// MockApplicationService is a mock for services.IApplicationService
type MockApplicationService struct { mock.Mock }
func (m *MockApplicationService) CreateApplication(accountSid string, app *domain.Application) (*domain.Application, error) { args := m.Called(accountSid, app); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Application), args.Error(1) }
func (m *MockApplicationService) GetApplicationBySID(accountSid string, appSid string) (*domain.Application, error) { args := m.Called(accountSid, appSid); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*domain.Application), args.Error(1) }
func (m *MockApplicationService) ListApplications(accountSid string) ([]*domain.Application, error) { args := m.Called(accountSid); if args.Get(0) == nil {return nil, args.Error(1)}; return args.Get(0).([]*domain.Application), args.Error(1) }
func (m *MockApplicationService) UpdateApplication(accountSid string, appSid string, updates map[string]interface{}) (*domain.Application, error) { args := m.Called(accountSid, appSid, updates); if args.Get(0) == nil {return nil, args.Error(1)}; return args.Get(0).(*domain.Application), args.Error(1) }
func (m *MockApplicationService) DeleteApplication(accountSid string, appSid string) error { args := m.Called(accountSid, appSid); return args.Error(0) }
func (m *MockApplicationService) GetApplicationByIncomingDID(ctx context.Context, did string) (*domain.Application, error) {
	args := m.Called(ctx, did)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*domain.Application), args.Error(1)
}

// MockFreeswitchOutboundConfigProvider is a mock for the config provider interface
type MockFreeswitchOutboundConfigProvider struct { mock.Mock }
func (m *MockFreeswitchOutboundConfigProvider) GetESLOutboundServerListenAddress() string { args := m.Called(); return args.String(0) }


func TestCallService_OriginateCall(t *testing.T) {
	logger := logrus.New(); logger.SetLevel(logrus.PanicLevel) 
	
	mockEslClient := new(MockFSInboundClient)
	mockAppService := new(MockApplicationService)
	mockConfigProvider := new(MockFreeswitchOutboundConfigProvider)
	
	// Using a nil *gorm.DB for tests not hitting the DB, or for tests where DB is mocked.
	// For DB interaction tests, a proper test DB or more complete GORM mock is needed.
	callServiceWithNilDB := services.NewCallService(nil, mockEslClient, mockAppService, mockConfigProvider, logger) 
	
	accountSid := "ACtest123"; fromNum := "1000";	toNum := "1001"; answerUrl := "http://example.com/answer"; testAppSid := "APapp123"

	t.Run("Successful call origination with direct answer_url", func(t *testing.T) {
		t.Skip("Skipping GORM/ESL dependent test that requires DB and live ESL interaction or more detailed mocks.")
		// mockDb := new(MockGormDB) // Assume this is a proper GORM mock or test DB connection
		// mockDb.On("Create", mock.AnythingOfType("*domain.Call")).Return(nil).Once().Run(func(args mock.Arguments) {
		// 	call := args.Get(0).(*domain.Call); call.SID = "CAmockedcall"
		// })
		// mockDb.On("Model", mock.AnythingOfType("*domain.Call")).Return(mockDb) 
		// mockDb.On("Updates", mock.AnythingOfType("map[string]interface{}")).Return(nil).Once()
		
		// mockEslClient.On("GetOutboundServerListenAddress").Return("127.0.0.1:8084").Once() // This is now on mockConfigProvider
		mockConfigProvider.On("GetESLOutboundServerListenAddress").Return("127.0.0.1:8084").Once()
		mockEslClient.On("SendCommand", mock.AnythingOfType("string")).Return("+OK Job-UUID: some-job-id", nil).Once()
		
		// callService := services.NewCallService(mockDb, mockEslClient, mockAppService, mockConfigProvider, logger)
		// call, err := callService.OriginateCall(accountSid, nil, fromNum, toNum, answerUrl, nil) // Corrected: appSidOrNil is second
		// assert.NoError(t, err); assert.NotNil(t, call); assert.Equal(t, domain.CallStatusInitiated, call.Status)
	})

	t.Run("ESL client not available", func(t *testing.T) {
		callServiceNoEsl := services.NewCallService(nil, nil, mockAppService, mockConfigProvider, logger) 
		_, err := callServiceNoEsl.OriginateCall(accountSid, fromNum, toNum, answerUrl, nil, nil)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, services.ErrESLClientNotAvailable_CS))
	})
}

// --- Tests for CreateRecording ---

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent), // Silence GORM logger for tests
	})
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening gorm database", err)
	}
	return gormDB, mock
}


func TestCallService_CreateRecording_Success(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	gormDB, dbMock := setupMockDB(t)
	callService := services.NewCallService(gormDB, nil, nil, nil, logger)

	mockCtx := new(MockMinimalCallContext)
	mockCtx.On("Log").Return(logEntry)
	mockCtx.On("GetAccountSid").Return("ACtestaccsid")

	callSid := "CAtestcallsid"
	recordingSid := "REtestrecsid"
	filePath := "/tmp/rec.wav"
	var duration uint32 = 60
	format := "wav"
	var sizeBytes int64 = 12345

	expectedQuery := `INSERT INTO recordings (sid, account_sid, call_sid, duration_seconds, file_path, format, size_bytes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

	dbMock.ExpectExec(regexp.QuoteMeta(expectedQuery)).
		WithArgs(recordingSid, "ACtestaccsid", &callSid, duration, filePath, format, sizeBytes, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := callService.CreateRecording(mockCtx, &callSid, recordingSid, filePath, duration, format, sizeBytes)
	assert.NoError(t, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	mockCtx.AssertExpectations(t)
}

func TestCallService_CreateRecording_NullCallSID(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	gormDB, dbMock := setupMockDB(t)
	callService := services.NewCallService(gormDB, nil, nil, nil, logger)

	mockCtx := new(MockMinimalCallContext)
	mockCtx.On("Log").Return(logEntry)
	mockCtx.On("GetAccountSid").Return("ACtestaccsid2")

	// callSid is nil for this test
	recordingSid := "REtestrecsid_nocall"
	filePath := "/tmp/rec_nocall.mp3"
	var duration uint32 = 30
	format := "mp3"
	var sizeBytes int64 = 54321

	expectedQuery := `INSERT INTO recordings (sid, account_sid, call_sid, duration_seconds, file_path, format, size_bytes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

	dbMock.ExpectExec(regexp.QuoteMeta(expectedQuery)).
		WithArgs(recordingSid, "ACtestaccsid2", nil, duration, filePath, format, sizeBytes, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := callService.CreateRecording(mockCtx, nil, recordingSid, filePath, duration, format, sizeBytes)
	assert.NoError(t, err)
	assert.NoError(t, dbMock.ExpectationsWereMet())
	mockCtx.AssertExpectations(t)
}

func TestCallService_CreateRecording_DBError(t *testing.T) {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	logEntry := logrus.NewEntry(logger)

	gormDB, dbMock := setupMockDB(t)
	callService := services.NewCallService(gormDB, nil, nil, nil, logger)

	mockCtx := new(MockMinimalCallContext)
	mockCtx.On("Log").Return(logEntry)
	mockCtx.On("GetAccountSid").Return("ACtesterror")

	callSid := "CAtesterr"
	dbError := errors.New("DB insert error")

	expectedQuery := `INSERT INTO recordings (sid, account_sid, call_sid, duration_seconds, file_path, format, size_bytes, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

	dbMock.ExpectExec(regexp.QuoteMeta(expectedQuery)).
		WithArgs(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		WillReturnError(dbError)

	err := callService.CreateRecording(mockCtx, &callSid, "REerr", "/tmp/err.wav", 10, "wav", 100)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inserting recording")
	assert.True(t, errors.Is(err, dbError))
	assert.NoError(t, dbMock.ExpectationsWereMet())
	mockCtx.AssertExpectations(t)
}

