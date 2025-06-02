package esl

import (
	"fmt"
	"net"
	"strings"
	"sync"
	// "bufio" // No longer needed for simplified handler

	// "github.com/fiorix/go-eventsocket/eventsocket" // Temporarily remove direct usage in handler if problematic
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/sirupsen/logrus"
)

// FSOutboundServer listens for and handles outbound ESL connections from Freeswitch.
type FSOutboundServer struct {
	cfg      config.Config // Full config
	logger   *logrus.Entry
	listener net.Listener
	wg       sync.WaitGroup
	shutdown chan struct{}
}

// NewFSOutboundServer creates and starts a new ESL outbound server.
func NewFSOutboundServer(cfg config.Config, logger *logrus.Logger) (*FSOutboundServer, error) {
	logEntry := logger.WithFields(logrus.Fields{"component": "esl_outbound_server"})
	listenAddress := cfg.Freeswitch.FSOutboundListenAddress
	if listenAddress == "" {
		listenAddress = ":8084" 
		logEntry.Warnf("FSOutboundListenAddress not set in config, defaulting to %s", listenAddress)
	}

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		logEntry.Errorf("Error starting ESL outbound listener on %s: %v", listenAddress, err)
		return nil, fmt.Errorf("failed to start ESL outbound listener: %w", err)
	}

	server := &FSOutboundServer{
		cfg:      cfg, 
		logger:   logEntry,
		listener: listener,
		shutdown: make(chan struct{}),
	}

	server.wg.Add(1)
	go server.acceptConnections()

	server.logger.Infof("ESL Outbound Server started, listening on %s", listener.Addr().String())
	return server, nil
}

func (s *FSOutboundServer) acceptConnections() {
	defer s.wg.Done()
	for {
		select {
		case <-s.shutdown:
			s.logger.Info("ESL Outbound Server shutting down acceptor loop.")
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					s.logger.Info("ESL Outbound listener closed.")
					return
				}
				s.logger.Errorf("Failed to accept ESL outbound connection: %v", err)
				continue 
			}
			
			s.wg.Add(1)
			go s.handleOutboundConnection(conn)
		}
	}
}

// handleOutboundConnection - Simplified: Logs connection and closes.
// Full ESL parsing with fiorix/go-eventsocket for server-side needs re-evaluation or alternative.
func (s *FSOutboundServer) handleOutboundConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()

	s.logger.Infof("Accepted and handling simplified new ESL outbound connection from: %s", netConn.RemoteAddr().String())

	// TODO: Revisit full ESL parsing for outbound connections.
	// The fiorix/go-eventsocket library is primarily a client library.
	// Using it to wrap a server-accepted connection (netConn) requires ensuring
	// its API supports this mode correctly (e.g., an exported NewConnectionFromSocket or similar).
	// The previous `eventsocket.NewConnection(netConn, s.cfg.Freeswitch.FSPassword)` failed.
	// For now, we demonstrate the server accepting the connection.

	s.logger.Infof("ESL Connection object creation/wrapping for %s deferred. Basic connection accepted.", netConn.RemoteAddr().String())
	s.logger.Infof("Finished handling (simplified) outbound call from %s", netConn.RemoteAddr().String())
}

// Shutdown gracefully stops the ESL outbound server.
func (s *FSOutboundServer) Shutdown() {
	s.logger.Info("ESL Outbound Server initiating shutdown...")
	close(s.shutdown) 
	if s.listener != nil {
		s.listener.Close() 
	}
	s.wg.Wait() 
	s.logger.Info("ESL Outbound Server shut down gracefully.")
}
