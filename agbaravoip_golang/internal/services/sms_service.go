package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils" // For GenerateSID
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type smsService struct {
	db            *sqlx.DB
	log           *logrus.Entry
	gatewayClient SMSGatewayClient
}

// NewSMSService creates a new SMSService.
// Note: The prompt for CallService update passes logger.Logger (base logger)
// to NewConferenceService, and NewSMSService. Here we expect *logrus.Logger.
func NewSMSService(db *sqlx.DB, logger *logrus.Logger, gwClient SMSGatewayClient) ISMSService { // Return interface
	return &smsService{
		db:            db,
		log:           logger.WithField("service", "sms"),
		gatewayClient: gwClient,
	}
}

func (s *smsService) SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error) {
	now := time.Now().UTC()
	sms := &domain.SMSMessage{
		SID:          msgSID,
		AccountSID:   accountSid,
		To:           to,
		From:         from,
		Body:         body,
		Status:       domain.SMSStatusQueued,
		Direction:    domain.SMSDirectionOutbound,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	query := `INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, created_at, updated_at)
			  VALUES (:sid, :account_sid, :msg_to, :msg_from, :body, :status, :direction, :created_at, :updated_at)
			  RETURNING id, created_at, updated_at` // Assuming these are auto-populated and we want them back

	stmt, err := s.db.PrepareNamedContext(ctx, query)
	if err != nil { return nil, fmt.Errorf("SendSMS: preparing named query: %w", err) }
	defer stmt.Close()

	// Scan into temporary variables as some fields in sms struct might not have db tags for all returned cols
	var dbID int64
	var dbCreatedAt time.Time
	var dbUpdatedAt time.Time
	if err := stmt.QueryRowxContext(ctx, sms).Scan(&dbID, &dbCreatedAt, &dbUpdatedAt); err != nil {
		if strings.Contains(err.Error(), "unique constraint") &&
		   (strings.Contains(err.Error(), "sms_messages_sid_key") || strings.Contains(err.Error(), "idx_sms_messages_sid")) { // Adjust if index name is different
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("SendSMS: executing insert for SID %s: %w", msgSID, err)
	}
	sms.ID = dbID
	sms.CreatedAt = dbCreatedAt
	sms.UpdatedAt = dbUpdatedAt
	s.log.Infof("SMS %s queued for sending to %s from %s", msgSID, to, from)

	// For ActionURL/StatusCallback from SmsElement, it's usually for gateway to call back to *our* system.
	// So, we need an endpoint that maps to UpdateSMSStatus.
	// This URL should be publicly accessible and ideally include the Agbara SID (msgSID) for correlation.
	// Example: "https://myapp.com/api/v1/sms/callbacks/gateway_dlr/{AgbaraMessageSID}"
	// For now, using a placeholder or a configured base URL.
	// The ActionURL from SmsElement itself might be for *application-level* callbacks after final status.

	// This statusCallback is what *we* give to the *external gateway*.
	// It should point to an endpoint on our app that will eventually call `UpdateSMSStatus`.
	// Let's assume a config provides the base URL for these callbacks.
	// cfgBaseURL := "http://localhost:8080" // This should come from config
	// gatewayCallbackURL := fmt.Sprintf("%s/api/v1/sms/gateway_status/%s", cfgBaseURL, msgSID)
	// For testability, we can make this simpler or pass it in if needed.
	// The prompt for SMSGatewayClient SendSMS just has `statusCallbackURL string`.
	// For now, let's assume the `actionURL` from `SmsElement` is what's passed to the gateway.
	// This might be an oversimplification, as `actionURL` is usually for the app developer,
	// and gateway callbacks are to our system.
	// If `actionURL` IS the URL for the gateway to call, then it needs to be robust.
	// If actionURL is for the *application* after final status, then it should be stored with SMSMessage.
	// Given the current structure, actionURL from SmsElement is likely for application level,
	// and we'd have a separate configured endpoint for gateway DLRs.
	// For now, we pass the SmsElement's actionURL to the gateway if provided.

	gatewaySID, gwErr := s.gatewayClient.SendSMS(ctx, to, from, body, actionURL) // Pass element's actionURL
	if gwErr != nil {
		s.log.Errorf("SMS %s: Gateway SendSMS failed: %v. Marking as failed.", msgSID, gwErr)
		errMsg := gwErr.Error()
		nowFail := time.Now().UTC()
		// Update status to failed immediately
		_ = s.UpdateSMSStatus(ctx, msgSID, &gatewaySID, domain.SMSStatusFailed, nil, &errMsg, &nowFail)
		// Update in-memory struct as well, though UpdateSMSStatus would have updated DB
		sms.Status = domain.SMSStatusFailed
		sms.ErrorMessage = sql.NullString{String: errMsg, Valid: true}
		if gatewaySID != "" {
			sms.GatewayMessageSID = sql.NullString{String: gatewaySID, Valid: true}
		}
		return sms, fmt.Errorf("gateway SendSMS failed: %w", gwErr)
	}

	s.log.Infof("SMS %s sent to gateway, GatewaySID: %s", msgSID, gatewaySID)
	nowSent := time.Now().UTC()
	updateErr := s.UpdateSMSStatus(ctx, msgSID, &gatewaySID, domain.SMSStatusSent, nil, nil, &nowSent)
	if updateErr != nil {
		s.log.Errorf("SMS %s: Failed to update status to SENT after gateway success: %v", msgSID, updateErr)
	}
	sms.Status = domain.SMSStatusSent
	sms.GatewayMessageSID = sql.NullString{String: gatewaySID, Valid: gatewaySID != ""}
	sms.SentAt = sql.NullTime{Time: nowSent, Valid: true}
	return sms, nil
}

// GetSMSBySID retrieves an SMS message by its SID, scoped to an account.
func (s *smsService) GetSMSBySID(ctx context.Context, accountSid string, sid string) (*domain.SMSMessage, error) {
	s.log.Infof("GetSMSBySID: Getting SMS SID %s for account %s", sid, accountSid)
	query := "SELECT * FROM sms_messages WHERE sid = $1 AND account_sid = $2" // Add account_sid
	var sms domain.SMSMessage
	if err := s.db.GetContext(ctx, &sms, query, sid, accountSid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.log.Warnf("GetSMSBySID: SMS %s not found for account %s", sid, accountSid)
			return nil, domain.ErrNotFound
		}
		s.log.Errorf("GetSMSBySID: Error querying SMS %s for account %s: %v", sid, accountSid, err)
		return nil, fmt.Errorf("querying SMS by SID %s for account %s: %w", sid, accountSid, err)
	}
	return &sms, nil
}

// ListSMSMessages lists SMS messages for a specific account, with optional filters.
func (s *smsService) ListSMSMessages(ctx context.Context, accountSid string, filters map[string]interface{}) ([]*domain.SMSMessage, error) {
	s.log.Infof("ListSMSMessages: Listing SMS for account SID: %s with filters: %v", accountSid, filters)

	params := map[string]interface{}{"account_sid": accountSid}
	queryBuilder := strings.Builder{}
	queryBuilder.WriteString("SELECT * FROM sms_messages WHERE account_sid = :account_sid")

	for key, value := range filters {
		switch key {
		case "to", "from", "status", "direction":
			if strVal, ok := value.(string); ok && strVal != "" {
				dbKey := key
				if key == "to" {
					dbKey = "msg_to"
				}
				if key == "from" {
					dbKey = "msg_from"
				}
				queryBuilder.WriteString(fmt.Sprintf(" AND %s = :%s", dbKey, key))
				params[key] = strVal
			}
		case "date_from":
			if dateStr, ok := value.(string); ok {
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					queryBuilder.WriteString(" AND created_at >= :date_from")
					params[key] = t.Format("2006-01-02 00:00:00")
				}
			}
		case "date_to":
			if dateStr, ok := value.(string); ok {
				if t, err := time.Parse("2006-01-02", dateStr); err == nil {
					queryBuilder.WriteString(" AND created_at <= :date_to")
					params[key] = t.Format("2006-01-02 23:59:59")
				}
			}
		}
	}
	queryBuilder.WriteString(" ORDER BY created_at DESC")
	// TODO: Add limit/offset for pagination

	query, args, err := sqlx.Named(queryBuilder.String(), params)
	if err != nil {
		return nil, fmt.Errorf("ListSMSMessages: preparing named query: %w", err)
	}
	query = s.db.Rebind(query)

	var messages []*domain.SMSMessage
	if err := s.db.SelectContext(ctx, &messages, query, args...); err != nil {
		s.log.Errorf("ListSMSMessages: Error querying SMS for account %s: %v", accountSid, err)
		return nil, fmt.Errorf("querying SMS for account %s: %w", accountSid, err)
	}
	return messages, nil
}

// SendSMSViaAPI handles sending an SMS message initiated via an API call.
func (s *smsService) SendSMSViaAPI(ctx context.Context, accountSid, from, to, body string, statusCallbackURL *string) (*domain.SMSMessage, error) {
	s.log.Infof("SendSMSViaAPI: Attempting to send SMS from %s to %s for account %s", from, to, accountSid)
	if accountSid == "" || from == "" || to == "" || body == "" {
		return nil, fmt.Errorf("SendSMSViaAPI: %w: accountSid, from, to, and body are required", domain.ErrValidationFailed)
	}

	msgSID := utils.GenerateSID("SM")
	now := time.Now().UTC()

	sms := &domain.SMSMessage{
		SID:        msgSID,
		AccountSID: accountSid,
		To:         to,
		From:       from,
		Body:       body,
		Status:     domain.SMSStatusQueued,
		Direction:  domain.SMSDirectionOutboundAPI, // Specific direction for API originated SMS
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	query := `INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, created_at, updated_at)
			  VALUES (:sid, :account_sid, :msg_to, :msg_from, :body, :status, :direction, :created_at, :updated_at)
			  RETURNING id, created_at, updated_at`
	stmt, err := s.db.PrepareNamedContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("SendSMSViaAPI: preparing named query: %w", err)
	}
	defer stmt.Close()

	var dbID int64
	var dbCreatedAt, dbUpdatedAt time.Time
	if err := stmt.QueryRowxContext(ctx, sms).Scan(&dbID, &dbCreatedAt, &dbUpdatedAt); err != nil {
		if strings.Contains(err.Error(), "unique constraint") &&
		   (strings.Contains(err.Error(), "sms_messages_sid_key") || strings.Contains(err.Error(), "idx_sms_messages_sid")) {
			return nil, domain.ErrConflict
		}
		return nil, fmt.Errorf("SendSMSViaAPI: executing insert for SID %s: %w", msgSID, err)
	}
	sms.ID = dbID
	sms.CreatedAt = dbCreatedAt
	sms.UpdatedAt = dbUpdatedAt
	s.log.Infof("SendSMSViaAPI: SMS %s queued for account %s", msgSID, accountSid)

	var effectiveGatewayCallbackURL string
	if statusCallbackURL != nil {
		effectiveGatewayCallbackURL = *statusCallbackURL
	} else {
		// TODO: Construct a default system callback URL if needed, e.g.,
		// effectiveGatewayCallbackURL = fmt.Sprintf("%s/api/v1/sms/gateway_status/%s", s.systemBaseURL, msgSID)
		// This requires s.systemBaseURL to be available from config.
	}

	gatewayMessageID, gwErr := s.gatewayClient.SendSMS(ctx, to, from, body, effectiveGatewayCallbackURL)
	if gwErr != nil {
		s.log.Errorf("SendSMSViaAPI: SMS %s: Gateway SendSMS failed: %v. Marking as failed.", msgSID, gwErr)
		errMsg := gwErr.Error()
		failTime := time.Now().UTC()
		_ = s.UpdateSMSStatus(ctx, msgSID, &gatewayMessageID, domain.SMSStatusFailed, nil, &errMsg, &failTime)

		sms.Status = domain.SMSStatusFailed
		sms.ErrorMessage = sql.NullString{String: errMsg, Valid: true}
		if gatewayMessageID != "" {
			sms.GatewayMessageSID = sql.NullString{String: gatewayMessageID, Valid: true}
		}
		return sms, fmt.Errorf("gateway SendSMS failed: %w", gwErr)
	}

	s.log.Infof("SendSMSViaAPI: SMS %s sent to gateway, GatewayMessageID: %s", msgSID, gatewayMessageID)
	sentTime := time.Now().UTC()
	updateErr := s.UpdateSMSStatus(ctx, msgSID, &gatewayMessageID, domain.SMSStatusSent, nil, nil, &sentTime)
	if updateErr != nil {
		s.log.Errorf("SendSMSViaAPI: SMS %s: Failed to update status to SENT: %v", msgSID, updateErr)
	}
	sms.Status = domain.SMSStatusSent
	sms.GatewayMessageSID = sql.NullString{String: gatewayMessageID, Valid: gatewayMessageID != ""}
	sms.SentAt = sql.NullTime{Time: sentTime, Valid: true}
	return sms, nil
}


func (s *smsService) UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error {
	now := time.Now().UTC()
	effectiveEventTime := now
	if eventTime != nil && !eventTime.IsZero() {
		effectiveEventTime = *eventTime
	}

	query := `UPDATE sms_messages SET status = :status, updated_at = :updated_at`
	params := map[string]interface{}{
		"sid":        agbaraSid, // For WHERE clause
		"status":     status,    // For SET clause
		"updated_at": now,
	}

	if gatewaySid != nil && *gatewaySid != "" { query += ", gateway_message_sid = :gateway_sid"; params["gateway_sid"] = *gatewaySid }
	if errorCode != nil { query += ", error_code = :error_code"; params["error_code"] = *errorCode }
	if errorMessage != nil { query += ", error_message = :error_message"; params["error_message"] = *errorMessage }

	if status == domain.SMSStatusSent && (eventTime != nil && !eventTime.IsZero()) {
		query += ", sent_at = :event_time"; params["event_time"] = effectiveEventTime
	}
	if status == domain.SMSStatusDelivered && (eventTime != nil && !eventTime.IsZero()) {
		query += ", delivered_at = :event_time"; params["event_time"] = effectiveEventTime
	}

	query += " WHERE sid = :sid"

	res, err := s.db.NamedExecContext(ctx, query, params)
	if err != nil { return fmt.Errorf("updating status for SMS %s: %w", agbaraSid, err) }
	count, _ := res.RowsAffected(); if count == 0 { return domain.ErrNotFound }
	s.log.Infof("Updated status for SMS %s to %s", agbaraSid, status)
	return nil
}

func (s *smsService) RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error) {
	now := time.Now().UTC()
	msgSID := utils.GenerateSID("SM")
	sms := &domain.SMSMessage{
		SID:          msgSID,
		AccountSID:   accountSid,
		To:           to,
		From:         from,
		Body:         body,
		Status:       domain.SMSStatusReceived,
		Direction:    domain.SMSDirectionInbound,
		GatewayMessageSID: sql.NullString{String: inboundGatewayMsgSid, Valid: inboundGatewayMsgSid != ""},
		CreatedAt:    now,
		UpdatedAt:    now,
		DeliveredAt:  sql.NullTime{Time: now, Valid: true},
	}
	query := `INSERT INTO sms_messages (sid, account_sid, msg_to, msg_from, body, status, direction, gateway_message_sid, created_at, updated_at, delivered_at)
			  VALUES (:sid, :account_sid, :msg_to, :msg_from, :body, :status, :direction, :gateway_message_sid, :created_at, :updated_at, :delivered_at)
			  RETURNING id, created_at, updated_at, delivered_at` // Match RETURNING with Scan

	stmt, err := s.db.PrepareNamedContext(ctx, query)
	if err != nil { return nil, fmt.Errorf("RecordInboundSMS: preparing named query: %w", err) }
	defer stmt.Close()

	var dbID int64
	var dbCreatedAt, dbUpdatedAt, dbDeliveredAt time.Time // Temp vars for scanning

	if err := stmt.QueryRowxContext(ctx, sms).Scan(&dbID, &dbCreatedAt, &dbUpdatedAt, &dbDeliveredAt); err != nil {
		return nil, fmt.Errorf("RecordInboundSMS: executing insert for new inbound SID %s: %w", msgSID, err)
	}
	sms.ID = dbID
	sms.CreatedAt = dbCreatedAt
	sms.UpdatedAt = dbUpdatedAt
	sms.DeliveredAt = sql.NullTime{Time: dbDeliveredAt, Valid: !dbDeliveredAt.IsZero()}

	s.log.Infof("Recorded inbound SMS %s to %s from %s (GatewaySID: %s)", msgSID, to, from, inboundGatewayMsgSid)
	return sms, nil
}
