package esl

import (
	"fmt"
	// "time" // Removed as it's not currently used

	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/logging"
)

// FSInboundClient represents an inbound Freeswitch ESL client.
type FSInboundClient struct {
	conn *eventsocket.Connection
	cfg  config.FreeswitchConfig
}

// NewFSInboundClient creates and connects a new inbound Freeswitch ESL client.
// It's a simplified example; production code would need robust reconnection logic
// and potentially custom dialer for fine-grained timeout control.
func NewFSInboundClient(cfg config.FreeswitchConfig) (*FSInboundClient, error) {
	address := fmt.Sprintf("%s:%s", cfg.FSAddress, cfg.FSPort)
	logging.Logger.Infof("Attempting to connect to Freeswitch (inbound) at %s", address)

	c, err := eventsocket.Dial(address, cfg.FSPassword)
	if err != nil {
		logging.Logger.Errorf("Failed to connect to Freeswitch at %s: %v", address, err)
		return nil, fmt.Errorf("esl.NewFSInboundClient: failed to dial Freeswitch: %w", err)
	}
	logging.Logger.Infof("Successfully connected to Freeswitch (inbound) at %s", address)

	return &FSInboundClient{conn: c, cfg: cfg}, nil
}

// SendCommand sends a generic API command to Freeswitch and returns the response.
func (fc *FSInboundClient) SendCommand(command string) (string, error) {
	if fc.conn == nil {
		return "", fmt.Errorf("esl.SendCommand: not connected to Freeswitch")
	}
	logging.Logger.Debugf("Sending Inbound ESL command: %s", command)

	ev, err := fc.conn.Send(command)
	if err != nil {
		logging.Logger.Errorf("Failed to send command '%s': %v", command, err)
		return "", fmt.Errorf("esl.SendCommand: failed to send command: %w", err)
	}

	replyText := ev.Get("Reply-Text")
	if replyText == "" {
		replyText = "No Reply-Text header"
	}
	logging.Logger.Debugf("Received reply for '%s': %s, Body: %s", command, replyText, ev.Body)

	return ev.Body, nil
}

// GetFSStatus is a convenience function to send "api status".
func (fc *FSInboundClient) GetFSStatus() (string, error) {
	return fc.SendCommand("api status")
}

// Close closes the connection to Freeswitch.
func (fc *FSInboundClient) Close() {
	if fc.conn != nil {
		logging.Logger.Info("Closing Freeswitch inbound connection.")
		fc.conn.Close()
	}
}
