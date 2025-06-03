package services

import (
	"context"
	"time"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/stretchr/testify/mock" // For MockSmsGatewayClient
)

// SMSGatewayClient defines the interface for an external SMS gateway client.
type SMSGatewayClient interface {
	SendSMS(ctx context.Context, to, from, body string, statusCallbackURL string) (gatewayMessageSID string, err error)
}

// MockSmsGatewayClient is a mock implementation for SMSGatewayClient.
type MockSmsGatewayClient struct {
	mock.Mock
}

func (m *MockSmsGatewayClient) SendSMS(ctx context.Context, to, from, body string, statusCallbackURL string) (string, error) {
	args := m.Called(ctx, to, from, body, statusCallbackURL)
	return args.String(0), args.Error(1)
}


type SMSService interface {
	// SendSMS initiates an outbound SMS.
	// msgSID is the pre-generated Agbara SID for this message.
	// actionURL & actionMethod are for callbacks upon final status update from the gateway
	// (though currently not directly stored or used by SMSService for its own callbacks,
	// they are passed to the domain.SmsElement which might use them).
	SendSMS(ctx context.Context, accountSid, to, from, body, msgSID, actionURL, actionMethod string) (*domain.SMSMessage, error)

	GetSMSBySID(ctx context.Context, sid string) (*domain.SMSMessage, error)

	// UpdateSMSStatus is typically called by a webhook handler receiving updates from the SMS gateway.
	UpdateSMSStatus(ctx context.Context, agbaraSid string, gatewaySid *string, status domain.SMSStatus, errorCode *int32, errorMessage *string, eventTime *time.Time) error

	// RecordInboundSMS records an SMS received from the gateway.
	// inboundGatewayMsgSid is the SID assigned by the gateway for this message.
	RecordInboundSMS(ctx context.Context, accountSid, to, from, body, inboundGatewayMsgSid string) (*domain.SMSMessage, error)
}
