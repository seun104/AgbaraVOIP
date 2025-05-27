package services

import (
	"agbara-go/pkg/freeswitch"
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"fmt"
	"log" 
	"os" // <-- ADDED for os.Getenv in getGatewayPrefix
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CallService interface (remains unchanged)

// Update PostgresCallService struct
type PostgresCallService struct {
	db             *sql.DB
	eslConn        *freeswitch.ESLConnection
	appService     services.ApplicationService 
	accountService services.AccountService // <-- ADDED AccountService
}

// Update NewPostgresCallService constructor
func NewPostgresCallService(db *sql.DB, eslConn *freeswitch.ESLConnection, appService services.ApplicationService, accountService services.AccountService) *PostgresCallService {
	return &PostgresCallService{
		db:             db,
		eslConn:        eslConn,
		appService:     appService,
		accountService: accountService, // <-- Store AccountService
	}
}

// getGatewayPrefix helper method
func (s *PostgresCallService) getGatewayPrefix(ctx context.Context, accountSid string, callTo string) (string, error) {
    if s.accountService == nil {
        log.Printf("WARN: AccountService not available in CallService. Cannot fetch account-specific gateway settings for AccountSID: %s. Using system default.", accountSid)
        // Fallback to system default if accountService is not wired up (should not happen with proper init)
        defaultSystemGateway := os.Getenv("FS_DEFAULT_GATEWAY_NAME")
        if defaultSystemGateway != "" {
            return fmt.Sprintf("sofia/gateway/%s/", defaultSystemGateway), nil
        }
        return "", nil // No prefix
    }

	account, err := s.accountService.GetAccount(ctx, accountSid)
	if err != nil {
		// Log the error but proceed to system default, or potentially fail if account is mandatory for routing
		log.Printf("WARN: Could not retrieve account %s for gateway settings: %v. Attempting system default gateway.", accountSid, err)
        // Fallthrough to system default
	} else {
        // Prioritize GatewaySelectionScript if present
        if account.GatewaySelectionScript != "" {
            log.Printf("INFO: Using GatewaySelectionScript '%s' for AccountSID: %s", account.GatewaySelectionScript, accountSid)
            // The script needs the number to dial as an argument.
            // Ensure 'callTo' doesn't have problematic characters for a Lua arg, though it usually doesn't.
            return fmt.Sprintf("lua(%s %s)", account.GatewaySelectionScript, callTo), nil 
        }
        // Then DefaultOutboundGateway
		if account.DefaultOutboundGateway != "" {
			log.Printf("INFO: Using DefaultOutboundGateway '%s' for AccountSID: %s", account.DefaultOutboundGateway, accountSid)
			return fmt.Sprintf("sofia/gateway/%s/", account.DefaultOutboundGateway), nil
		}
	}

	// If no account-specific gateway, try system default
	defaultSystemGateway := os.Getenv("FS_DEFAULT_GATEWAY_NAME")
	if defaultSystemGateway != "" {
		log.Printf("INFO: Using system default gateway '%s' for AccountSID: %s", defaultSystemGateway, accountSid)
		return fmt.Sprintf("sofia/gateway/%s/", defaultSystemGateway), nil
	}

	log.Printf("INFO: No specific or system default gateway found for AccountSID: %s. Calls will use FreeSWITCH's default routing for destination '%s'.", accountSid, callTo)
	return "", nil // No prefix, rely on FreeSWITCH's default routing
}


// CreateCall method - modification for gateway selection
func (s *PostgresCallService) CreateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.Sid = "CA" + uuid.NewString()
	call.Status = models.CallStatusInitiating 
	now := time.Now().UTC()
	call.DateCreated = now
	call.DateUpdated = now
	if call.StartTime.IsZero() {
		call.StartTime = now
	}

	var timeoutSeconds sql.NullInt64
	if call.Timeout != "" { 
		if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil {
			timeoutSeconds.Int64 = parsedTimeout
			timeoutSeconds.Valid = true
		}
	}
    var appSidForDB sql.NullString
    if call.ApplicationSid != "" {
        appSidForDB.String = call.ApplicationSid
        appSidForDB.Valid = true
    }
    var answerURLForDB sql.NullString
    if call.AnswerUrl != "" {
        answerURLForDB.String = call.AnswerUrl
        answerURLForDB.Valid = true
    }

	insertQuery := `
		INSERT INTO calls (
			sid, account_sid, caller_id, call_to, answer_url, application_sid, status, 
			direction, duration_seconds, price, start_time, end_time, 
			date_created, date_updated, answered_by, timeout_seconds, freeswitch_call_id 
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NULL)
		RETURNING date_created, date_updated, start_time;`
	
	// Use a different error variable name for the DB operation to avoid shadowing
	dbErr := s.db.QueryRowContext(ctx, insertQuery,
		call.Sid, call.AccountSid, call.CallerId, call.CallTo, 
        answerURLForDB, appSidForDB, call.Status,
		call.Direction, call.Duration, call.Price, call.StartTime, sql.NullTime{Time: call.EndTime, Valid: !call.EndTime.IsZero()},
		call.DateCreated, call.DateUpdated, sql.NullString{String: call.AnsweredBy, Valid: call.AnsweredBy != ""},
		timeoutSeconds,
	).Scan(&call.DateCreated, &call.DateUpdated, &call.StartTime)

	if dbErr != nil { // Check dbErr
		return nil, fmt.Errorf("failed to insert initial call record: %w", dbErr)
	}

	if s.eslConn == nil {
		log.Printf("WARN: ESLConnection is nil for call %s. Skipping FreeSWITCH origination.", call.Sid)
		call.Status = models.CallStatusFailed
		updatedCallWithFailure, updateErr := s.UpdateCall(ctx, call) 
        if updateErr != nil {
            return call, fmt.Errorf("failed to update call status after ESL connection error: %w (call SID: %s)", updateErr, call.Sid)
        }
		return updatedCallWithFailure, fmt.Errorf("ESL connection not available, FreeSWITCH origination skipped for call %s", call.Sid)
	}

	var channelVars []string
	channelVars = append(channelVars, fmt.Sprintf("agbara_call_sid=%s", call.Sid))
	channelVars = append(channelVars, fmt.Sprintf("agbara_account_sid=%s", call.AccountSid))
	if call.ApplicationSid != "" {
		channelVars = append(channelVars, fmt.Sprintf("agbara_application_sid=%s", call.ApplicationSid))
	}
	callerIDName := call.CallerId 
	callerIDNumber := call.CallerId
	if strings.Contains(call.CallerId, "<") && strings.HasSuffix(call.CallerId, ">") {
		parts := strings.SplitN(call.CallerId, "<", 2)
		callerIDName = strings.TrimSpace(parts[0])
		callerIDNumber = strings.TrimSuffix(strings.TrimSpace(parts[1]), ">")
	}
	channelVars = append(channelVars, fmt.Sprintf("origination_caller_id_name='%s'", strings.ReplaceAll(callerIDName, "'", "\\'")))
	channelVars = append(channelVars, fmt.Sprintf("origination_caller_id_number='%s'", strings.ReplaceAll(callerIDNumber, "'", "\\'")))
	if call.Timeout != "" { 
        if callTimeoutSec, errConv := strconv.Atoi(call.Timeout); errConv == nil && callTimeoutSec > 0 { // Renamed err to errConv
            channelVars = append(channelVars, fmt.Sprintf("call_timeout=%d", callTimeoutSec))
        }
    }
	originateVarsString := "{" + strings.Join(channelVars, ",") + "}"
	
	var fsAppString string
	if call.ApplicationSid != "" && s.appService != nil {
		app, errApp := s.appService.GetApplication(ctx, call.ApplicationSid) // Renamed err to errApp
		if errApp != nil {
			log.Printf("WARN: Failed to get application %s for call %s: %v. Using direct AnswerUrl if available.", call.ApplicationSid, call.Sid, errApp)
            if call.AnswerUrl == "" { 
                call.Status = models.CallStatusFailed
                _, _ = s.UpdateCall(ctx, call)
                return call, fmt.Errorf("failed to get application %s and no fallback AnswerUrl for call %s", call.ApplicationSid, call.Sid)
            }
            fsAppString = call.AnswerUrl 
		} else {
            if app.VoiceUrl == "" {
                log.Printf("WARN: Application %s has no VoiceUrl for call %s. Using direct AnswerUrl if available.", app.Sid, call.Sid)
                 if call.AnswerUrl == "" {
                    call.Status = models.CallStatusFailed
                    _, _ = s.UpdateCall(ctx, call)
                    return call, fmt.Errorf("application %s has no VoiceUrl and no fallback AnswerUrl for call %s", app.Sid, call.Sid)
                }
                fsAppString = call.AnswerUrl
            } else if strings.HasPrefix(app.VoiceUrl, "http") {
                fsAppString = fmt.Sprintf("lua(handle_agbara_voiceurl.lua %s %s %s %s)", 
                                   app.VoiceUrl, call.Sid, call.AccountSid, call.ApplicationSid)
            } else {
                fsAppString = app.VoiceUrl 
            }
		}
	} else if call.AnswerUrl != "" {
		fsAppString = call.AnswerUrl
	} else {
		log.Printf("ERROR: No ApplicationSid or AnswerUrl for call %s\n", call.Sid)
		call.Status = models.CallStatusFailed
		_, _ = s.UpdateCall(ctx, call) 
		return call, fmt.Errorf("no application/answer URL provided for call %s", call.Sid)
	}

    gatewayPrefix, gwErr := s.getGatewayPrefix(ctx, call.AccountSid, call.To)
    if gwErr != nil {
        log.Printf("WARN: Failed to determine gateway for account %s: %v. Proceeding without specific prefix or failing.", call.AccountSid, gwErr)
        // Depending on policy, might fail call here
        // call.Status = models.CallStatusFailed
        // _, _ = s.UpdateCall(ctx, call)
        // return call, fmt.Errorf("gateway configuration error for account %s: %w", call.AccountSid, gwErr)
    }

    destinationString := call.To
    if strings.HasPrefix(gatewayPrefix, "lua(") { 
        // If gatewayPrefix is a Lua script, it takes precedence and typically forms the whole dial string part before the app.
        // The Lua script itself would handle the 'call.To' (destination number).
        destinationString = "" // Lua script handles the destination.
    }

	dialString := fmt.Sprintf("%s%s%s %s", originateVarsString, gatewayPrefix, destinationString, fsAppString)
	
	log.Printf("Attempting to originate call %s with dial string: %s\n", call.Sid, dialString)
	fsCallID, originateErr := s.eslConn.SendBgApiCommand("originate", dialString)

	if originateErr != nil {
		log.Printf("ERROR: FreeSWITCH origination failed for call %s: %v\n", call.Sid, originateErr)
		call.Status = models.CallStatusFailed
	} else {
		log.Printf("FreeSWITCH origination successful for call %s. Job UUID: %s\n", call.Sid, fsCallID)
		call.FreeswitchCallID = fsCallID
		call.Status = models.CallStatusRinging 
	}
	
	updatedCall, updateErr := s.UpdateCall(ctx, call)
	if updateErr != nil {
		log.Printf("ERROR: Failed to update call %s with FS details: %v\n", call.Sid, updateErr)
        if originateErr == nil { 
            return call, fmt.Errorf("origination succeeded (FS ID: %s) but failed to update call record: %w", fsCallID, updateErr)
        }
		// Both origination and DB update failed. Return the call object with its current state.
		return call, fmt.Errorf("origination failed (%v) AND failed to update call record (%w)", originateErr, updateErr)
	}
	
    if originateErr != nil { // Origination failed, but DB update of this failure was successful
        return updatedCall, fmt.Errorf("FreeSWITCH origination command failed: %w", originateErr)
    }
	return updatedCall, nil
}


// scanCall, UpdateCall, GetCall, ListCalls, UpdateCallStatus methods
// (These should be the versions from Turn 88/89 that handle all necessary fields)
func scanCall(scanner interface{ Scan(...interface{}) error }) (*models.Call, error) {
	call := &models.Call{}
    var timeoutSeconds sql.NullInt64
    var answeredBy sql.NullString
    var endTime sql.NullTime
    var fsCallID sql.NullString 
    var appSid sql.NullString   
    var answerURL sql.NullString 

	err := scanner.Scan(
		&call.Sid, &call.AccountSid, &call.CallerId, &call.CallTo, &answerURL, &appSid, &call.Status,
		&timeoutSeconds, &call.Direction, &call.Duration, &call.Price,
		&call.StartTime, &endTime, &call.DateCreated, &call.DateUpdated, &answeredBy,
        &fsCallID,
	)
	if err != nil { return nil, err }
    if timeoutSeconds.Valid { call.Timeout = fmt.Sprintf("%d", timeoutSeconds.Int64) } else { call.Timeout = "" }
    call.AnsweredBy = answeredBy.String
    if endTime.Valid { call.EndTime = endTime.Time } else { call.EndTime = time.Time{} } 
    call.FreeswitchCallID = fsCallID.String
    call.ApplicationSid = appSid.String
    call.AnswerUrl = answerURL.String
	return call, nil
}
func (s *PostgresCallService) UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error) {
	call.DateUpdated = time.Now().UTC()
    var timeoutSeconds sql.NullInt64
    if call.Timeout != "" { if parsedTimeout, err := strconv.ParseInt(call.Timeout, 10, 64); err == nil { timeoutSeconds.Int64 = parsedTimeout; timeoutSeconds.Valid = true } }
    var fsCallID sql.NullString
    if call.FreeswitchCallID != "" { fsCallID.String = call.FreeswitchCallID; fsCallID.Valid = true }
    var appSid sql.NullString
    if call.ApplicationSid != "" { appSid.String = call.ApplicationSid; appSid.Valid = true }
    var answerURL sql.NullString
    if call.AnswerUrl != "" { answerURL.String = call.AnswerUrl; answerURL.Valid = true }

	query := `
		UPDATE calls SET
			account_sid = $1, caller_id = $2, call_to = $3, answer_url = $4, application_sid = $5, status = $6,
			timeout_seconds = $7, direction = $8, duration_seconds = $9, price = $10,
			start_time = $11, end_time = $12, date_updated = $13, answered_by = $14,
            freeswitch_call_id = $15 
		WHERE sid = $16
		RETURNING sid, account_sid, caller_id, call_to, answer_url, application_sid, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id;
	`
	row := s.db.QueryRowContext(ctx, query,
		call.AccountSid, call.CallerId, call.CallTo, answerURL, appSid, call.Status,
		timeoutSeconds, call.Direction, call.Duration, call.Price,
		call.StartTime, sql.NullTime{Time: call.EndTime, Valid: !call.EndTime.IsZero()}, 
        call.DateUpdated, sql.NullString{String: call.AnsweredBy, Valid: call.AnsweredBy != ""},
        fsCallID, call.Sid,
	)
	return scanCall(row)
}
func (s *PostgresCallService) GetCall(ctx context.Context, callSid string) (*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, application_sid, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id
		FROM calls WHERE sid = $1;
	`
	row := s.db.QueryRowContext(ctx, query, callSid)
    call, err := scanCall(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) { // Use errors.Is for sql.ErrNoRows
            return nil, fmt.Errorf("call with SID %s not found: %w", callSid, err)
        }
        return nil, fmt.Errorf("failed to get call %s: %w", callSid, err)
    }
    return call, nil
}
func (s *PostgresCallService) ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error) {
	query := `
		SELECT sid, account_sid, caller_id, call_to, answer_url, application_sid, status, 
		       timeout_seconds, direction, duration_seconds, price, 
		       start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id
		FROM calls WHERE account_sid = $1 ORDER BY date_created DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, accountSid)
	if err != nil { return nil, fmt.Errorf("failed to query calls for account %s: %w", accountSid, err) }
	defer rows.Close()
	var calls []*models.Call
	for rows.Next() {
		call, scanErr := scanCall(rows)
		if scanErr != nil { return nil, fmt.Errorf("failed to scan call row during ListCalls: %w", scanErr) }
		calls = append(calls, call)
	}
	if rowsErr := rows.Err(); rowsErr != nil { return nil, fmt.Errorf("error iterating call rows for account %s: %w", accountSid, rowsErr) } // Renamed err to rowsErr
    if calls == nil { calls = []*models.Call{} }
	return calls, nil
}
func (s *PostgresCallService) UpdateCallStatus(ctx context.Context, callSid string, status models.CallStatus) (*models.Call, error) {
	query := `
		UPDATE calls SET status = $1, date_updated = $2
		WHERE sid = $3
		RETURNING sid, account_sid, caller_id, call_to, answer_url, application_sid, status, 
		          timeout_seconds, direction, duration_seconds, price, 
		          start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id;
	`
	row := s.db.QueryRowContext(ctx, query, status, time.Now().UTC(), callSid)
    updatedCall, err := scanCall(row)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) { // Use errors.Is for sql.ErrNoRows
            return nil, fmt.Errorf("call with SID %s not found for status update: %w", callSid, err)
        }
        return nil, fmt.Errorf("failed to update call status for SID %s: %w", callSid, err)
    }
    return updatedCall, nil
}
