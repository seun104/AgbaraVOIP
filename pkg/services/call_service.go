package services

import (
	"agbara-go/pkg/freeswitch" // New import
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log"     // Added for logging
	"strconv"
	"strings" // For dial string construction
	"time"

	"github.com/google/uuid"
)

// CallService interface (ensure it matches what API handlers expect)
// No changes needed to the interface itself for this step if CreateCall still takes models.Call.

// Update PostgresCallService struct
type PostgresCallService struct {
	db      *sql.DB
	eslConn *freeswitch.ESLConnection // Added ESL connection
}

// Update NewPostgresCallService constructor
func NewPostgresCallService(db *sql.DB, eslConn *freeswitch.ESLConnection) *PostgresCallService {
	return &PostgresCallService{db: db, eslConn: eslConn}
}

// scanCall (from existing call_service.go, ensure it includes FreeswitchCallID)
func scanCall(scanner interface{ Scan(...interface{}) error }) (*models.Call, error) {
	call := &models.Call{}
	var timeoutSeconds sql.NullInt64
	var answeredBy sql.NullString
	var endTime sql.NullTime
	var fsCallID sql.NullString // For freeswitch_call_id

	// Ensure the order of scan matches the SELECT query columns
	err := scanner.Scan(
		&call.Sid, &call.AccountSid, &call.CallerId, &call.CallTo, &call.AnswerUrl, &call.Status,
		&timeoutSeconds, &call.Direction, &call.Duration, &call.Price,
		&call.StartTime, &endTime, &call.DateCreated, &call.DateUpdated, &answeredBy,
		&fsCallID, // Scan the new field
	)
	if err != nil {
		return nil, err
	}
	if timeoutSeconds.Valid {
		call.Timeout = fmt.Sprintf("%d", timeoutSeconds.Int64)
	} else {
		call.Timeout = ""
	}
	call.AnsweredBy = answeredBy.String
	if endTime.Valid {
		call.EndTime = endTime.Time
	} else {
		call.EndTime = time.Time{}
	} // Set to zero if NULL
	call.FreeswitchCallID = fsCallID.String
	return call, nil
}

// CreateCall now also originates the call via FreeSWITCH.
func (s *PostgresCallService) CreateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.Sid = "CA" + uuid.NewString()
	// Initial status before attempting FreeSWITCH origination
	call.Status = models.CallStatusQueued // Or a new "initiating" status
	now := time.Now().UTC()
	call.DateCreated = now
	call.DateUpdated = now
	if call.StartTime.IsZero() {
		call.StartTime = now
	}
	// EndTime remains zero until call ends. sql.NullTime handles DB NULL.

	var timeoutSeconds sql.NullInt64
	if call.Timeout != "" {
		if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil {
			timeoutSeconds.Int64 = parsedTimeout
			timeoutSeconds.Valid = true
		}
		// else: invalid timeout string, will be inserted as NULL
	}

	// 1. Insert initial call record into the database
	insertQuery := `
		INSERT INTO calls (
			sid, account_sid, caller_id, call_to, answer_url, status, 
			direction, duration_seconds, price, start_time, end_time, 
			date_created, date_updated, answered_by, timeout_seconds, freeswitch_call_id 
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, NULL)
		RETURNING date_created, date_updated, start_time; 
	`
	err := s.db.QueryRowContext(ctx, insertQuery,
		call.Sid, call.AccountSid, call.CallerId, call.CallTo, call.AnswerUrl, call.Status,
		call.Direction, call.Duration, call.Price, call.StartTime, sql.NullTime{Time: call.EndTime, Valid: !call.EndTime.IsZero()},
		call.DateCreated, call.DateUpdated, sql.NullString{String: call.AnsweredBy, Valid: call.AnsweredBy != ""},
		timeoutSeconds,
	).Scan(&call.DateCreated, &call.DateUpdated, &call.StartTime)

	if err != nil {
		return nil, fmt.Errorf("failed to insert initial call record: %w", err)
	}

	// 2. Originate call via FreeSWITCH
	if s.eslConn == nil {
		log.Printf("WARN: ESLConnection is nil for call %s. Skipping FreeSWITCH origination.", call.Sid)
		call.Status = models.CallStatusFailed
		// Update the DB record with this failure status
		updatedCallWithFailure, updateErr := s.UpdateCall(ctx, call) // UpdateCall updates status and other fields
		if updateErr != nil {
			// Log the error of updating the call status, but return the original error about ESL connection
			log.Printf("ERROR: Failed to update call %s status after ESL connection error: %v", call.Sid, updateErr)
			return call, fmt.Errorf("ESL connection not available, FreeSWITCH origination skipped for call %s (DB update for failure also failed: %w)", call.Sid, updateErr)
		}
		return updatedCallWithFailure, fmt.Errorf("ESL connection not available, FreeSWITCH origination skipped for call %s", call.Sid)
	}

	callerIDName := call.CallerId
	callerIDNumber := call.CallerId
	if strings.Contains(call.CallerId, "<") && strings.HasSuffix(call.CallerId, ">") {
		parts := strings.SplitN(call.CallerId, "<", 2)
		callerIDName = strings.TrimSpace(parts[0])
		callerIDNumber = strings.TrimSuffix(strings.TrimSpace(parts[1]), ">")
	}

	originateVars := fmt.Sprintf("{agbara_call_sid=%s,agbara_account_sid=%s,origination_caller_id_name='%s',origination_caller_id_number='%s'}",
		call.Sid, call.AccountSid, callerIDName, callerIDNumber)

	// Assuming call.To is the destination and call.AnswerUrl is the application string
	// This needs to be configured properly in FreeSWITCH (e.g., dialplan, gateway)
	// Example: "sofia/gateway/my_gateway/" + call.To
	// For now, using a generic format that implies call.To is a full target string for originate
	dialString := fmt.Sprintf("%s%s %s", originateVars, call.To, call.AnswerUrl)

	log.Printf("Attempting to originate call %s with dial string: %s\n", call.Sid, dialString)

	fsCallID, err := s.eslConn.SendBgApiCommand("originate", dialString)
	if err != nil {
		log.Printf("ERROR: FreeSWITCH origination failed for call %s: %v\n", call.Sid, err)
		call.Status = models.CallStatusFailed
		updatedCallWithFailure, updateErr := s.UpdateCall(ctx, call)
		if updateErr != nil {
			log.Printf("ERROR: Failed to update call %s status after origination failure: %v\n", call.Sid, updateErr)
		}
		// Return the call object with the "failed" status, even if DB update failed, along with the primary error
		return updatedCallWithFailure, fmt.Errorf("FreeSWITCH origination command failed for call %s: %w", call.Sid, err)
	}

	log.Printf("FreeSWITCH origination successful for call %s. Job UUID: %s\n", call.Sid, fsCallID)
	call.FreeswitchCallID = fsCallID
	call.Status = models.CallStatusRinging // Or another appropriate status post-originate

	// Update the call record in DB with FreeswitchCallID and new status
	updatedCall, err := s.UpdateCall(ctx, call)
	if err != nil {
		log.Printf("ERROR: Failed to update call %s with FreeswitchCallID %s: %v\n", call.Sid, fsCallID, err)
		// Call was originated, but DB update failed. This is a critical state.
		// Return the call object with FS ID and status, but also the error.
		return call, fmt.Errorf("failed to update call record with Freeswitch ID after successful origination: %w", err)
	}

	return updatedCall, nil
}

// UpdateCall needs to handle FreeswitchCallID
func (s *PostgresCallService) UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.DateUpdated = time.Now().UTC()

	var timeoutSeconds sql.NullInt64
	if call.Timeout != "" {
		if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil {
			timeoutSeconds.Int64 = parsedTimeout
			timeoutSeconds.Valid = true
		}
	}
	var fsCallID sql.NullString
	if call.FreeswitchCallID != "" {
		fsCallID.String = call.FreeswitchCallID
		fsCallID.Valid = true
	}

	query := `
		UPDATE calls SET
			account_sid = $1, caller_id = $2, call_to = $3, answer_url = $4, status = $5,
			timeout_seconds = $6, direction = $7, duration_seconds = $8, price = $9,
			start_time = $10, end_time = $11, date_updated = $12, answered_by = $13,
            freeswitch_call_id = $14 
		WHERE sid = $15
		RETURNING sid, account_sid, caller_id, call_to, answer_url, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id;
	`
	row := s.db.QueryRowContext(ctx, query,
		call.AccountSid, call.CallerId, call.CallTo, call.AnswerUrl, call.Status,
		timeoutSeconds, call.Direction, call.Duration, call.Price,
		call.StartTime, sql.NullTime{Time: call.EndTime, Valid: !call.EndTime.IsZero()},
		call.DateUpdated, sql.NullString{String: call.AnsweredBy, Valid: call.AnsweredBy != ""},
		fsCallID, // Pass the new field
		call.Sid,
	)
	return scanCall(row)
}

// GetCall retrieves a call by its SID.
func (s *PostgresCallService) GetCall(ctx context.Context, callSid string) (*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id
		FROM calls WHERE sid = $1;
	`
	row := s.db.QueryRowContext(ctx, query, callSid)
	call, err := scanCall(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("call with SID %s not found: %w", callSid, err)
		}
		return nil, fmt.Errorf("failed to get call %s: %w", callSid, err)
	}
	return call, nil
}

// ListCalls retrieves all calls for a given AccountSid.
func (s *PostgresCallService) ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id
		FROM calls WHERE account_sid = $1 ORDER BY date_created DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil {
		return nil, fmt.Errorf("failed to query calls for account %s: %w", accountSid, err)
	}
	defer rows.Close()

	var calls []*models.Call
	for rows.Next() {
		call, err := scanCall(rows)
		if err != nil {
			// Log the error and continue? Or return immediately?
			// For now, return immediately.
			return nil, fmt.Errorf("failed to scan call row during ListCalls: %w", err)
		}
		calls = append(calls, call)
	}
	if err = rows.Err(); err != nil { // Check for errors encountered during iteration.
		return nil, fmt.Errorf("error iterating call rows for account %s: %w", accountSid, err)
	}
	if calls == nil { // Ensure empty slice is returned, not nil, if query returned no rows
		calls = []*models.Call{}
	}
	return calls, nil
}

// UpdateCallStatus updates the status of a call and its DateUpdated timestamp.
func (s *PostgresCallService) UpdateCallStatus(ctx context.Context, callSid string, status models.CallStatus) (*models.Call, error) {
	query := `
		UPDATE calls SET status = $1, date_updated = $2
		WHERE sid = $3
		RETURNING sid, account_sid, caller_id, call_to, answer_url, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id;
	`
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, query, status, now, callSid)

	updatedCall, err := scanCall(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("call with SID %s not found for status update: %w", callSid, err)
		}
		return nil, fmt.Errorf("failed to update call status for SID %s: %w", callSid, err)
	}
	return updatedCall, nil
}
