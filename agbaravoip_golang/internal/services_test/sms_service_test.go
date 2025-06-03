package services_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"regexp" // For sqlmock
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/services"
	// "github.com/user/agbaravoip_golang/internal/utils" // For GenerateSID, not directly used in service tests unless service calls it
)

// MockSMSGatewayClient (from sms_interfaces.go, or define here for test scope)
type MockSmsGatewayClient struct {
	mock.Mock
}

func (m *MockSmsGatewayClient) SendSMS(ctx context.Context, to, from, body string, statusCallbackURL string) (string, error) {
	args := m.Called(ctx, to, from, body, statusCallbackURL)
	return args.String(0), args.Error(1)
}

// setupSMSServiceTest helper
func setupSMSServiceTest(t *testing.T) (services.SMSService, sqlmock.Sqlmock, *MockSmsGatewayClient) {
	db, mockDB, err := sqlmock.New()
	assert.NoError(t, err)
	sqlxDB := sqlx.NewDb(db, "sqlmock")

	logger := logrus.New()
	logger.SetOutput(io.Discard)

	mockGateway := new(MockSmsGatewayClient)

	// NewSMSService returns the SMSService interface, so no cast needed here.
	smsSvc := services.NewSMSService(sqlxDB, logger, mockGateway)

	return smsSvc, mockDB, mockGateway
}


// --- SMSService Tests ---

func TestSMSService_SendSMS_Success(t *testing.T) {
	smsSvc, mockDB, mockGateway := setupSMSServiceTest(t)

	accSID := "AC123"
	toNum := "15550001"
	fromNum := "15550002"
	body := "Hello from Agbara"
	msgSID := "SMtest1" // Pre-generated SID
	actionURL := "http://example.com/sms_status"

	// 1. Expect DB INSERT for new SMS (status Queued)
	// Query uses named exec, sqlmock needs positional ($1, $2) or regexp.
	// The query in smsService: INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, created_at, updated_at)
	// VALUES (:sid, :account_sid, :msg_to, :msg_from, :body, :status, :direction, :created_at, :updated_at)
	// RETURNING id, created_at, updated_at
	// For sqlmock with NamedExec, it's tricky. Simpler to match query string if PrepareNamed is used.
	// The smsService uses PrepareNamed.

	// Since PrepareNamed is used, we expect a prepare, then a query.
	mockDB.ExpectPrepare(regexp.QuoteMeta("INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id, created_at, updated_at")).
		ExpectQuery().
		WithArgs(msgSID, accSID, toNum, fromNum, body, domain.SMSStatusQueued, domain.SMSDirectionOutbound, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), time.Now()))

	// 2. Expect Gateway SendSMS call
	gatewaySID := "gw_sid_123"
	mockGateway.On("SendSMS", mock.Anything, toNum, fromNum, body, actionURL).Return(gatewaySID, nil).Once()

	// 3. Expect DB UPDATE for status Sent
	// UPDATE sms_messages SET status = :status, updated_at = :updated_at, gateway_message_sid = :gateway_sid, sent_at = :event_time WHERE sid = :sid
	// Again, for NamedExec, matching can be tricky.
	mockDB.ExpectExec(regexp.QuoteMeta("UPDATE sms_messages SET status = $1, updated_at = $2, gateway_message_sid = $3, sent_at = $4 WHERE sid = $5")).
		WithArgs(domain.SMSStatusSent, sqlmock.AnyArg(), gatewaySID, sqlmock.AnyArg(), msgSID).
		WillReturnResult(sqlmock.NewResult(1, 1))


	smsMsg, err := smsSvc.SendSMS(context.Background(), accSID, toNum, fromNum, body, msgSID, actionURL, "POST")

	assert.NoError(t, err)
	assert.NotNil(t, smsMsg)
	assert.Equal(t, msgSID, smsMsg.SID)
	assert.Equal(t, domain.SMSStatusSent, smsMsg.Status)
	assert.Equal(t, gatewaySID, smsMsg.GatewayMessageSID.String)
	assert.True(t, smsMsg.SentAt.Valid)

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockGateway.AssertExpectations(t)
}


func TestSMSService_SendSMS_DBInsertError(t *testing.T) {
	smsSvc, mockDB, mockGateway := setupSMSServiceTest(t)
	dbErr := errors.New("db insert failed")

	// Expect DB INSERT to fail
	mockDB.ExpectPrepare(regexp.QuoteMeta("INSERT INTO sms_messages")).WillReturnError(dbErr)
		// Or .ExpectQuery().WillReturnError(dbErr) if prepare succeeds but query fails

	smsMsg, err := smsSvc.SendSMS(context.Background(), "AC123", "t", "f", "b", "SMfail1", "", "POST")

	assert.Error(t, err)
	assert.Nil(t, smsMsg)
	assert.Contains(t, err.Error(), "SendSMS: preparing named query") // Or "executing insert"
	assert.True(t, errors.Is(err, dbErr) || strings.Contains(err.Error(), dbErr.Error()))

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockGateway.AssertNotCalled(t, "SendSMS", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestSMSService_SendSMS_GatewayError(t *testing.T) {
	smsSvc, mockDB, mockGateway := setupSMSServiceTest(t)
	gatewayErr := errors.New("gateway rejected")
	msgSID := "SMgwFail"

	// 1. DB INSERT success
	mockDB.ExpectPrepare(regexp.QuoteMeta("INSERT INTO sms_messages")).
		ExpectQuery().
		WithArgs(msgSID, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), time.Now()))

	// 2. Gateway SendSMS fails
	mockGateway.On("SendSMS", mock.Anything, "to", "from", "body", "").Return("", gatewayErr).Once()

	// 3. Expect DB UPDATE for status Failed
	// UPDATE sms_messages SET status = :status, updated_at = :updated_at, gateway_message_sid = :gateway_sid, error_code = :error_code, error_message = :error_message, sent_at = :event_time WHERE sid = :sid
	// Note: gatewaySID might be empty if SendSMS returns it as "" on error
	// ErrorCode and ErrorMessage are nil in this specific call to UpdateSMSStatus from SendSMS on gateway error
	mockDB.ExpectExec(regexp.QuoteMeta("UPDATE sms_messages SET status = $1, updated_at = $2, gateway_message_sid = $3, error_message = $4 WHERE sid = $5")).
		WithArgs(domain.SMSStatusFailed, sqlmock.AnyArg(), "", gatewayErr.Error(), msgSID).
		WillReturnResult(sqlmock.NewResult(1,1))


	smsMsg, err := smsSvc.SendSMS(context.Background(), "AC123", "to", "from", "body", msgSID, "", "POST")

	assert.Error(t, err)
	assert.NotNil(t, smsMsg) // smsMsg struct is returned even on gateway error, but its status is Failed
	assert.Equal(t, msgSID, smsMsg.SID)
	assert.Equal(t, domain.SMSStatusFailed, smsMsg.Status)
	assert.True(t, strings.Contains(smsMsg.ErrorMessage.String, gatewayErr.Error()))
	assert.Contains(t, err.Error(), "gateway SendSMS failed")

	assert.NoError(t, mockDB.ExpectationsWereMet())
	mockGateway.AssertExpectations(t)
}

// --- GetSMSBySID Tests ---
func TestSMSService_GetSMSBySID_Success(t *testing.T) {
	smsSvc, mockDB, _ := setupSMSServiceTest(t)
	sid := "SMget123"
	expectedSMS := domain.SMSMessage{ID: 1, SID: sid, AccountSID: "AC1", To:"t", From:"f", Body:"b", Status:domain.SMSStatusSent}

	rows := sqlmock.NewRows([]string{"id", "sid", "account_sid", "msg_to", "msg_from", "body", "status"}).
		AddRow(expectedSMS.ID, expectedSMS.SID, expectedSMS.AccountSID, expectedSMS.To, expectedSMS.From, expectedSMS.Body, expectedSMS.Status)
	mockDB.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "sms_messages" WHERE sid = $1`)).WithArgs(sid).WillReturnRows(rows)

	sms, err := smsSvc.GetSMSBySID(context.Background(), sid)
	assert.NoError(t, err)
	assert.NotNil(t, sms)
	assert.Equal(t, sid, sms.SID)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestSMSService_GetSMSBySID_NotFound(t *testing.T) {
	smsSvc, mockDB, _ := setupSMSServiceTest(t)
	sid := "SMnotfound"
	mockDB.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "sms_messages" WHERE sid = $1`)).WithArgs(sid).WillReturnError(sql.ErrNoRows)

	sms, err := smsSvc.GetSMSBySID(context.Background(), sid)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
	assert.Nil(t, sms)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- UpdateSMSStatus Tests ---
func TestSMSService_UpdateSMSStatus_Success(t *testing.T) {
	smsSvc, mockDB, _ := setupSMSServiceTest(t)
	agbaraSid := "SMupdate123"
	gatewaySid := "GWsid"
	status := domain.SMSStatusDelivered
	var errCode int32 = 0
	errMsg := "Delivered OK"
	eventTime := time.Now().UTC()

	// UPDATE sms_messages SET status = :status, updated_at = :updated_at, gateway_message_sid = :gateway_sid,
	// error_code = :error_code, error_message = :error_message, delivered_at = :event_time WHERE sid = :sid
	// Using Exec because NamedExec with map and dynamic query building is harder to mock precisely with current sqlmock
	mockDB.ExpectExec(regexp.QuoteMeta("UPDATE sms_messages SET status = $1, updated_at = $2, gateway_message_sid = $3, error_code = $4, error_message = $5, delivered_at = $6 WHERE sid = $7")).
		WithArgs(status, sqlmock.AnyArg(), gatewaySid, errCode, errMsg, eventTime, agbaraSid).
		WillReturnResult(sqlmock.NewResult(1,1))

	err := smsSvc.UpdateSMSStatus(context.Background(), agbaraSid, &gatewaySid, status, &errCode, &errMsg, &eventTime)
	assert.NoError(t, err)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

// --- RecordInboundSMS Tests ---
func TestSMSService_RecordInboundSMS_Success(t *testing.T) {
	smsSvc, mockDB, _ := setupSMSServiceTest(t)
	accSID, to, from, body, gwSID := "ACinbound", "1555TO", "1555FROM", "Inbound msg", "GW_IN_SID"

	// INSERT ... RETURNING id, created_at, updated_at, delivered_at
	mockDB.ExpectPrepare(regexp.QuoteMeta("INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, gateway_message_sid, created_at, updated_at, delivered_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id, created_at, updated_at, delivered_at")).
		ExpectQuery().
		WithArgs(sqlmock.AnyArg(), accSID, to, from, body, domain.SMSStatusReceived, domain.SMSDirectionInbound, gwSID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "delivered_at"}).AddRow(1, time.Now(), time.Now(), time.Now()))

	sms, err := smsSvc.RecordInboundSMS(context.Background(), accSID, to, from, body, gwSID)
	assert.NoError(t, err)
	assert.NotNil(t, sms)
	assert.True(t, strings.HasPrefix(sms.SID, "SM"))
	assert.Equal(t, domain.SMSStatusReceived, sms.Status)
	assert.Equal(t, domain.SMSDirectionInbound, sms.Direction)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestMain(m *testing.M) {
	m.Run()
}
