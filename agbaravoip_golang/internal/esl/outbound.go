package esl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http" // Required for http.MethodGet/Post in handleOutboundConnection
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/interpreter"
	"github.com/sirupsen/logrus"
	"github.com/google/uuid" // For generating SIDs if needed by event handlers
)

type FSOutboundServer struct {
	cfg         config.Config
	logger      *logrus.Entry
	listener    net.Listener
	wg          sync.WaitGroup
	shutdown    chan struct{}
	xmlProcessor *callcontrol.XMLProcessor
	callService CallServicerForESL // Use the interface
}

func NewFSOutboundServer(cfg config.Config, logger *logrus.Logger, cs CallServicerForESL, xp *callcontrol.XMLProcessor) (*FSOutboundServer, error) {
	logEntry := logger.WithFields(logrus.Fields{"component": "esl_outbound_server"})
	listenAddress := cfg.Freeswitch.FSOutboundListenAddress
	if listenAddress == "" {
		listenAddress = ":8084" // Default ESL Outbound listen address
	}
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", listenAddress, err)
	}
	server := &FSOutboundServer{
		cfg:           cfg,
		logger:        logEntry,
		listener:      listener,
		shutdown:      make(chan struct{}),
		xmlProcessor:  xp,
		callService:   cs,
	}
	server.wg.Add(1)
	go server.acceptConnections()
	server.logger.Infof("ESL Outbound Server listening on %s", listener.Addr().String())
	return server, nil
}

func (s *FSOutboundServer) acceptConnections() {
	defer s.wg.Done()
	for {
		select {
		case <-s.shutdown:
			s.logger.Info("Accept loop shutting down.")
			return
		default:
			netConn, err := s.listener.Accept()
			if err != nil {
				// Check if the error is due to the listener being closed
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					s.logger.Info("Listener closed, stopping accept loop.")
					return
				}
				s.logger.Errorf("Failed to accept connection: %v", err)
				continue // Or handle more gracefully, maybe with a delay
			}
			s.logger.Infof("Accepted new ESL connection from %s", netConn.RemoteAddr().String())
			s.wg.Add(1)
			go s.handleOutboundConnection(netConn)
		}
	}
}

func (s *FSOutboundServer) handleOutboundConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()
	remoteAddr := netConn.RemoteAddr().String()
	s.logger.Infof("Handling outbound connection for %s", remoteAddr)

	rawEslConn, err := eventsocket.NewConnection(netConn, s.cfg.Freeswitch.FSPassword)
	if err != nil {
		s.logger.Errorf("ESL NewConnection error for %s: %v", remoteAddr, err)
		return
	}
	defer rawEslConn.Close()

	evConnect, err := rawEslConn.Send("connect")
	if err != nil {
		s.logger.Errorf("ESL connect command error for %s: %v", remoteAddr, err)
		return
	}

	eslExecutor := NewESLConnectionAdapter(rawEslConn)
	callCtx, err := NewCallContextFromEvent(evConnect, eslExecutor, s.logger.Logger)
	if err != nil {
		s.logger.Errorf("CallContext creation error for %s: %v", remoteAddr, err)
		fsUUID := evConnect.Get("Unique-ID")
		if fsUUID != "" {
			_, _ = eslExecutor.Hangup("CALLCTX_CREATE_ERROR")
		}
		return
	}
	callCtx.Log().Infof("CallContext created. AgbaraCallSID: %s, FS UUID: %s", callCtx.GetUuid(), callCtx.GetFreeswitchUUID())

	go s.handleEslEvents(callCtx, rawEslConn, eslExecutor)

	if _, err := eslExecutor.SendMsg(map[string]string{"command": "myevents"}); err != nil {
		callCtx.Log().Errorf("myevents command error: %v", err)
		s.finalHangup(callCtx, "ESL_SETUP_ERR", eslExecutor)
		return
	}
	if _, err := eslExecutor.SendMsg(map[string]string{"command": "linger"}); err != nil {
		callCtx.Log().Errorf("linger command error: %v", err)
		s.finalHangup(callCtx, "ESL_SETUP_ERR", eslExecutor)
		return
	}

	channelState := callCtx.GetVariable("channel_state") // Assuming NewCallContextFromEvent populates this
	if channelState != "CS_EXECUTE" && channelState != "CS_EXCHANGE_MEDIA" && channelState != "CS_ROUTING" {
		if _, errAns := eslExecutor.Answer(); errAns != nil {
			errMsg := errAns.Error()
			if !strings.Contains(strings.ToLower(errMsg), "already answered") && !strings.Contains(strings.ToLower(errMsg), "unknown command") { // "unknown command" if answered by dialplan before connect
				callCtx.Log().Errorf("Error answering call: %v", errAns)
				s.finalHangup(callCtx, "ANSWER_ERR", eslExecutor)
				return
			}
			callCtx.Log().Infof("Call already answered or answer handled by dialplan: %s", errMsg)
		} else {
			callCtx.Log().Info("Call answered successfully.")
		}
	} else {
		callCtx.Log().Infof("Call already in state %s, not sending explicit answer.", channelState)
	}

	if s.callService != nil {
		if err := s.callService.UpdateCallStatus(callCtx, string(domain.CallStatusInProgressXML), ""); err != nil {
			callCtx.Log().Errorf("Failed to update call status to InProgressXML: %v", err)
		}
	}

	var currentElements []domain.CallControlElement
	currentURL := callCtx.GetAnswerURL()
	currentMethod := http.MethodGet
	var currentParams url.Values

	if s.xmlProcessor == nil {
		callCtx.Log().Error("XMLProcessor is nil. Cannot fetch initial XML.")
		s.finalHangup(callCtx, "INTERNAL_SERVER_ERROR", eslExecutor)
		return
	}
	if currentURL == "" {
		callCtx.Log().Error("AnswerURL is empty. Cannot fetch initial XML.")
		s.finalHangup(callCtx, "NO_ANSWER_URL", eslExecutor)
		return
	}

	elements, xmlErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, currentURL, currentMethod, currentParams)
	if xmlErr != nil {
		callCtx.Log().Errorf("Initial XML fetch/parse from %s failed: %v", currentURL, xmlErr)
		s.finalHangup(callCtx, "XML_FETCH_PARSE_ERROR", eslExecutor)
		return
	}
	currentElements = elements
	callCtx.Log().Infof("Fetched & parsed initial XML (%d elements) for %s.", len(currentElements), callCtx.GetUuid())

	for {
		if callCtx.IsHangupInitiated() {
			callCtx.Log().Info("Hangup initiated, terminating XML processing loop.")
			break
		}

		if len(currentElements) == 0 {
			callCtx.Log().Info("No more XML elements to process. Waiting for async events or timeout.")
			select {
			case asyncElements, ok := <-callCtx.GetNextElementsChannel():
				if !ok {
					callCtx.Log().Info("Next elements channel closed, main loop terminating.")
					s.finalHangup(callCtx, "NORMAL_TERMINATION_NO_ASYNC_CHAN_MAINLOOP", eslExecutor)
					return
				}
				currentElements = asyncElements
				callCtx.Log().Infof("Received %d new XML elements from async event.", len(currentElements))
				// Reset URL/Method/Params as these elements came from an async callback
				currentURL = ""
				currentMethod = http.MethodPost // Typically async callbacks POST results
				currentParams = nil
				continue
			case <-time.After(60 * time.Second): // Configurable timeout
				callCtx.Log().Info("Timeout waiting for more elements or async events. Hanging up.")
				s.finalHangup(callCtx, "NO_MORE_ELEMENTS_TIMEOUT", eslExecutor)
				return
			case <-callCtx.HangupChan(): // Assuming CallContext has a channel that closes on hangup
				callCtx.Log().Info("Context hangup signal received. Terminating XML processing loop.")
				return
			}
		}

		result := interpreter.ExecuteAgbaraXML(callCtx, currentElements, eslExecutor, s.callService)
		currentElements = nil // Elements processed

		// Check for async elements that might have arrived during sync execution
		select {
		case asyncElements, ok := <-callCtx.GetNextElementsChannel():
			if !ok {
				callCtx.Log().Info("Next elements channel closed after sync execution, terminating.")
				s.finalHangup(callCtx, "NORMAL_TERMINATION_AFTER_EXEC_NO_ASYNC_CHAN", eslExecutor)
				return
			}
			callCtx.Log().Info("Received new XML elements from async event immediately after sync execution. Prioritizing async elements.")
			currentElements = asyncElements
			currentURL = ""
			currentMethod = http.MethodPost
			currentParams = nil
			continue // Process these new elements first
		default:
			// No async elements immediately, proceed with result of synchronous execution
		}

		if result.Action == domain.ActionRedirect {
			if result.RedirectURL == "" {
				callCtx.Log().Error("Redirect action with empty URL. Hanging up.")
				s.finalHangup(callCtx, "REDIRECT_URL_EMPTY", eslExecutor)
				return
			}
			callCtx.Log().Infof("Redirecting to %s %s", result.RedirectMethod, result.RedirectURL)
			currentURL = result.RedirectURL
			currentMethod = result.RedirectMethod // Use method from RedirectResult
			currentParams = nil                   // Reset params for redirect

			elements, xmlErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, currentURL, currentMethod, currentParams)
			if xmlErr != nil {
				callCtx.Log().Errorf("Redirect XML fetch/parse from %s failed: %v", currentURL, xmlErr)
				s.finalHangup(callCtx, "XML_FETCH_PARSE_ERROR_REDIRECT", eslExecutor)
				return
			}
			currentElements = elements
			callCtx.Log().Infof("Fetched & parsed redirect XML (%d elements).", len(currentElements))
			continue
		} else if result.Action == domain.ActionHangup || result.Err != nil {
			hangupReason := "NORMAL_CLEARING"
			if result.Err != nil {
				hangupReason = "ERROR_IN_EXECUTION"
				callCtx.Log().Errorf("Error during XML execution, hanging up. Error: %v", result.Err)
			}
			s.finalHangup(callCtx, hangupReason, eslExecutor)
			return
		}
		// If ActionContinue and no more elements, loop will re-evaluate len(currentElements) and wait.
	}
	callCtx.Log().Info("Main XML processing loop in handleOutboundConnection has ended.")
}

// handleEslEvents runs in a separate goroutine to process incoming ESL events for the call.
func (s *FSOutboundServer) handleEslEvents(callCtx *callcontrol.CallContext, rawEslConn *eventsocket.Connection, eslExecutor domain.EslConnectionExecutor) {
	defer callCtx.Log().Info("Exiting ESL event handler goroutine.")
	// Consider closing callCtx.nextElementsChannel here if this is the sole controller for it,
	// but only if it's certain no other goroutine will try to write to it.
	// defer close(callCtx.GetNextElementsChannel()) // This is unsafe if SendNextElements can be called elsewhere.

	reader := bufio.NewReader(rawEslConn)
	for {
		if callCtx.IsHangupInitiated() {
			callCtx.Log().Info("Main loop initiated hangup, stopping event handling.")
			return
		}

		event, errRead := eventsocket.ReadEvent(reader)
		if errRead != nil {
			if callCtx.IsHangupInitiated() {
				callCtx.Log().Infof("ESL connection closed during hangup for %s: %v", callCtx.GetUuid(), errRead)
			} else {
				callCtx.Log().Errorf("ESL connection read error for %s: %v. Signaling hangup.", callCtx.GetUuid(), errRead)
				callCtx.SetHangupInitiated()
			}
			return
		}

		eventName := event.Get("Event-Name")
		eventUUID := event.Get("Unique-ID")
		callCtx.Log().Debugf("Received ESL Event: %s for UUID: %s", eventName, eventUUID)

		switch eventName {
		case "RECORD_STOP":
			filePath := event.Get("variable_record_file_path")
			if filePath == "" { filePath = event.Get("Record-File-Path") } // Fallback for older FS versions or different event formats

			pendingRecInfoInter, exists := callCtx.GetPendingRecording()
			if exists && pendingRecInfoInter != nil {
				pendingRecInfo := pendingRecInfoInter.(*callcontrol.PendingRecordInfo)
				if pendingRecInfo.ExpectedFilePath == filePath {
					callCtx.Log().Infof("Processing RECORD_STOP for matched file: %s", filePath)
					callCtx.ClearPendingRecording()

					durationStr := event.Get("variable_record_seconds")
					if durationStr == "" { durationStr = event.Get("variable_duration_ms")} // Or variable_record_ms
					duration, _ := time.ParseDuration(durationStr + "s") // Or ms
					if durationStr == "" { // If seconds not there, try ms
						durationStrMs := event.Get("variable_record_ms")
						if durationStrMs != "" {
							duration, _ = time.ParseDuration(durationStrMs + "ms")
						}
					}

					// SizeBytes might not be available directly. Passing 0 for now.
					// recordingSid needs to be generated.
					recordingSid := fmt.Sprintf("RE%s", strings.ReplaceAll(uuid.New().String(), "-", ""))


					if s.callService != nil {
						err := s.callService.CreateRecording(callCtx, callCtx.GetCallSID(), recordingSid, filePath, uint32(duration.Seconds()), pendingRecInfo.OriginalElement.FileFormat, 0)
						if err != nil {
							callCtx.Log().Errorf("Error saving recording metadata for %s: %v", filePath, err)
						} else {
							callCtx.Log().Infof("Recording metadata saved for %s with SID %s", filePath, recordingSid)
						}
					}

					if pendingRecInfo.OriginalElement.ActionURL != "" {
						params := url.Values{}
						params.Set("RecordingSid", recordingSid)
						params.Set("RecordingDuration", fmt.Sprintf("%d", int(duration.Seconds())))
						params.Set("RecordingUrl", filePath) // This should be a public URL, not FS path
						params.Set("CallSid", callCtx.GetUuid()) // Agbara Call SID of A-leg
						params.Set("AccountSid", callCtx.GetAccountSid())

						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingRecInfo.OriginalElement.GetActionURL(), pendingRecInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && newElements != nil && len(newElements) > 0 {
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil {
								callCtx.Log().Errorf("Failed to send new elements from Record ActionURL to channel: %v", sendErr)
							}
						} else if fetchErr != nil {
							callCtx.Log().Errorf("Error fetching/parsing XML from Record ActionURL %s: %v", pendingRecInfo.OriginalElement.GetActionURL(), fetchErr)
						}
					}
				} else {
					callCtx.Log().Warnf("RECORD_STOP for file '%s' does not match pending recording path '%s'", filePath, pendingRecInfo.ExpectedFilePath)
				}
			} else {
				callCtx.Log().Warnf("Received RECORD_STOP but no pending recording info found for file: %s", filePath)
			}

		case "CHANNEL_ANSWER":
			bLegUUID := event.Get("Unique-ID")
			if bLegUUID == callCtx.GetFreeswitchUUID() {
				callCtx.Log().Info("Received CHANNEL_ANSWER for A-leg.")
				if s.callService != nil {
					_ = s.callService.UpdateCallStatus(callCtx, string(domain.CallStatusInProgress), "")
				}
			} else { // B-leg answered
				pendingDialInfoInter, exists := callCtx.GetPendingDial(bLegUUID)
				if exists && pendingDialInfoInter != nil {
					pendingDialInfo := pendingDialInfoInter.(*callcontrol.PendingDialInfo)
					callCtx.Log().Infof("B-leg %s answered for Dial to %s (Parent AgbaraCallSID: %s)",
						bLegUUID, pendingDialInfo.OriginalElement.CalleeIDToDial, pendingDialInfo.ParentAgbaraCallSID)

					// Bridge A-leg to B-leg. Using Execute, assuming it's okay to block event handler briefly.
					// For true async, eslExecutor.ExecuteAsync (if it existed and used bgapi) would be better.
					_, err := eslExecutor.Execute("uuid_bridge", callCtx.GetFreeswitchUUID(), bLegUUID)
					if err != nil {
						callCtx.Log().Errorf("Error bridging A-leg %s to B-leg %s: %v", callCtx.GetFreeswitchUUID(), bLegUUID, err)
					} else {
						callCtx.Log().Infof("Successfully bridged A-leg %s to B-leg %s", callCtx.GetFreeswitchUUID(), bLegUUID)
					}

					if pendingDialInfo.OriginalElement.ActionURL != "" {
						params := url.Values{}
						params.Set("DialCallStatus", "answered")
						params.Set("DialCallSid", bLegUUID)
						params.Set("DialBlegUuid", bLegUUID) // Alias for clarity
						params.Set("ParentCallSid", pendingDialInfo.ParentAgbaraCallSID)

						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingDialInfo.OriginalElement.GetActionURL(), pendingDialInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && newElements != nil && len(newElements) > 0 {
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil {
								callCtx.Log().Errorf("Failed to send new elements from Dial Answer ActionURL to channel: %v", sendErr)
							}
						} else if fetchErr != nil {
							callCtx.Log().Errorf("Error fetching/parsing XML from Dial Answer ActionURL %s: %v", pendingDialInfo.OriginalElement.GetActionURL(), fetchErr)
						}
					}
				} else {
					callCtx.Log().Debugf("Received CHANNEL_ANSWER for unknown B-leg UUID: %s", bLegUUID)
				}
			}
		case "CHANNEL_HANGUP":
			hangedUpUUID := event.Get("Unique-ID")
			hangupCause := event.Get("Hangup-Cause")
			if hangedUpUUID == callCtx.GetFreeswitchUUID() {
				callCtx.Log().Infof("Received CHANNEL_HANGUP for A-leg %s. Reason: %s", callCtx.GetUuid(), hangupCause)
				callCtx.SetHangupInitiated()
			} else {
				pendingDialInfoInter, exists := callCtx.GetPendingDial(hangedUpUUID)
				if exists && pendingDialInfoInter != nil {
					pendingDialInfo := pendingDialInfoInter.(*callcontrol.PendingDialInfo)
					callCtx.Log().Infof("B-leg %s hung up for Dial to %s (Parent AgbaraCallSID: %s). Cause: %s",
						hangedUpUUID, pendingDialInfo.OriginalElement.CalleeIDToDial, pendingDialInfo.ParentAgbaraCallSID, hangupCause)

					callCtx.RemovePendingDial(hangedUpUUID)

					if pendingDialInfo.OriginalElement.ActionURL != "" {
						params := url.Values{}
						// Determine DialCallStatus from hangupCause
						dialCallStatus := "completed" // Default
						if hc, ok := domain.FreeswitchHangupCauseToAgbara[hangupCause]; ok {
							if hc == domain.CallStatusBusy { dialCallStatus = "busy"}
							if hc == domain.CallStatusNoAnswer { dialCallStatus = "no-answer"}
							if hc == domain.CallStatusFailed { dialCallStatus = "failed"} // Or other failure causes
						} else if hangupCause == "NORMAL_CLEARING" && event.Get("variable_bridge_hangup_cause") != "" {
							// If it was bridged and the other leg hung up, this is completed.
						} else if hangupCause != "NORMAL_CLEARING" {
							dialCallStatus = "failed" // Generic failure for other causes
						}

						params.Set("DialCallStatus", dialCallStatus)
						params.Set("DialCallSid", hangedUpUUID)
						params.Set("DialBlegUuid", hangedUpUUID)
						params.Set("ParentCallSid", pendingDialInfo.ParentAgbaraCallSID)
						params.Set("DialHangupCause", hangupCause)

						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingDialInfo.OriginalElement.GetActionURL(), pendingDialInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && newElements != nil && len(newElements) > 0 {
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil {
								callCtx.Log().Errorf("Failed to send new elements from Dial Hangup ActionURL to channel: %v", sendErr)
							}
						} else if fetchErr != nil {
							callCtx.Log().Errorf("Error fetching/parsing XML from Dial Hangup ActionURL %s: %v", pendingDialInfo.OriginalElement.GetActionURL(), fetchErr)
						}
					}
				} else {
					callCtx.Log().Debugf("Received CHANNEL_HANGUP for unrelated/unknown UUID: %s", hangedUpUUID)
				}
			}
		case "CHANNEL_HANGUP_COMPLETE":
			if eventUUID == callCtx.GetFreeswitchUUID() {
				callCtx.Log().Infof("Received CHANNEL_HANGUP_COMPLETE for A-leg %s. Terminating event handler.", callCtx.GetUuid())
				callCtx.SetHangupInitiated()
				return
			}
		case "DTMF":
			callCtx.Log().Infof("Received DTMF: Digit='%s', Duration=%s", event.Get("DTMF-Digit"), event.Get("DTMF-Duration"))
			// TODO: Implement DTMF handling for FinishOnKey (Record/Gather)
		default:
			// Log other events if needed for debugging
			// callCtx.Log().Tracef("ESL Event: %s, Content: %s", eventName, event.String())
		}
	}
}

// finalHangup now takes EslConnectionExecutor as it cannot be reliably obtained from MinimalCallContext
func (s *FSOutboundServer) finalHangup(callCtx domain.MinimalCallContext, reason string, eslExecutor domain.EslConnectionExecutor) {
	if callCtx == nil {
		s.logger.Error("finalHangup called with nil callCtx")
		return
	}
	// Ensure eslExecutor is valid, especially if called from paths where it might not have been initialized
	if eslExecutor == nil {
		callCtx.Log().Error("finalHangup called with nil eslExecutor. Cannot send Hangup command.")
		// If callService is available, at least try to update status.
		if s.callService != nil {
			_ = s.callService.UpdateCallStatus(callCtx, string(domain.CallStatusFailed), "ESL_EXECUTOR_NIL_ON_HANGUP")
		}
		callCtx.SetHangupInitiated() // Ensure other loops/goroutines know to stop
		return
	}

	if callCtx.IsHangupInitiated() && reason != "CHANNEL_UNBRIDGE_HANGUP" { // Allow specific reasons for specific cases if needed
		callCtx.Log().Infof("Hangup already initiated or in progress for %s, new reason %s - not sending another hangup command.", callCtx.GetUuid(), reason)
		return
	}
	callCtx.SetHangupInitiated()

	callCtx.Log().Infof("Final hangup for %s, reason: %s", callCtx.GetUuid(), reason)
	_, err := eslExecutor.Hangup(reason)
	if err != nil {
		callCtx.Log().Errorf("Error sending hangup command for %s: %v", callCtx.GetUuid(), err)
	}

	if s.callService != nil {
		// Determine final status based on reason more accurately if possible
		finalStatus := domain.CallStatusCompleted
		if reason != "NORMAL_CLEARING" && reason != "NORMAL_TEMPORARY_FAILURE" { // Add more failure reasons
			// Many FS hangup causes might map to "failed" or specific Agbara statuses
			if _, ok := domain.FreeswitchHangupCauseToAgbara[reason]; ok {
				// Use mapped status if available, otherwise default to completed/failed
			} else {
				finalStatus = domain.CallStatusFailed // Default for unmapped error reasons
			}
		}
		err := s.callService.UpdateCallStatus(callCtx, string(finalStatus), reason)
		if err != nil {
			callCtx.Log().Errorf("Failed to update final call status for %s: %v", callCtx.GetUuid(), err)
		}
	}
}

func (s *FSOutboundServer) Shutdown() {
	s.logger.Info("ESL Outbound Server shutting down...")
	close(s.shutdown)
	if s.listener != nil {
		s.listener.Close()
	}
	s.wg.Wait()
	s.logger.Info("ESL Outbound Server shut down.")
}

// ESLConnectionAdapter implements domain.EslConnectionExecutor using *eventsocket.Connection
type ESLConnectionAdapter struct {
	conn *eventsocket.Connection
	log  *logrus.Entry
}

func NewESLConnectionAdapter(eslConn *eventsocket.Connection) domain.EslConnectionExecutor {
	adapterLogger := logrus.New().WithField("adapter", "ESLConnectionAdapter")
	adapterLogger.Logger.SetOutput(io.Discard) // Quiet by default, or pass parent logger
	return &ESLConnectionAdapter{conn: eslConn, log: adapterLogger}
}

func (a *ESLConnectionAdapter) Execute(command string, args ...string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter Execute")}
	argStr := strings.Join(args, " ")
	a.log.Debugf("Adapter Executing: App='%s', Args='%s'", command, argStr)
	ev, err := a.conn.Execute(command, argStr, true)
	if err != nil { return "", err }
	if ev == nil { return "", errors.New("nil event received from ESL Execute") }
	// Return Reply-Text or Body, depending on command. For apps, Reply-Text is usually the status.
	return ev.Get("Reply-Text"), nil
}

func (a *ESLConnectionAdapter) ExecuteSofia(command string, args ...string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter ExecuteSofia")}
	// Example: sofia status profile internal
	fullCmd := fmt.Sprintf("sofia %s %s", command, strings.Join(args, " "))
	a.log.Debugf("Adapter ExecuteSofia (as API): %s", fullCmd)
	ev, err := a.conn.Send(fmt.Sprintf("api %s", fullCmd))
	if err != nil { return "", err}
	if ev == nil { return "", errors.New("nil event received")}
	return ev.Body, nil // Sofia commands often return body
}

func (a *ESLConnectionAdapter) SendMsg(msg map[string]string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter SendMsg")}
	if cmd, ok := msg["command"]; ok {
		// This is for sending generic ESL commands like "myevents", "linger"
		// For API commands, use ExecuteSofia or a dedicated API method if interface expands.
		fullCmd := cmd
		if cmdArgs, hasArgs := msg["args"]; hasArgs && cmdArgs != "" {
			fullCmd = fmt.Sprintf("%s %s", cmd, cmdArgs)
		}
		a.log.Debugf("Adapter SendMsg (as Send): %s", fullCmd)
		ev, err := a.conn.Send(fullCmd)
		if err != nil { return "", err}
		if ev == nil { return "", errors.New("nil event received")}
		return ev.String(), nil
	}
	return "", errors.New("SendMsg: 'command' key missing in message map")
}

func (a *ESLConnectionAdapter) GetVar(varName string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter GetVar")}
	a.log.Debugf("Adapter GetVar: %s", varName)
	return a.conn.GetVar(varName)
}

func (a *ESLConnectionAdapter) Answer() (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter Answer")}
	a.log.Debug("Adapter Answer called")
	ev, err := a.conn.Execute("answer", "", true)
	if err != nil { return "", err}
	if ev == nil { return "", errors.New("nil event received from Answer")}
	replyText := ev.Get("Reply-Text")
	if strings.Contains(replyText, "-ERR Already Answered") || strings.Contains(replyText, "UNKNOWN COMMAND") { // FS might return UNKNOWN if dialplan answered.
		return replyText, errors.New(replyText)
	}
	if !strings.HasPrefix(replyText, "+OK") {
		return replyText, fmt.Errorf("answer command failed: %s", replyText)
	}
	return replyText, nil
}

func (a *ESLConnectionAdapter) Hangup(reason string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter Hangup")}
	a.log.Debugf("Adapter Hangup: %s", reason)
	ev, err := a.conn.Execute("hangup", reason, true)
	if err != nil { return "", err}
	if ev == nil { return "", errors.New("nil event received from Hangup")}
	return ev.Get("Reply-Text"), nil
}

func (a *ESLConnectionAdapter) RecordSession(filePath string, maxDurationSec uint32, silenceThreshold uint, silenceHits uint) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter RecordSession")}
	cmdArgs := []string{filePath}
	if maxDurationSec > 0 {
		cmdArgs = append(cmdArgs, fmt.Sprintf("%d", maxDurationSec))
		// Only add silence params if limit is also set, as per typical FS app behavior
		if silenceThreshold > 0 {
			cmdArgs = append(cmdArgs, fmt.Sprintf("%d", silenceThreshold))
			if silenceHits > 0 {
				cmdArgs = append(cmdArgs, fmt.Sprintf("%d", silenceHits))
			}
		}
	}
	a.log.Debugf("Adapter RecordSession: Path=%s, Args=%v", filePath, cmdArgs)
	return a.Execute("record_session", cmdArgs...)
}

func (a *ESLConnectionAdapter) Originate(dialString string, vars map[string]string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter Originate")}
	var varList []string
	for k, v := range vars {
		// Escape commas and curly braces for FS originate variable string
		escapedValue := strings.ReplaceAll(v, ",", "\\,")
		escapedValue = strings.ReplaceAll(escapedValue, "{", "\\{")
		escapedValue = strings.ReplaceAll(escapedValue, "}", "\\}")
		varList = append(varList, fmt.Sprintf("%s='%s'", k, escapedValue)) // Use single quotes for values
	}
	varsString := ""
	if len(varList) > 0 { varsString = "{" + strings.Join(varList, ",") + "}" }

	// Using 'bgapi originate' for non-blocking originate. The response will be Job-UUID.
	// The actual new channel UUID will come in a subsequent CHANNEL_CREATE event.
	// The caller (DialElement.Execute) needs to be aware of this or this adapter needs more logic (event listening).
	// For now, returning Job-UUID as per prompt's note on this being a complex area.
	originateCmd := fmt.Sprintf("bgapi originate %s%s", varsString, dialString) // Simplified: assumes dialString includes target app e.g. &socket()
	a.log.Debugf("Adapter Originate (via bgapi): %s", originateCmd)

	ev, err := a.conn.Send(originateCmd) // Send is for generic commands
	if err != nil { return "", err }
	if ev == nil { return "", errors.New("nil event from bgapi originate")}

	// bgapi commands usually return Job-UUID in the body or a specific header
	jobUUID := ev.Get("Job-UUID")
	if jobUUID == "" {
		// Check body if not in header
		if strings.HasPrefix(ev.Body, "+OK Job-UUID: ") {
			jobUUID = strings.TrimPrefix(ev.Body, "+OK Job-UUID: ")
		} else if strings.HasPrefix(ev.Body, "+OK") { // Some simple OK might not have Job-UUID if command is very simple
            return "OK_NO_JOB_UUID", nil // Or an empty string with nil error if that's acceptable
        } else {
			return "", fmt.Errorf("no Job-UUID from bgapi originate, response: %s", ev.Body)
		}
	}
	a.log.Infof("Originate job started with Job-UUID: %s", jobUUID)
	return jobUUID, nil
}

// NewCallContextFromEvent helper
func NewCallContextFromEvent(ev *eventsocket.Event, eslExecutor domain.EslConnectionExecutor, baseLogger *logrus.Logger) (*callcontrol.CallContext, error) {
	if ev == nil { return nil, errors.New("connect event is nil") }
	fsUUID := ev.Get("Unique-ID") // This is the Freeswitch Channel UUID for the A-leg
	if fsUUID == "" { return nil, errors.New("Unique-ID not found in connect event") }

	agbaraCallSid := ev.Get("variable_agbara_call_sid")
	if agbaraCallSid == "" {
		agbaraCallSid = fsUUID // Default Agbara Call SID to FS UUID if not provided
		baseLogger.Warnf("variable_agbara_call_sid not found, defaulting to Freeswitch Unique-ID: %s", fsUUID)
	}

	accountSid := ev.Get("variable_agbara_account_sid")
	if accountSid == "" { accountSid = ev.Get("variable_account_sid") }
	if accountSid == "" {
		baseLogger.Warn("No account SID found in channel variables (agbara_account_sid or account_sid). Using 'unknown_account'.")
		accountSid = "unknown_account"
	}

	appSid := ev.Get("variable_agbara_application_sid")
	if appSid == "" { appSid = ev.Get("variable_application_sid")}
	// appSid can be empty if not specified for the call.

	answerURL := ev.Get("variable_agbara_answer_url")
	if answerURL == "" {
		baseLogger.Error("variable_agbara_answer_url not found in connect event. Cannot proceed.")
		return nil, errors.New("variable_agbara_answer_url not found")
	}

	vars := make(map[string]string)
	for key, values := range ev.Header {
		if strings.HasPrefix(key, "Variable-") {
			varName := strings.TrimPrefix(key, "Variable-")
			decodedValue, err := url.QueryUnescape(values[0])
			if err != nil {
				baseLogger.Warnf("Failed to URL decode variable %s: %v. Using raw value: %s", varName, err, values[0])
				vars[varName] = values[0]
			} else {
				vars[varName] = decodedValue
			}
		}
	}
	// Ensure essential variables always present in the map for CallContext
	vars["uuid"] = fsUUID // Freeswitch Channel UUID
	vars["agbara_call_sid"] = agbaraCallSid // Our application's Call SID
	vars["channel_state"] = ev.Get("Channel-State")
	vars["channel_call_state"] = ev.Get("Channel-Call-State")
	vars["caller_id_number"] = ev.Get("Caller-Caller-ID-Number")
	vars["destination_number"] = ev.Get("Caller-Destination-Number")
	vars["agbara_account_sid"] = accountSid
	vars["agbara_application_sid"] = appSid
	vars["agbara_answer_url"] = answerURL


	// Pass agbaraCallSid as the primary SID for CallContext, and fsUUID as a specific variable.
	return callcontrol.NewCallContext(agbaraCallSid, fsUUID, accountSid, appSid, answerURL, vars, eslExecutor, baseLogger), nil
}

// MinimalCallContextTyped is no longer needed as handleEslEvents will use *callcontrol.CallContext
// type MinimalCallContextTyped interface {
// 	domain.MinimalCallContext
// 	ESLConnection() domain.EslConnectionExecutor // Example of a method not on MinimalCallContext
//  HangupChan() <-chan struct{} // Example, if CallContext had this
// }

// Helper to generate a unique recording SID
func GenerateRecordingSID() string {
    // Example: RE + base36 of nanoseconds or a proper UUID
    // Using a simpler version for now. Replace with robust SID generator.
    return fmt.Sprintf("RE%d", time.Now().UnixNano())
}

// Helper for logging nullable strings, can be moved to a common utils package
func LogNullString(ns *string) string {
    if ns == nil {
        return "<nil>"
    }
    return *ns
}
