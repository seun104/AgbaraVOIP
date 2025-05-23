package services

import (
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt" // For error wrapping
	"strconv" // Added import for strconv
	"time"

	"github.com/google/uuid"
	// "errors" // For sql.ErrNoRows comparison if needed explicitly
)

// CallService defines the interface for call operations.
type CallService interface {
	CreateCall(ctx context.Context, call *models.Call) (*models.Call, error)
	UpdateCallStatus(ctx context.Context, callSid string, status string) (*models.Call, error)
	UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error)
	GetCall(ctx context.Context, callSid string) (*models.Call, error)
	ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error)
}

// PostgresCallService implements CallService for PostgreSQL
type PostgresCallService struct {
	db *sql.DB
}

// NewPostgresCallService creates a new PostgresCallService
func NewPostgresCallService(db *sql.DB) *PostgresCallService {
	return &PostgresCallService{db: db}
}

// CreateCall adds a new call log to the PostgreSQL database
func (s *PostgresCallService) CreateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.Sid = "CA" + uuid.NewString() // Generate SID
	call.Status = "queued"            // Default status
	now := time.Now().UTC()
	call.DateCreated = now
	call.DateUpdated = now
    if call.StartTime.IsZero() {
        call.StartTime = now
    }
    // Ensure EndTime is not before StartTime if both are set or defaulted.
    // If EndTime is zero and StartTime is set, it might be okay to leave EndTime as zero
    // or set it to StartTime depending on business logic (e.g., for very short/instantaneous events).
    // The provided C# code sets EndTime = StartTime if EndTime is not provided.
    if call.EndTime.IsZero() && !call.StartTime.IsZero() {
        call.EndTime = call.StartTime 
    }


    // Convert call.Timeout string to sql.NullInt64 for DB insert
    var timeoutSeconds sql.NullInt64
    if call.Timeout != "" {
        if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil {
            timeoutSeconds.Int64 = parsedTimeout
            timeoutSeconds.Valid = true
        } else {
            // Optionally handle error if Timeout string is not a valid int
            // For now, if it's not a valid int, it will be inserted as NULL (as timeoutSeconds.Valid remains false)
            // return nil, fmt.Errorf("invalid Timeout value in CreateCall: %s: %w", call.Timeout, err)
        }
    }

	query := `
		INSERT INTO calls (
			sid, account_sid, caller_id, call_to, answer_url, status, 
			direction, duration_seconds, price, start_time, end_time, 
			date_created, date_updated, answered_by, timeout_seconds
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING date_created, date_updated, start_time, end_time; 
	`
    // Note: The RETURNING clause might need to return more fields if the DB auto-generates/modifies them
    // and you need those values back in the `call` struct immediately.
    // For instance, if start_time or end_time had defaults in the DB triggered by NULL inputs.
    // Given the current logic, DateCreated, DateUpdated, StartTime, EndTime are explicitly set before insert.
	err := s.db.QueryRowContext(ctx, query,
		call.Sid, call.AccountSid, call.CallerId, call.CallTo, call.AnswerUrl, call.Status,
		call.Direction, call.Duration, call.Price, call.StartTime, call.EndTime,
		call.DateCreated, call.DateUpdated, call.AnsweredBy, timeoutSeconds, // Added timeoutSeconds
	).Scan(&call.DateCreated, &call.DateUpdated, &call.StartTime, &call.EndTime)

	if err != nil {
		return nil, fmt.Errorf("failed to insert call: %w", err)
	}
	return call, nil
}

// GetCall retrieves a call by its SID
func (s *PostgresCallService) GetCall(ctx context.Context, callSid string) (*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by
		FROM calls WHERE sid = $1;
	`
	row := s.db.QueryRowContext(ctx, query, callSid)
	call := &models.Call{}
    var timeoutSeconds sql.NullInt64 

	err := row.Scan(
		&call.Sid, &call.AccountSid, &call.CallerId, &call.CallTo, &call.AnswerUrl, &call.Status,
		&timeoutSeconds, &call.Direction, &call.Duration, &call.Price,
		&call.StartTime, &call.EndTime, &call.DateCreated, &call.DateUpdated, &call.AnsweredBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("call with SID %s not found: %w", callSid, err)
		}
		return nil, fmt.Errorf("failed to get call: %w", err)
	}
    if timeoutSeconds.Valid {
        call.Timeout = fmt.Sprintf("%d", timeoutSeconds.Int64) 
    } else {
        call.Timeout = "" 
    }
	return call, nil
}

// ListCalls retrieves all calls for a given AccountSid
func (s *PostgresCallService) ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by
		FROM calls WHERE account_sid = $1 ORDER BY date_created DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query calls: %w", err)
	}
	defer rows.Close()

	var calls []*models.Call
	for rows.Next() {
		call := &models.Call{}
        var timeoutSeconds sql.NullInt64
		if err := rows.Scan(
			&call.Sid, &call.AccountSid, &call.CallerId, &call.CallTo, &call.AnswerUrl, &call.Status,
			&timeoutSeconds, &call.Direction, &call.Duration, &call.Price,
			&call.StartTime, &call.EndTime, &call.DateCreated, &call.DateUpdated, &call.AnsweredBy,
		); err != nil {
			// If a single row scan fails, we return error immediately.
			// Depending on requirements, one might choose to log and skip, or collect errors.
			return nil, fmt.Errorf("failed to scan call row: %w", err) 
		}
        if timeoutSeconds.Valid {
            call.Timeout = fmt.Sprintf("%d", timeoutSeconds.Int64)
        } else {
            call.Timeout = ""
        }
		calls = append(calls, call)
	}

	if err = rows.Err(); err != nil { // Check for errors encountered during iteration
		return nil, fmt.Errorf("error iterating call rows: %w", err)
	}
    if calls == nil { // Ensure empty slice is returned, not nil, if query returned no rows
        calls = []*models.Call{}
    }
	return calls, nil
}

// UpdateCallStatus updates the status of a call and its DateUpdated timestamp
func (s *PostgresCallService) UpdateCallStatus(ctx context.Context, callSid string, status string) (*models.Call, error) {
	query := `
		UPDATE calls SET status = $1, date_updated = $2
		WHERE sid = $3
		RETURNING sid, account_sid, caller_id, call_to, answer_url, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by;
	`
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, query, status, now, callSid)
	
	updatedCall := &models.Call{}
    var timeoutSeconds sql.NullInt64 // For scanning the timeout_seconds from RETURNING
	err := row.Scan(
		&updatedCall.Sid, &updatedCall.AccountSid, &updatedCall.CallerId, &updatedCall.CallTo, 
		&updatedCall.AnswerUrl, &updatedCall.Status, &timeoutSeconds, &updatedCall.Direction, 
		&updatedCall.Duration, &updatedCall.Price, &updatedCall.StartTime, &updatedCall.EndTime, 
		&updatedCall.DateCreated, &updatedCall.DateUpdated, &updatedCall.AnsweredBy,
	)
	if err != nil {
		if err == sql.ErrNoRows { // Check if the specific SID was not found for update
			return nil, fmt.Errorf("call with SID %s not found for status update: %w", callSid, err)
		}
		return nil, fmt.Errorf("failed to update call status: %w", err)
	}
    if timeoutSeconds.Valid { // Convert scanned timeout_seconds back to string
        updatedCall.Timeout = fmt.Sprintf("%d", timeoutSeconds.Int64)
    } else {
        updatedCall.Timeout = ""
    }
	return updatedCall, nil
}

// UpdateCall updates specified fields of an existing call.
func (s *PostgresCallService) UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.DateUpdated = time.Now().UTC() // Always update the DateUpdated timestamp
	
    var timeoutSeconds sql.NullInt64 // For converting string Timeout to sql.NullInt64 for DB
    if call.Timeout != "" {
        if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil {
            timeoutSeconds.Int64 = parsedTimeout
            timeoutSeconds.Valid = true
        } else {
            // Optionally return an error if Timeout is non-empty but invalid
            // return nil, fmt.Errorf("invalid Timeout value in UpdateCall: %s: %w", call.Timeout, err)
            // If not returning error, invalid string Timeout means NULL will be written for timeout_seconds
        }
    }

	query := `
		UPDATE calls SET
			account_sid = $1, caller_id = $2, call_to = $3, answer_url = $4, status = $5,
			timeout_seconds = $6, direction = $7, duration_seconds = $8, price = $9,
			start_time = $10, end_time = $11, date_updated = $12, answered_by = $13
		WHERE sid = $14
		RETURNING sid, account_sid, caller_id, call_to, answer_url, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by;
	`
	row := s.db.QueryRowContext(ctx, query,
		call.AccountSid, call.CallerId, call.CallTo, call.AnswerUrl, call.Status,
		timeoutSeconds, // Use the sql.NullInt64 version for the DB
        call.Direction, call.Duration, call.Price,
		call.StartTime, call.EndTime, call.DateUpdated, call.AnsweredBy,
		call.Sid, // This is for the WHERE clause
	)

	updatedCall := &models.Call{}
    var scannedTimeoutSeconds sql.NullInt64 // For scanning the timeout_seconds from RETURNING
	err := row.Scan(
		&updatedCall.Sid, &updatedCall.AccountSid, &updatedCall.CallerId, &updatedCall.CallTo, 
		&updatedCall.AnswerUrl, &updatedCall.Status, &scannedTimeoutSeconds, &updatedCall.Direction, 
		&updatedCall.Duration, &updatedCall.Price, &updatedCall.StartTime, &updatedCall.EndTime, 
		&updatedCall.DateCreated, &updatedCall.DateUpdated, &updatedCall.AnsweredBy,
	)

	if err != nil {
		if err == sql.ErrNoRows { // Check if the specific SID was not found for update
			return nil, fmt.Errorf("call with SID %s not found for update: %w", call.Sid, err)
		}
		return nil, fmt.Errorf("failed to update call: %w", err)
	}
    if scannedTimeoutSeconds.Valid { // Convert scanned timeout_seconds back to string for the returned model
        updatedCall.Timeout = fmt.Sprintf("%d", scannedTimeoutSeconds.Int64)
    } else {
        updatedCall.Timeout = ""
    }
	return updatedCall, nil
}
