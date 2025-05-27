package services

import (
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt"
	// "strings" // Not directly used in this file's logic, but ToAppModel in models uses it.
	"time"

	"github.com/google/uuid"
)

// ApplicationService defines the interface for application management operations.
type ApplicationService interface {
	CreateApplication(ctx context.Context, accountSid string, appRequest *models.ApplicationRequest) (*models.Application, error)
	GetApplication(ctx context.Context, applicationSid string) (*models.Application, error)
	ListApplications(ctx context.Context, accountSid string) ([]*models.Application, error)
	UpdateApplication(ctx context.Context, applicationSid string, appRequest *models.ApplicationRequest) (*models.Application, error)
	DeleteApplication(ctx context.Context, applicationSid string) error
}

// PostgresApplicationService implements ApplicationService for PostgreSQL.
type PostgresApplicationService struct {
	db *sql.DB
}

// NewPostgresApplicationService creates a new PostgresApplicationService.
func NewPostgresApplicationService(db *sql.DB) *PostgresApplicationService {
	return &PostgresApplicationService{db: db}
}

// scanApplication is a helper to scan a sql.Row or sql.Rows into a models.Application struct.
func scanApplication(scanner interface{ Scan(...interface{}) error }) (*models.Application, error) {
	app := &models.Application{}
	// Nullable strings for URL and Method fields from DB
	var voiceUrl, voiceFallbackUrl, statusCallback, smsUrl, smsFallbackUrl, smsStatusCallback, heartbeatUrl sql.NullString
	var voiceMethod, voiceFallbackMethod, statusCallbackMethod, smsMethod, smsFallbackMethod, smsStatusCallbackMethod sql.NullString

	err := scanner.Scan(
		&app.Sid, &app.AccountSid, &app.FriendlyName,
		&voiceUrl, &voiceMethod, &voiceFallbackUrl, &voiceFallbackMethod,
		&statusCallback, &statusCallbackMethod,
		&smsUrl, &smsMethod, &smsFallbackUrl, &smsFallbackMethod,
		&smsStatusCallback, &smsStatusCallbackMethod,
		&heartbeatUrl,
		&app.DateCreated, &app.DateUpdated,
	)
	if err != nil {
		return nil, err
	}

	app.VoiceUrl = voiceUrl.String
	app.VoiceMethod = models.HTTPMethod(voiceMethod.String)
	app.VoiceFallbackUrl = voiceFallbackUrl.String
	app.VoiceFallbackMethod = models.HTTPMethod(voiceFallbackMethod.String)
	app.StatusCallback = statusCallback.String
	app.StatusCallbackMethod = models.HTTPMethod(statusCallbackMethod.String)
	app.SmsUrl = smsUrl.String
	app.SmsMethod = models.HTTPMethod(smsMethod.String)
	app.SmsFallbackUrl = smsFallbackUrl.String
	app.SmsFallbackMethod = models.HTTPMethod(smsFallbackMethod.String)
	app.SmsStatusCallback = smsStatusCallback.String
    app.SmsStatusCallbackMethod = models.HTTPMethod(smsStatusCallbackMethod.String)
	app.HeartbeatUrl = heartbeatUrl.String

	return app, nil
}

// CreateApplication creates a new application for an account.
func (s *PostgresApplicationService) CreateApplication(ctx context.Context, accountSid string, appRequest *models.ApplicationRequest) (*models.Application, error) {
	app := appRequest.ToAppModel() // Convert DTO to model
	app.Sid = "AP" + uuid.NewString()
	app.AccountSid = accountSid
	app.DateCreated = time.Now().UTC()
	app.DateUpdated = app.DateCreated

    // Validate HTTP methods (ToAppModel already uppercases, Validate checks if GET/POST or empty)
    // If a method is invalid (not GET/POST/empty), ToAppModel might set it to empty or keep as is.
    // The Validate method on HTTPMethod type is used here to ensure only valid ones or empty go to DB.
    if !app.VoiceMethod.Validate() { app.VoiceMethod = "" }
    if !app.VoiceFallbackMethod.Validate() { app.VoiceFallbackMethod = "" }
    if !app.StatusCallbackMethod.Validate() { app.StatusCallbackMethod = "" }
    if !app.SmsMethod.Validate() { app.SmsMethod = "" }
    if !app.SmsFallbackMethod.Validate() { app.SmsFallbackMethod = "" }
    if !app.SmsStatusCallbackMethod.Validate() { app.SmsStatusCallbackMethod = "" }

	query := `
		INSERT INTO applications (
			sid, account_sid, friendly_name,
			voice_url, voice_method, voice_fallback_url, voice_fallback_method,
			status_callback, status_callback_method,
			sms_url, sms_method, sms_fallback_url, sms_fallback_method,
			sms_status_callback, sms_status_callback_method,
			heartbeat_url, date_created, date_updated
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 
			$11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING sid, account_sid, friendly_name, voice_url, voice_method, voice_fallback_url, voice_fallback_method,
		          status_callback, status_callback_method, sms_url, sms_method, sms_fallback_url, sms_fallback_method,
		          sms_status_callback, sms_status_callback_method, heartbeat_url, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		app.Sid, app.AccountSid, app.FriendlyName,
		sql.NullString{String: app.VoiceUrl, Valid: app.VoiceUrl != ""},
		sql.NullString{String: string(app.VoiceMethod), Valid: app.VoiceMethod != ""},
		sql.NullString{String: app.VoiceFallbackUrl, Valid: app.VoiceFallbackUrl != ""},
		sql.NullString{String: string(app.VoiceFallbackMethod), Valid: app.VoiceFallbackMethod != ""},
		sql.NullString{String: app.StatusCallback, Valid: app.StatusCallback != ""},
		sql.NullString{String: string(app.StatusCallbackMethod), Valid: app.StatusCallbackMethod != ""},
		sql.NullString{String: app.SmsUrl, Valid: app.SmsUrl != ""},
		sql.NullString{String: string(app.SmsMethod), Valid: app.SmsMethod != ""},
		sql.NullString{String: app.SmsFallbackUrl, Valid: app.SmsFallbackUrl != ""},
		sql.NullString{String: string(app.SmsFallbackMethod), Valid: app.SmsFallbackMethod != ""},
		sql.NullString{String: app.SmsStatusCallback, Valid: app.SmsStatusCallback != ""},
        sql.NullString{String: string(app.SmsStatusCallbackMethod), Valid: app.SmsStatusCallbackMethod != ""},
		sql.NullString{String: app.HeartbeatUrl, Valid: app.HeartbeatUrl != ""},
		app.DateCreated, app.DateUpdated,
	)
	return scanApplication(row)
}

// GetApplication retrieves an application by its SID.
func (s *PostgresApplicationService) GetApplication(ctx context.Context, applicationSid string) (*models.Application, error) {
	query := `
		SELECT sid, account_sid, friendly_name, voice_url, voice_method, voice_fallback_url, voice_fallback_method,
		       status_callback, status_callback_method, sms_url, sms_method, sms_fallback_url, sms_fallback_method,
		       sms_status_callback, sms_status_callback_method, heartbeat_url, date_created, date_updated
		FROM applications WHERE sid = $1;
	`
	row := s.db.QueryRowContext(ctx, query, applicationSid)
	application, err := scanApplication(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("application with SID %s not found: %w", applicationSid, err)
		}
		return nil, fmt.Errorf("failed to get application: %w", err)
	}
	return application, nil
}

// ListApplications retrieves all applications for a given AccountSid.
func (s *PostgresApplicationService) ListApplications(ctx context.Context, accountSid string) ([]*models.Application, error) {
	query := `
		SELECT sid, account_sid, friendly_name, voice_url, voice_method, voice_fallback_url, voice_fallback_method,
		       status_callback, status_callback_method, sms_url, sms_method, sms_fallback_url, sms_fallback_method,
		       sms_status_callback, sms_status_callback_method, heartbeat_url, date_created, date_updated
		FROM applications WHERE account_sid = $1 ORDER BY date_created DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query applications: %w", err)
	}
	defer rows.Close()

	var applications []*models.Application
	for rows.Next() {
		application, err := scanApplication(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan application row: %w", err)
		}
		applications = append(applications, application)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating application rows: %w", err)
	}
    if applications == nil {
        applications = []*models.Application{}
    }
	return applications, nil
}

// UpdateApplication updates an existing application.
func (s *PostgresApplicationService) UpdateApplication(ctx context.Context, applicationSid string, appRequest *models.ApplicationRequest) (*models.Application, error) {
    app := appRequest.ToAppModel() 
    app.Sid = applicationSid       
	app.DateUpdated = time.Now().UTC()

    if !app.VoiceMethod.Validate() { app.VoiceMethod = "" } 
    if !app.VoiceFallbackMethod.Validate() { app.VoiceFallbackMethod = "" }
    if !app.StatusCallbackMethod.Validate() { app.StatusCallbackMethod = "" }
    if !app.SmsMethod.Validate() { app.SmsMethod = "" }
    if !app.SmsFallbackMethod.Validate() { app.SmsFallbackMethod = "" }
    if !app.SmsStatusCallbackMethod.Validate() { app.SmsStatusCallbackMethod = "" }

	query := `
		UPDATE applications SET
			friendly_name = $1,
			voice_url = $2, voice_method = $3, voice_fallback_url = $4, voice_fallback_method = $5,
			status_callback = $6, status_callback_method = $7,
			sms_url = $8, sms_method = $9, sms_fallback_url = $10, sms_fallback_method = $11,
			sms_status_callback = $12, sms_status_callback_method = $13,
			heartbeat_url = $14, date_updated = $15
		WHERE sid = $16
		RETURNING sid, account_sid, friendly_name, voice_url, voice_method, voice_fallback_url, voice_fallback_method,
		          status_callback, status_callback_method, sms_url, sms_method, sms_fallback_url, sms_fallback_method,
		          sms_status_callback, sms_status_callback_method, heartbeat_url, date_created, date_updated;
	`
	row := s.db.QueryRowContext(ctx, query,
		app.FriendlyName,
		sql.NullString{String: app.VoiceUrl, Valid: app.VoiceUrl != ""},
		sql.NullString{String: string(app.VoiceMethod), Valid: app.VoiceMethod != ""},
		sql.NullString{String: app.VoiceFallbackUrl, Valid: app.VoiceFallbackUrl != ""},
		sql.NullString{String: string(app.VoiceFallbackMethod), Valid: app.VoiceFallbackMethod != ""},
		sql.NullString{String: app.StatusCallback, Valid: app.StatusCallback != ""},
		sql.NullString{String: string(app.StatusCallbackMethod), Valid: app.StatusCallbackMethod != ""},
		sql.NullString{String: app.SmsUrl, Valid: app.SmsUrl != ""},
		sql.NullString{String: string(app.SmsMethod), Valid: app.SmsMethod != ""},
		sql.NullString{String: app.SmsFallbackUrl, Valid: app.SmsFallbackUrl != ""},
		sql.NullString{String: string(app.SmsFallbackMethod), Valid: app.SmsFallbackMethod != ""},
		sql.NullString{String: app.SmsStatusCallback, Valid: app.SmsStatusCallback != ""},
        sql.NullString{String: string(app.SmsStatusCallbackMethod), Valid: app.SmsStatusCallbackMethod != ""},
		sql.NullString{String: app.HeartbeatUrl, Valid: app.HeartbeatUrl != ""},
		app.DateUpdated,
		app.Sid,
	)
	
    updatedApp, err := scanApplication(row)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("application with SID %s not found for update: %w", applicationSid, err)
        }
        return nil, fmt.Errorf("failed to update application: %w", err)
    }
    return updatedApp, nil
}

// DeleteApplication deletes an application by its SID.
func (s *PostgresApplicationService) DeleteApplication(ctx context.Context, applicationSid string) error {
	query := `DELETE FROM applications WHERE sid = $1;`
	result, err := s.db.ExecContext(ctx, query, applicationSid)
	if err != nil {
		return fmt.Errorf("failed to delete application %s: %w", applicationSid, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for application deletion: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("application %s not found for deletion (no rows affected)", applicationSid)
	}
	return nil
}
