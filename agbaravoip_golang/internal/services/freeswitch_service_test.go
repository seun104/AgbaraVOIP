package services

import (
	"errors"
	"io" // Added for io.Discard
	"regexp"
	"testing"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func TestNewFreeswitchServerService(t *testing.T) {
	db, _ := setupMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)

	service := NewFreeswitchServerService(db, logger)
	assert.NotNil(t, service)
}

func TestCreateFreeswitchServer_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewFreeswitchServerService(db, logger)

	host := "fs.example.com"
	port := 8021
	password := "strongpassword"
	outboundAddress := "1.2.3.4:5080"
	isActive := true

	// Expected for Create
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(
		`INSERT INTO "freeswitch_servers" ("sid","host","port","password","outbound_address","is_active","created_at","updated_at","id") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING "id"`)).
		WithArgs(sqlmock.AnyArg(), host, port, sqlmock.AnyArg(), outboundAddress, isActive, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()). // SID, password, created_at, updated_at are dynamic
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	fsServer, err := service.CreateFreeswitchServer(host, port, password, outboundAddress, &isActive)

	assert.NoError(t, err)
	assert.NotNil(t, fsServer)
	assert.Equal(t, host, fsServer.Host)
	// The service clears the password field before returning the domain object for security reasons.
	// We cannot directly check the bcrypt hash against the original password from the returned object.
	// The fact that CreateFreeswitchServer returns without error and a non-nil fsServer implies hashing was successful internally.
	assert.Equal(t, "", fsServer.Password, "Password should be cleared before returning")


	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateFreeswitchServer_ValidationFailure(t *testing.T) {
	db, _ := setupMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewFreeswitchServerService(db, logger)

	_, err := service.CreateFreeswitchServer("", 0, "", "", nil) // Invalid inputs
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFreeswitchServerValidation) || bcrypt.ErrPasswordTooShort.Error() == err.Error() || err.Error() == "failed to hash password: password is empty", "Error should be ErrFreeswitchServerValidation or a bcrypt error for empty password")
}


func TestGetFreeswitchServerBySID_Success(t *testing.T) {
	db, mock := setupMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewFreeswitchServerService(db, logger)

	expectedSID := "FS123"
	now := time.Now()

	// Note: The actual FreeswitchServer struct in domain.go might have gorm.Model embedded or direct fields.
	// Adjust column names if they differ due to gorm conventions (e.g., "deleted_at" if using gorm.Model with soft delete).
	// For this test, assuming direct field names as per the service implementation's interaction with domain.FreeswitchServer.
	rows := sqlmock.NewRows([]string{"id", "sid", "host", "port", "password", "outbound_address", "is_active", "created_at", "updated_at"}).
		AddRow(1, expectedSID, "fs.example.com", 8021, "hashedpass", "1.2.3.4:5080", true, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "freeswitch_servers" WHERE sid = $1 ORDER BY "freeswitch_servers"."id" LIMIT 1`)).
		WithArgs(expectedSID).
		WillReturnRows(rows)

	fsServer, err := service.GetFreeswitchServerBySID(expectedSID)
	assert.NoError(t, err)
	assert.NotNil(t, fsServer)
	assert.Equal(t, expectedSID, fsServer.SID)
    assert.Equal(t, "", fsServer.Password, "Password should be cleared")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetFreeswitchServerBySID_NotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	service := NewFreeswitchServerService(db, logger)

	nonExistentSID := "FS_NONEXISTENT"

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT * FROM "freeswitch_servers" WHERE sid = $1 ORDER BY "freeswitch_servers"."id" LIMIT 1`)).
		WithArgs(nonExistentSID).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := service.GetFreeswitchServerBySID(nonExistentSID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFreeswitchServerNotFound))

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TODO: Add more tests for List, Update, Delete, and edge cases for Create/Get.
// For Update, especially test password change vs. no password change.
