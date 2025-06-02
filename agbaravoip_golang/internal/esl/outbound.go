package esl
import ( "bufio"; "fmt"; "net"; "strings"; "sync"; "time";
	"github.com/fiorix/go-eventsocket/eventsocket"; "github.com/user/agbaravoip_golang/internal/config";
	"github.com/user/agbaravoip_golang/internal/callcontrol"; "github.com/user/agbaravoip_golang/internal/domain";
	"github.com/sirupsen/logrus" )
type FSOutboundServer struct {
	cfg config.Config; logger *logrus.Entry; listener net.Listener; wg sync.WaitGroup;
	shutdown chan struct{}; xmlProcessor *callcontrol.XMLProcessor; callService CallServicerForESL // Use the interface
}
func NewFSOutboundServer(cfg config.Config, logger *logrus.Logger, cs CallServicerForESL, xp *callcontrol.XMLProcessor) (*FSOutboundServer, error) { // Changed cs type
	logEntry := logger.WithFields(logrus.Fields{"component": "esl_outbound_server"})
	listenAddress := cfg.Freeswitch.FSOutboundListenAddress; if listenAddress == "" { listenAddress = ":8084" }
	listener, err := net.Listen("tcp", listenAddress); if err != nil { return nil, err }
	server := &FSOutboundServer{ cfg: cfg, logger: logEntry, listener: listener, shutdown: make(chan struct{}), xmlProcessor: xp, callService: cs }
	server.wg.Add(1); go server.acceptConnections()
	server.logger.Infof("ESL Outbound Server listening on %s", listener.Addr().String()); return server, nil
}
func (s *FSOutboundServer) acceptConnections() { defer s.wg.Done(); for { select { case <-s.shutdown: return
		default: conn, err := s.listener.Accept(); if err != nil { if strings.Contains(err.Error(), "closed") { return }; continue }; s.wg.Add(1); go s.handleOutboundConnection(conn) } } }
func (s *FSOutboundServer) handleOutboundConnection(netConn net.Conn) {
	defer s.wg.Done(); defer netConn.Close(); remoteAddr := netConn.RemoteAddr().String()
	eslConn, err := eventsocket.NewConnection(netConn, s.cfg.Freeswitch.FSPassword); if err != nil { s.logger.Errorf("ESL conn err for %s: %v", remoteAddr, err); return }
	evConnect, err := eslConn.Send("connect"); if err != nil { s.logger.Errorf("connect err for %s: %v", remoteAddr, err); return }
	callCtx, err := callcontrol.NewCallContext(evConnect, eslConn, s.logger.Logger); if err != nil { s.logger.Errorf("CallCtx err for %s: %v", remoteAddr, err); fsUUID := evConnect.Get("Unique-ID"); if fsUUID != "" { _, _ = eslConn.Execute("hangup", fsUUID, true) }; return }
	callCtx.Logger.Infof("CallContext created for %s.", callCtx.AgbaraCallSID)
	if _, err := eslConn.Send("myevents"); err != nil { callCtx.Logger.Errorf("myevents err: %v", err); s.finalHangup(callCtx, "ESL_SETUP_ERR"); return }
	if _, err := eslConn.Send("linger"); err != nil { callCtx.Logger.Errorf("linger err: %v", err); s.finalHangup(callCtx, "ESL_SETUP_ERR"); return }
	if callCtx.ChannelState != "CS_EXECUTE" && callCtx.ChannelState != "CS_EXCHANGE_MEDIA" && callCtx.ChannelState != "CS_HANGUP" {
		_, errAns := eslConn.Execute("answer", "", true); if errAns != nil { callCtx.Logger.Errorf("answer err: %v", errAns); s.finalHangup(callCtx, "ANSWER_ERR"); return }
		if s.callService != nil && callCtx.AgbaraCallSID != "" { _, _ = s.callService.UpdateCallStatus(callCtx.AgbaraCallSID, domain.CallStatusInProgress, "", 0) }
	}
	callCtx.Logger.Infof("ESL session for %s. XML from %s", callCtx.AgbaraCallSID, callCtx.AnswerURL)
	if s.xmlProcessor != nil && callCtx.AnswerURL != "" {
		_, xmlErr := s.xmlProcessor.FetchAndParseXML(callCtx) // Simplified return
		if xmlErr != nil { callCtx.Logger.Errorf("XML fetch/parse for %s failed: %v", callCtx.AgbaraCallSID, xmlErr)
		} else { callCtx.Logger.Info("Fetched & validated XML for %s.", callCtx.AgbaraCallSID) }
	} else { callCtx.Logger.Warnf("XMLProcessor nil or no AnswerURL for %s.", callCtx.AgbaraCallSID) }
	s.finalHangup(callCtx, "NORMAL_CLEARING_PLACEHOLDER")
	reader := bufio.NewReader(eslConn); for { event, errRead := eventsocket.ReadEvent(reader); if errRead != nil { callCtx.Logger.Infof("ESL conn closed for %s: %v", callCtx.AgbaraCallSID, errRead); return }
		if event.Get("Event-Name") == "CHANNEL_HANGUP_COMPLETE" { callCtx.Logger.Info("CHH_COMPLETE for %s.", callCtx.AgbaraCallSID); return } }
}
func (s *FSOutboundServer) finalHangup(callCtx *callcontrol.CallContext, reason string) { if callCtx == nil || callCtx.ESLConnection == nil { return }; callCtx.Logger.Infof("Final hangup for %s, reason: %s", callCtx.AgbaraCallSID, reason); _, _ = callCtx.ESLConnection.Execute("hangup", reason, true); if s.callService != nil && callCtx.AgbaraCallSID != "" { _, _ = s.callService.UpdateCallStatus(callCtx.AgbaraCallSID, domain.CallStatusCompleted, reason, 0) } }
func (s *FSOutboundServer) Shutdown() { s.logger.Info("ESL Outbound Server shutting down..."); close(s.shutdown); if s.listener != nil { s.listener.Close() }; s.wg.Wait(); s.logger.Info("ESL Outbound Server shut down.") }

