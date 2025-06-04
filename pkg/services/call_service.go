package services

import (
	"agbara-go/pkg/freeswitch"
	"agbara-go/pkg/freeswitch_events"
	"agbara-go/pkg/models"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os" 
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Updated CallService interface
type CallService interface {
	CreateCall(ctx context.Context, call *models.Call, callRequest *models.CallRequest) (*models.Call, error) // Pass full CallRequest
	GetCall(ctx context.Context, callSid string) (*models.Call, error)
	ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error)
	UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error)
	UpdateCallStatus(ctx context.Context, callSid string, status models.CallStatus) (*models.Call, error)
}

// PostgresCallService struct (from Turn 110/114)
type PostgresCallService struct {
	db             *sql.DB
	eslConn        *freeswitch.ESLConnection
	appService     ApplicationService 
	accountService AccountService     
	eventDispatcher *freeswitch_events.EventDispatcher 
}

// NewPostgresCallService constructor (from Turn 110/114)
func NewPostgresCallService(db *sql.DB, eslConn *freeswitch.ESLConnection, 
                            appService ApplicationService, accService AccountService, 
                            evtDisp *freeswitch_events.EventDispatcher) *PostgresCallService {
	s := &PostgresCallService{
		db:             db,
		eslConn:        eslConn,
		appService:     appService,
		accountService: accService, 
		eventDispatcher: evtDisp,
	}
	if evtDisp != nil {
		log.Println("CallService: Subscribing to FreeSWITCH events: CHANNEL_CREATE, CHANNEL_ANSWER, CHANNEL_HANGUP_COMPLETE, CHANNEL_PROGRESS_MEDIA")
		evtDisp.Subscribe("CHANNEL_CREATE", s.handleChannelCreate)
		evtDisp.Subscribe("CHANNEL_ANSWER", s.handleChannelAnswer)
		evtDisp.Subscribe("CHANNEL_HANGUP_COMPLETE", s.handleChannelHangupComplete)
		evtDisp.Subscribe("CHANNEL_PROGRESS_MEDIA", s.handleChannelProgressMedia)
	}
	return s
}


// Modify CreateCall to accept the full CallRequest for transient parameters
func (s *PostgresCallService) CreateCall(ctx context.Context, call *models.Call, callRequest *models.CallRequest) (*models.Call, error) {
	// Initial call setup (SID, Status, Timestamps)
	call.Sid = "CA" + uuid.NewString()
	call.Status = models.CallStatusInitiating 
	now := time.Now().UTC()
	call.DateCreated = now
	call.DateUpdated = now
	if call.StartTime.IsZero() {
		call.StartTime = now
	}
    // Populate call.Timeout from callRequest.TimeLimit
    call.Timeout = callRequest.TimeLimit 

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
	
	dbErr := s.db.QueryRowContext(ctx, insertQuery,
		call.Sid, call.AccountSid, call.CallerId, call.CallTo, 
        answerURLForDB, appSidForDB, call.Status,
		call.Direction, call.Duration, call.Price, call.StartTime, sql.NullTime{Time: call.EndTime, Valid: !call.EndTime.IsZero()},
		call.DateCreated, call.DateUpdated, sql.NullString{String: call.AnsweredBy, Valid: call.AnsweredBy != ""},
		timeoutSeconds,
	).Scan(&call.DateCreated, &call.DateUpdated, &call.StartTime)

	if dbErr != nil {
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

	// --- Construct Originate Parameters ---
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
        if callTimeoutSec, errConv := strconv.Atoi(call.Timeout); errConv == nil && callTimeoutSec > 0 {
            channelVars = append(channelVars, fmt.Sprintf("call_timeout=%d", callTimeoutSec))
        }
    }

    // --- NEW: Transient CallRequest Parameters as Channel Variables ---
    if callRequest.SendDigits != "" {
        safeSendDigits := SanitizeSendDigits(callRequest.SendDigits) 
        if safeSendDigits != "" {
            channelVars = append(channelVars, fmt.Sprintf("execute_on_answer='send_dtmf %s'", safeSendDigits))
        }
    }
    if callRequest.StatusCallbackUrl != "" {
        channelVars = append(channelVars, fmt.Sprintf("AGBARA_STATUS_CALLBACK_URL='%s'", callRequest.StatusCallbackUrl))
        if callRequest.StatusCallbackMethod != "" {
            channelVars = append(channelVars, fmt.Sprintf("AGBARA_STATUS_CALLBACK_METHOD='%s'", callRequest.StatusCallbackMethod))
        } else {
            channelVars = append(channelVars, fmt.Sprintf("AGBARA_STATUS_CALLBACK_METHOD='POST'")) // Default method
        }
    }
    if callRequest.HangupOnRing != "" {
        channelVars = append(channelVars, fmt.Sprintf("AGBARA_HANGUP_ON_RING='%s'", callRequest.HangupOnRing))
    }
    // --- End of New Transient Parameters ---

	originateVarsString := "{" + strings.Join(channelVars, ",") + "}"
	
	var fsAppString string
	if call.ApplicationSid != "" && s.appService != nil {
		app, errApp := s.appService.GetApplication(ctx, call.ApplicationSid) 
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
        log.Printf("WARN: Failed to determine gateway for account %s: %v. Using default/no prefix.", call.AccountSid, gwErr)
    }
    
    destinationPart := call.To
    if strings.HasPrefix(gatewayPrefix, "lua(") { 
        destinationPart = "" 
    }

	dialString := fmt.Sprintf("%s%s%s %s", originateVarsString, gatewayPrefix, destinationPart, fsAppString)
	
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
		return call, fmt.Errorf("origination failed (%v) AND failed to update call record (%w)", originateErr, updateErr)
	}
	
    if originateErr != nil { 
        return updatedCall, fmt.Errorf("FreeSWITCH origination command failed: %w", originateErr)
    }
	return updatedCall, nil
}

// SanitizeSendDigits helper function
func SanitizeSendDigits(digits string) string {
    validChars := "0123456789*#wW"
    var result strings.Builder
    for _, char := range digits {
        if strings.ContainsRune(validChars, char) {
            result.WriteRune(char)
        }
    }
    return result.String()
}

// Event handlers (from Turn 110/114)
func (s *PostgresCallService) getCallByFreeswitchOrAgbaraSid(ctx context.Context, eventUniqueID string, agbaraCallSidFromEvent string) (*models.Call, error) {
    var call *models.Call; var err error
    query := `SELECT sid, account_sid, caller_id, call_to, answer_url, application_sid, status, timeout_seconds, direction, duration_seconds, price, start_time, end_time, date_created, date_updated, answered_by, freeswitch_call_id FROM calls WHERE freeswitch_call_id = $1;`
    row := s.db.QueryRowContext(ctx, query, eventUniqueID)
    call, err = scanCall(row)
    if errors.Is(err, sql.ErrNoRows) {
        if agbaraCallSidFromEvent != "" {
            log.Printf("CallService: Call not found by FS UUID %s, trying agbara_call_sid %s.\n", eventUniqueID, agbaraCallSidFromEvent)
            return s.GetCall(ctx, agbaraCallSidFromEvent) 
        }
        log.Printf("CallService: Call not found by FS UUID %s and no agbara_call_sid in event. Trying event.UniqueID %s as Agbara SID.\n", eventUniqueID, eventUniqueID)
        return s.GetCall(ctx, eventUniqueID)
    }
    return call, err
}
func (s *PostgresCallService) handleChannelCreate(event *freeswitch_events.ParsedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	fsUUID := event.UniqueID; agbaraCallSid := event.Headers["variable_agbara_call_sid"]
	log.Printf("CallService: Handling CHANNEL_CREATE event. FS UUID: %s, AgbaraCallSid Var: %s\n", fsUUID, agbaraCallSid)
    call, err := s.getCallByFreeswitchOrAgbaraSid(ctx, fsUUID, agbaraCallSid)
    if err != nil { if errors.Is(err, sql.ErrNoRows) {log.Printf("CallService: CHANNEL_CREATE: Call not found (FS UUID: %s, Agbara SID Var: %s).\n", fsUUID, agbaraCallSid)}else{log.Printf("CallService: CHANNEL_CREATE: Error fetching call (FS UUID: %s, Agbara SID Var: %s): %v\n", fsUUID, agbaraCallSid, err)}; return }
    if call.FreeswitchCallID == "" || call.FreeswitchCallID != fsUUID { call.FreeswitchCallID = fsUUID }
    if call.Status == models.CallStatusQueued { call.Status = models.CallStatusInitiating }
	call.DateUpdated = time.Now().UTC()
	updatedCall, errUpdate := s.UpdateCall(ctx, call)
	if errUpdate != nil { log.Printf("CallService: CHANNEL_CREATE: Failed to update call %s: %v\n", call.Sid, errUpdate)
	} else { log.Printf("CallService: CHANNEL_CREATE: Call %s (FS %s) updated, status: %s\n", updatedCall.Sid, updatedCall.FreeswitchCallID, updatedCall.Status) }
}
func (s *PostgresCallService) handleChannelProgressMedia(event *freeswitch_events.ParsedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	fsUUID := event.UniqueID; agbaraCallSid := event.Headers["variable_agbara_call_sid"]
	log.Printf("CallService: Handling CHANNEL_PROGRESS_MEDIA. FS UUID: %s, AgbaraCallSid Var: %s\n", fsUUID, agbaraCallSid)
	call, err := s.getCallByFreeswitchOrAgbaraSid(ctx, fsUUID, agbaraCallSid)
	if err != nil { log.Printf("CallService: PROGRESS_MEDIA: Call not found (FS UUID: %s, Agbara SID Var: %s): %v\n", fsUUID, agbaraCallSid, err); return }
	if call.Status == models.CallStatusQueued || call.Status == models.CallStatusInitiating {
		call.Status = models.CallStatusRinging; call.DateUpdated = time.Now().UTC()
        if call.FreeswitchCallID == "" || call.FreeswitchCallID != fsUUID { call.FreeswitchCallID = fsUUID }
		updatedCall, errUpdate := s.UpdateCall(ctx, call)
		if errUpdate != nil {log.Printf("CallService: PROGRESS_MEDIA: Failed to update call %s: %v\n", call.Sid, errUpdate)
		} else { log.Printf("CallService: PROGRESS_MEDIA: Call %s (FS %s) updated, status: %s\n", updatedCall.Sid, updatedCall.FreeswitchCallID, updatedCall.Status) }
	} else { log.Printf("CallService: PROGRESS_MEDIA: Call %s already in status %s.\n", call.Sid, call.Status) }
}
func (s *PostgresCallService) handleChannelAnswer(event *freeswitch_events.ParsedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	fsUUID := event.UniqueID; agbaraCallSid := event.Headers["variable_agbara_call_sid"]
	log.Printf("CallService: Handling CHANNEL_ANSWER. FS UUID: %s, AgbaraCallSid Var: %s\n", fsUUID, agbaraCallSid)
	call, err := s.getCallByFreeswitchOrAgbaraSid(ctx, fsUUID, agbaraCallSid)
	if err != nil { log.Printf("CallService: ANSWER: Call not found (FS UUID: %s, Agbara SID Var: %s): %v\n", fsUUID, agbaraCallSid, err); return }
	call.Status = models.CallStatusInProgress
    if event.Timestamp > 0 { call.StartTime = time.Unix(event.Timestamp, 0).UTC() 
    } else if answerTimeStr := event.Headers["Answer-Time"]; answerTimeStr != "" { if ansTS, eC := strconv.ParseInt(answerTimeStr,10,64); eC==nil {call.StartTime = time.Unix(ansTS/1000000,(ansTS%1000000)*1000).UTC()} else {call.StartTime=time.Now().UTC()}} else {call.StartTime=time.Now().UTC()}
	call.DateUpdated = time.Now().UTC(); if answeredBy, ok := event.Headers["Answered-By"]; ok { call.AnsweredBy = answeredBy }; if call.FreeswitchCallID == "" || call.FreeswitchCallID != fsUUID { call.FreeswitchCallID = fsUUID }
	updatedCall, errUpdate := s.UpdateCall(ctx, call)
	if errUpdate != nil { log.Printf("CallService: ANSWER: Failed to update call %s: %v\n", call.Sid, errUpdate)
	} else { log.Printf("CallService: ANSWER: Call %s (FS %s) updated, status: %s, StartTime: %s\n", updatedCall.Sid, updatedCall.FreeswitchCallID, updatedCall.Status, updatedCall.StartTime.String()) }
}
func (s *PostgresCallService) handleChannelHangupComplete(event *freeswitch_events.ParsedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second); defer cancel()
	fsUUID := event.UniqueID; agbaraCallSid := event.Headers["variable_agbara_call_sid"]
	log.Printf("CallService: Handling CHANNEL_HANGUP_COMPLETE. FS UUID: %s, AgbaraCallSid Var: %s. Headers: %+v\n", fsUUID, agbaraCallSid, event.Headers)
	call, err := s.getCallByFreeswitchOrAgbaraSid(ctx, fsUUID, agbaraCallSid)
	if err != nil { log.Printf("CallService: HANGUP_COMPLETE: Call not found (FS UUID: %s, Agbara SID Var: %s): %v\n", fsUUID, agbaraCallSid, err); return }
    if call.Status == models.CallStatusCompleted || call.Status == models.CallStatusFailed || call.Status == models.CallStatusBusy || call.Status == models.CallStatusNoAnswer || call.Status == models.CallStatusCanceled { log.Printf("CallService: HANGUP_COMPLETE: Call %s already in terminal state (%s).\n", call.Sid, call.Status); return }
    hangupCause := event.Headers["Hangup-Cause"]; billsecStr := event.Headers["variable_billsec"]; durationStr := event.Headers["variable_duration"]
    if call.Status == models.CallStatusInProgress { call.Status = models.CallStatusCompleted
    } else { switch hangupCause { case "NORMAL_CLEARING","ORIGINATOR_CANCEL": if call.Status==models.CallStatusRinging||call.Status==models.CallStatusInitiating {call.Status=models.CallStatusCanceled} else{call.Status=models.CallStatusFailed}; case "USER_BUSY":call.Status=models.CallStatusBusy; case "NO_ANSWER","NORMAL_TEMPORARY_FAILURE","NO_USER_RESPONSE","CALL_REJECTED":call.Status=models.CallStatusNoAnswer; case "UNALLOCATED_NUMBER","NORMAL_UNSPECIFIED","DESTINATION_OUT_OF_ORDER","INVALID_NUMBER_FORMAT":call.Status=models.CallStatusFailed; default: call.Status=models.CallStatusFailed } }
    if event.Timestamp > 0 { call.EndTime = time.Unix(event.Timestamp, 0).UTC() } else { call.EndTime = time.Now().UTC() }
    if !call.StartTime.IsZero() && call.EndTime.Before(call.StartTime) { call.EndTime = call.StartTime }
	if billsec, eC := strconv.Atoi(billsecStr); eC == nil && call.Status == models.CallStatusCompleted { call.Duration = billsec } else if duration, eC := strconv.Atoi(durationStr); eC == nil { call.Duration = duration }
    if call.FreeswitchCallID == "" || call.FreeswitchCallID != fsUUID { call.FreeswitchCallID = fsUUID }
	call.DateUpdated = time.Now().UTC()
	updatedCall, errUpdate := s.UpdateCall(ctx, call)
	if errUpdate != nil { log.Printf("CallService: HANGUP_COMPLETE: Failed to update call %s: %v\n", call.Sid, errUpdate)
	} else { log.Printf("CallService: HANGUP_COMPLETE: Call %s (FS %s) final status: %s, Duration: %d, EndTime: %s\n", updatedCall.Sid, updatedCall.FreeswitchCallID, updatedCall.Status, updatedCall.Duration, updatedCall.EndTime.String())}
}
func (s *PostgresCallService) UpdateCall(ctx context.Context, call *models.Call) (*models.Call, error) { /* ... from Turn 98 ... */ 
	call.DateUpdated = time.Now().UTC()
    var ts sql.NullInt64; if call.Timeout != "" { if p,e := strconv.ParseInt(call.Timeout,10,64);e==nil { ts.Int64=p;ts.Valid=true } }
    var fs sql.NullString; if call.FreeswitchCallID != "" { fs.String=call.FreeswitchCallID;fs.Valid=true }
    var as sql.NullString; if call.ApplicationSid != "" { as.String=call.ApplicationSid;as.Valid=true }
    var au sql.NullString; if call.AnswerUrl != "" { au.String=call.AnswerUrl;au.Valid=true }
	q := `UPDATE calls SET account_sid=$1,caller_id=$2,call_to=$3,answer_url=$4,application_sid=$5,status=$6,timeout_seconds=$7,direction=$8,duration_seconds=$9,price=$10,start_time=$11,end_time=$12,date_updated=$13,answered_by=$14,freeswitch_call_id=$15 WHERE sid=$16 RETURNING sid,account_sid,caller_id,call_to,answer_url,application_sid,status,timeout_seconds,direction,duration_seconds,price,start_time,end_time,date_created,date_updated,answered_by,freeswitch_call_id`
	r := s.db.QueryRowContext(ctx,q,call.AccountSid,call.CallerId,call.CallTo,au,as,call.Status,ts,call.Direction,call.Duration,call.Price,call.StartTime,sql.NullTime{Time:call.EndTime,Valid:!call.EndTime.IsZero()},call.DateUpdated,sql.NullString{String:call.AnsweredBy,Valid:call.AnsweredBy!=""},fs,call.Sid)
	return scanCall(r)
}
func (s *PostgresCallService) GetCall(ctx context.Context, callSid string) (*models.Call, error) { /* ... from Turn 98 ... */ 
    q := `SELECT sid,account_sid,caller_id,call_to,answer_url,application_sid,status,timeout_seconds,direction,duration_seconds,price,start_time,end_time,date_created,date_updated,answered_by,freeswitch_call_id FROM calls WHERE sid=$1`
    call, err := scanCall(s.db.QueryRowContext(ctx, q, callSid)); if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("call with SID %s not found: %w", callSid, err) } ; return nil, fmt.Errorf("failed to get call %s: %w", callSid, err) } ; return call, nil
}
func (s *PostgresCallService) ListCalls(ctx context.Context, accountSid string) ([]*models.Call, error) { /* ... from Turn 98 ... */
    q := `SELECT sid,account_sid,caller_id,call_to,answer_url,application_sid,status,timeout_seconds,direction,duration_seconds,price,start_time,end_time,date_created,date_updated,answered_by,freeswitch_call_id FROM calls WHERE account_sid=$1 ORDER BY date_created DESC`
    rs, e := s.db.QueryContext(ctx, q, accountSid); if e != nil { return nil, e }; defer rs.Close(); var cs []*models.Call
    for rs.Next() { c,eS:=scanCall(rs); if eS!=nil{return nil,eS}; cs=append(cs,c) }; if eR:=rs.Err();eR!=nil{return nil,eR}; if cs==nil{cs=[]*models.Call{}}; return cs,nil
}
func (s *PostgresCallService) UpdateCallStatus(ctx context.Context, callSid string, status models.CallStatus) (*models.Call, error) { /* ... from Turn 98 ... */
    q := `UPDATE calls SET status=$1,date_updated=$2 WHERE sid=$3 RETURNING sid,account_sid,caller_id,call_to,answer_url,application_sid,status,timeout_seconds,direction,duration_seconds,price,start_time,end_time,date_created,date_updated,answered_by,freeswitch_call_id`
    call, err := scanCall(s.db.QueryRowContext(ctx, q, status, time.Now().UTC(), callSid)); if err != nil { if errors.Is(err, sql.ErrNoRows) { return nil, fmt.Errorf("call with SID %s not found for status update: %w", callSid, err) }; return nil, fmt.Errorf("failed to update call status for SID %s: %w", callSid, err) }; return call, nil
}
func (s *PostgresCallService) getGatewayPrefix(ctx context.Context, accountSid string, callTo string) (string, error) { /* ... from Turn 98 ... */
    if s.accountService == nil { log.Println("WARN: AccountService not available..."); gateway := os.Getenv("FS_DEFAULT_GATEWAY_NAME"); if gateway != "" { return fmt.Sprintf("sofia/gateway/%s/", gateway), nil }; return "", nil }
    acc, err := s.accountService.GetAccount(ctx, accountSid)
    if err != nil { log.Printf("WARN: Could not fetch account %s: %v", accountSid, err); gateway := os.Getenv("FS_DEFAULT_GATEWAY_NAME"); if gateway != "" { return fmt.Sprintf("sofia/gateway/%s/", gateway), nil }; return "", nil }
    if acc.GatewaySelectionScript != "" { return fmt.Sprintf("lua(%s %s)", acc.GatewaySelectionScript, callTo), nil }
    if acc.DefaultOutboundGateway != "" { return fmt.Sprintf("sofia/gateway/%s/", acc.DefaultOutboundGateway), nil }
    gateway := os.Getenv("FS_DEFAULT_GATEWAY_NAME"); if gateway != "" { return fmt.Sprintf("sofia/gateway/%s/", gateway), nil }
    log.Printf("INFO: No specific gateway for account %s, no system default. Using FS default routing.", accountSid); return "", nil
}
