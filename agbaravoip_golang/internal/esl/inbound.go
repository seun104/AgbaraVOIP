package esl 
import ( "fmt"; "strings"; "github.com/fiorix/go-eventsocket/eventsocket"; "github.com/user/agbaravoip_golang/internal/config"; "github.com/sirupsen/logrus")
type FSInboundClient struct { conn *eventsocket.Connection; logger *logrus.Entry; cfg config.Config }
func NewFSInboundClient(appCfg config.Config, logger *logrus.Logger) (*FSInboundClient, error) {
	logEntry := logger.WithField("component", "esl_inbound_client")
	fsCfg := appCfg.Freeswitch
	if fsCfg.FSAddress == "" || fsCfg.FSPort == "" || fsCfg.FSPassword == "" { return nil, fmt.Errorf("FS params not configured") }
	serverAddr := fmt.Sprintf("%s:%s", fsCfg.FSAddress, fsCfg.FSPort)
	logEntry.Infof("Connecting to FS ESL at %s", serverAddr)
	conn, err := eventsocket.Dial(serverAddr, fsCfg.FSPassword)
	if err != nil { logEntry.Errorf("Failed to connect to FS ESL at %s: %v", serverAddr, err); return nil, fmt.Errorf("esl dial failed: %w", err) }
	logEntry.Info("Connected to FS ESL (Inbound).")
	return &FSInboundClient{conn: conn, logger: logEntry, cfg: appCfg}, nil
}
func (c *FSInboundClient) SendCommand(cmd string) (string, error) {
	if c.conn == nil { return "", fmt.Errorf("not connected to FS ESL") }
	c.logger.Debugf("Sending ESL Command: %s", cmd)
	ev, err := c.conn.Send(cmd); if err != nil { c.logger.Errorf("ESL Command failed: %s - %v", cmd, err); return "", err }
	c.logger.Debugf("ESL Command Reply: H: %+v, B: %s", ev.Header, ev.Body) 
	return ev.Body, nil
}
func (c *FSInboundClient) GetFSStatus() (string, error) { return c.SendCommand("api status") }
func (c *FSInboundClient) GetOutboundServerListenAddress() string { 
    addr := c.cfg.Freeswitch.FSOutboundListenAddress
    if addr == "" { addr = ":8084" } // Default if empty
	if strings.HasPrefix(addr, ":") {
		// This needs to be a host reachable by Freeswitch. "localhost" or "127.0.0.1" might work for local dev.
		// For Docker, this should be the Go app service name, e.g., "app:8084".
		// This logic is simplified and may need adjustment for specific environments.
		c.logger.Warnf("FSOutboundListenAddress (%s) starts with :, FS might need explicit host for Docker.", addr)
		// Example: return "app" + addr // if app is the service name in docker-compose
		return "127.0.0.1" + addr // Fallback for local development if only port is given
	}
    return addr
}
func (c *FSInboundClient) Close() { if c.conn != nil { c.logger.Info("Closing FS ESL Inbound conn."); c.conn.Close() } }

