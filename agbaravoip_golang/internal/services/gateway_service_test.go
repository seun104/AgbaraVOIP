package services

import (
	"errors"
	"io" // For io.Discard
	"regexp"
	"testing"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Re-using setupMockDB from freeswitch_service_test.go (conceptually)
// If in different packages, it would need to be a shared test utility or redefined.
// For this subtask, we assume it's available or redefine it if necessary.
func setupGatewayTestMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDb, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to open sqlmock database: %s", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:       mockDb,
		DriverName: "postgres",
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to open gorm database: %s", err)
	}
	return db, mock
}


func TestNewGatewayService(t *testing.T) {
	db, _ := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	service := NewGatewayService(db, logger)
	assert.NotNil(t, service)
}

func TestCreateGateway_Success(t *testing.T) {
	db, mock := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewGatewayService(db, logger)

	accountSID := "AC123"
	friendlyName := "Test Gateway"
	gatewayString := "sofia/gateway/testgw/"
	isEnabled := true
	var retryCount int = 1
	var timeoutSeconds int = 30


	// Expected for Create
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "gateways" ("sid","account_sid","freeswitch_server_sid","friendly_name","gateway_string","codecs","retry_count","timeout_seconds","routes","is_enabled","created_at","updated_at","id") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), accountSID, nil, friendlyName, gatewayString, sqlmock.AnyArg(), retryCount, timeoutSeconds, sqlmock.AnyArg(), isEnabled, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	// sqlmock.AnyArg() for codecs and routes as they can be nil/empty and GORM might handle them as such.
	// If they are guaranteed to be non-nil empty slices/maps, you might match `pq.Array([]string{})` or similar for codecs.
	// For JSONB (routes), matching `nil` or an empty JSON `"{}"` or `null` might be needed depending on how GORM serializes empty maps.
	// The current service logic for CreateGateway does not explicitly convert nil slices/maps to empty ones before DB call,
	// so GORM's default marshaling for nil will apply (likely NULL in DB).

	gw, err := service.CreateGateway(accountSID, nil, friendlyName, gatewayString, nil, &retryCount, &timeoutSeconds, nil, &isEnabled)

	assert.NoError(t, err)
	assert.NotNil(t, gw)
	assert.Equal(t, friendlyName, gw.FriendlyName)
	assert.Equal(t, accountSID, gw.AccountSID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateGateway_ValidationFailure(t *testing.T) {
	db, _ := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewGatewayService(db, logger)

	_, err := service.CreateGateway("", nil, "", "", nil, nil,nil, nil, nil) // Invalid inputs
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrGatewayValidation))
}

func TestGetGatewayBySID_Success(t *testing.T) {
	db, mock := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewGatewayService(db, logger)

	accountSID := "AC123"
	expectedSID := "GW123"
	now := time.Now()

	// Adjust columns based on your domain.Gateway struct and GORM's behavior, esp. for JSONB/array types.
	// For this example, assuming codecs and routes are not fetched or are simple enough for default sqlmock.Rows.
	rows := sqlmock.NewRows([]string{"id", "sid", "account_sid", "freeswitch_server_sid", "friendly_name", "gateway_string", "codecs", "retry_count", "timeout_seconds", "routes", "is_enabled", "created_at", "updated_at"}).
		AddRow(1, expectedSID, accountSID, nil, "Test GW", "sofia/gw/test", nil, 0, 30, nil, true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "gateways" WHERE account_sid = $1 AND sid = $2 ORDER BY "gateways"."id" LIMIT 1`)).
		WithArgs(accountSID, expectedSID).
		WillReturnRows(rows)

	gw, err := service.GetGatewayBySID(accountSID, expectedSID)
	assert.NoError(t, err)
	assert.NotNil(t, gw)
	assert.Equal(t, expectedSID, gw.SID)
	assert.Equal(t, accountSID, gw.AccountSID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGatewayBySID_NotFound(t *testing.T) {
	db, mock := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewGatewayService(db, logger)

	accountSID := "AC123"
	nonExistentSID := "GW_NONEXISTENT"

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "gateways" WHERE account_sid = $1 AND sid = $2 ORDER BY "gateways"."id" LIMIT 1`)).
		WithArgs(accountSID, nonExistentSID).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := service.GetGatewayBySID(accountSID, nonExistentSID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrGatewayNotFound))

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetGlobalGatewayBySID_Success(t *testing.T) {
	db, mock := setupGatewayTestMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewGatewayService(db, logger)

	expectedSID := "GWGlobal123"
	accountSID := "ACGlobalOwner"
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "sid", "account_sid", "freeswitch_server_sid", "friendly_name", "gateway_string", "codecs", "retry_count", "timeout_seconds", "routes", "is_enabled", "created_at", "updated_at"}).
		AddRow(1, expectedSID, accountSID, nil, "Global Test GW", "sofia/gw/globaltest", nil, 0, 30, nil, true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "gateways" WHERE sid = $1 ORDER BY "gateways"."id" LIMIT 1`)).
		WithArgs(expectedSID).
		WillReturnRows(rows)

	gw, err := service.GetGlobalGatewayBySID(expectedSID)
	assert.NoError(t, err)
	assert.NotNil(t, gw)
	assert.Equal(t, expectedSID, gw.SID)
	assert.Equal(t, accountSID, gw.AccountSID) // Ensure it returns the correct associated account

	assert.NoError(t, mock.ExpectationsWereMet())
}


// TODO: Add more tests for ListGateways, ListGlobalGateways, UpdateGateway, UpdateGlobalGateway,
// DeleteGateway, DeleteGlobalGateway, and edge cases.
// Test filtering in List methods.
// Test updates with various fields changed / unchanged.
// Test handling of array (Codecs) and JSONB (Routes) fields with sqlmock.
