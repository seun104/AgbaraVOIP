package esl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io" // Required by NewMockMinimalCallContext for tests, but not directly here. Keep for consistency if tests are in same package.
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/config"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/interpreter"
	"github.com/user/agbaravoip_golang/internal/utils" // For GenerateSID in event handlers
)

type FSOutboundServer struct {
	cfg         config.Config
	logger      *logrus.Entry
	listener    net.Listener
	wg          sync.WaitGroup
	shutdown    chan struct{}
	xmlProcessor *callcontrol.XMLProcessor
	callService domain.CallServicerForESL // Use the interface from domain package
}

// NewFSOutboundServer constructor
func NewFSOutboundServer(cfg config.Config, logger *logrus.Logger, cs domain.CallServicerForESL, xp *callcontrol.XMLProcessor) (*FSOutboundServer, error) {
	logEntry := logger.WithFields(logrus.Fields{"component": "esl_outbound_server"})
	listenAddress := cfg.Freeswitch.FSOutboundListenAddress
	if listenAddress == "" {
		listenAddress = ":8084"
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
				if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
					s.logger.Info("Listener closed, stopping accept loop.")
					return
				}
				s.logger.Errorf("Failed to accept connection: %v", err)
				continue
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

	eslExecutor := NewESLConnectionAdapter(rawEslConn, s.logger.WithField("adapter", "ESLConnectionAdapter")) // Pass logger to adapter
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
	if _, err := eslExecutor.SendMsg(map[string]string{"command": "event json CONFERENCE_MAINTENANCE ALL"}); err != nil {
	callCtx.Log().Warnf("Failed to subscribe to CONFERENCE_MAINTENANCE events: %v. Conference callbacks might not work.", err)
	}


	channelState := callCtx.GetVariable("channel_state")
	if channelState != "CS_EXECUTE" && channelState != "CS_EXCHANGE_MEDIA" && channelState != "CS_ROUTING" {
		if _, errAns := eslExecutor.Answer(); errAns != nil {
			errMsg := errAns.Error()
			if !strings.Contains(strings.ToLower(errMsg), "already answered") && !strings.Contains(strings.ToLower(errMsg), "unknown command") {
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
				currentURL = ""
				currentMethod = http.MethodPost
				currentParams = nil
				continue
			case <-callCtx.HangupChan(): // Listen to hangup signal from context
				callCtx.Log().Info("Context hangup signal received in main loop. Terminating.")
				// s.finalHangup(callCtx, "CONTEXT_HANGUP_SIGNAL", eslExecutor) // Already handled by IsHangupInitiated check
				return
			case <-time.After(60 * time.Second):
				callCtx.Log().Info("Timeout waiting for more elements or async events. Hanging up.")
				s.finalHangup(callCtx, "NO_MORE_ELEMENTS_TIMEOUT", eslExecutor)
				return
			}
		}

		result := interpreter.ExecuteAgbaraXML(callCtx, currentElements, eslExecutor, s.callService)
		currentElements = nil

		processExecutionResult := true
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
			processExecutionResult = false // Skip processing result of the just-executed sync elements
		default:
			// No async elements immediately available
		}

		if processExecutionResult {
			if result.Action == domain.ActionRedirect {
				if result.RedirectURL == "" {
					callCtx.Log().Error("Redirect action with empty URL. Hanging up.")
					s.finalHangup(callCtx, "REDIRECT_URL_EMPTY", eslExecutor)
					return
				}
				callCtx.Log().Infof("Redirecting to %s %s", result.RedirectMethod, result.RedirectURL)
				currentURL = result.RedirectURL
				currentMethod = result.RedirectMethod
				currentParams = nil

				elements, xmlErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, currentURL, currentMethod, currentParams)
				if xmlErr != nil {
					callCtx.Log().Errorf("Redirect XML fetch/parse from %s failed: %v", currentURL, xmlErr)
					s.finalHangup(callCtx, "XML_FETCH_PARSE_ERROR_REDIRECT", eslExecutor)
					return
				}
				currentElements = elements
				callCtx.Log().Infof("Fetched & parsed redirect XML (%d elements).", len(currentElements))
				continue
			} else if result.Action == domain.ActionHangup || (result.Err != nil && result.Action != domain.ActionContinue) { // Hangup on error unless it's explicitly continue
				hangupReason := "NORMAL_CLEARING"
				if result.Err != nil {
					hangupReason = "ERROR_IN_EXECUTION"
					callCtx.Log().Errorf("Error during XML execution, hanging up. Error: %v", result.Err)
				}
				s.finalHangup(callCtx, hangupReason, eslExecutor)
				return
			}
		}
	}
	callCtx.Log().Info("Main XML processing loop in handleOutboundConnection has ended.")
}

// Simplified local map for Dial status from Hangup Cause
var localFreeswitchHangupCauseToDialStatus = map[string]string{
	"NORMAL_CLEARING":      "completed",
	"USER_BUSY":            "busy",
	"NO_ANSWER":            "no-answer",
	"CALL_REJECTED":        "failed",
	"UNALLOCATED_NUMBER":   "failed",
	"NORMAL_TEMPORARY_FAILURE": "failed",
	"NO_ROUTE_DESTINATION": "failed",
	"ORIGINATOR_CANCEL":    "canceled",
}


func (s *FSOutboundServer) handleEslEvents(callCtx *callcontrol.CallContext, rawEslConn *eventsocket.Connection, eslExecutor domain.EslConnectionExecutor) {
	defer callCtx.Log().Info("Exiting ESL event handler goroutine.")
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
		eventUUID := event.Get("Unique-ID") // FS UUID of the channel the event pertains to
		callCtx.Log().Debugf("Received ESL Event: %s for UUID: %s", eventName, eventUUID)

		switch eventName {
		case "RECORD_STOP":
			filePath := event.Get("variable_record_file_path")
			if filePath == "" { filePath = event.Get("Record-File-Path") }

			pendingRecInfoInter, exists := callCtx.GetPendingRecording()
			if exists && pendingRecInfoInter != nil {
				pendingRecInfo := pendingRecInfoInter.(*callcontrol.PendingRecordInfo)
				if pendingRecInfo.ExpectedFilePath == filePath {
					callCtx.Log().Infof("Processing RECORD_STOP for matched file: %s", filePath)
					callCtx.ClearPendingRecording()

					durationStr := event.Get("variable_record_seconds")
					if durationStr == "" { durationStr = event.Get("variable_record_ms") }

					var durationSec uint32
					if strings.HasSuffix(durationStr, "ms") {
						msDuration, _ := time.ParseDuration(durationStr)
						durationSec = uint32(msDuration.Seconds())
					} else if durationStr != "" {
						secDuration, _ := time.ParseDuration(durationStr + "s")
						durationSec = uint32(secDuration.Seconds())
					}

					sizeBytes := int64(0) // Default, FS might not provide easily
					if sizeStr := event.Get("variable_record_megabytes"); sizeStr != "" {
						var megaBytes float64; fmt.Sscanf(sizeStr, "%f", &megaBytes)
						sizeBytes = int64(megaBytes * 1024 * 1024)
					} else if samplesStr := event.Get("variable_record_samples"); samplesStr != "" {
						var samples int64; fmt.Sscanf(samplesStr, "%d", &samples)
						sizeBytes = samples * 2 // Assuming 16-bit mono
					}

					recordingSid := utils.GenerateSID("RE")

					if s.callService != nil {
						err := s.callService.CreateRecording(callCtx, callCtx.GetCallSID(), recordingSid, filePath, durationSec, pendingRecInfo.OriginalElement.FileFormat, sizeBytes)
						if err != nil { callCtx.Log().Errorf("Error saving recording metadata for %s: %v", filePath, err)
						} else { callCtx.Log().Infof("Recording metadata saved for %s with SID %s", filePath, recordingSid) }
					}

					if pendingRecInfo.OriginalElement.ActionURL != "" && s.xmlProcessor != nil {
						params := url.Values{}
						params.Set("RecordingSid", recordingSid)
						params.Set("RecordingDuration", fmt.Sprintf("%d", durationSec))
						params.Set("RecordingUrl", filePath) // Placeholder: This should be a public URL
						params.Set("CallSid", callCtx.GetUuid())
						params.Set("AccountSid", callCtx.GetAccountSid())

						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingRecInfo.OriginalElement.GetActionURL(), pendingRecInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && len(newElements) > 0 { // Check for newElements != nil too
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil {
								callCtx.Log().Errorf("Failed to send new elements from Record ActionURL to channel: %v", sendErr)
							}
						} else if fetchErr != nil {
							callCtx.Log().Errorf("Error fetching/parsing XML from Record ActionURL %s: %v", pendingRecInfo.OriginalElement.GetActionURL(), fetchErr)
						}
					}
				} else { callCtx.Log().Warnf("RECORD_STOP for file '%s' does not match pending recording path '%s'", filePath, pendingRecInfo.ExpectedFilePath) }
			} else { callCtx.Log().Warnf("Received RECORD_STOP but no pending recording info found for file: %s", filePath) }

		case "CHANNEL_ANSWER":
			answeredUUID := event.Get("Unique-ID")
			if answeredUUID == callCtx.GetFreeswitchUUID() {
				callCtx.Log().Info("Received CHANNEL_ANSWER for A-leg.")
				if s.callService != nil { _ = s.callService.UpdateCallStatus(callCtx, string(domain.CallStatusInProgress), "") }
			} else {
				pendingDialInfoInter, exists := callCtx.GetPendingDial(answeredUUID)
				if exists && pendingDialInfoInter != nil {
					pendingDialInfo := pendingDialInfoInter.(*callcontrol.PendingDialInfo)
					callCtx.Log().Infof("B-leg %s answered for Dial to %s (Parent AgbaraCallSID: %s)",
						answeredUUID, pendingDialInfo.OriginalElement.CalleeIDToDial, pendingDialInfo.ParentAgbaraCallSID)

					_, bridgeErr := eslExecutor.Execute("uuid_bridge", callCtx.GetFreeswitchUUID(), answeredUUID)
					if bridgeErr != nil { callCtx.Log().Errorf("Error bridging A-leg %s to B-leg %s: %v", callCtx.GetFreeswitchUUID(), answeredUUID, bridgeErr)
					} else { callCtx.Log().Infof("Successfully sent uuid_bridge command for A-leg %s to B-leg %s", callCtx.GetFreeswitchUUID(), answeredUUID) }

					if digitsToSend := callCtx.GetVariable("agbara_dial_send_digits_on_answer"); digitsToSend != "" { // Check on A-leg's context
						callCtx.Log().Infof("Sending digits '%s' to B-leg %s", digitsToSend, answeredUUID)
						_, dtmfErr := eslExecutor.Execute("uuid_send_dtmf", answeredUUID, digitsToSend)
						if dtmfErr != nil { callCtx.Log().Errorf("Error sending DTMF '%s' to B-leg %s: %v", digitsToSend, answeredUUID, dtmfErr) }
					}

					if pendingDialInfo.OriginalElement.ActionURL != "" && s.xmlProcessor != nil {
						params := url.Values{}
						params.Set("DialCallStatus", "answered"); params.Set("DialCallSid", answeredUUID); params.Set("DialBlegUuid", answeredUUID); params.Set("ParentCallSid", pendingDialInfo.ParentAgbaraCallSID)
						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingDialInfo.OriginalElement.GetActionURL(), pendingDialInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && len(newElements) > 0 { // Check for newElements != nil
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil { callCtx.Log().Errorf("Failed to send new elements from Dial Answer ActionURL: %v", sendErr) }
						} else if fetchErr != nil { callCtx.Log().Errorf("Error fetching/parsing XML from Dial Answer ActionURL %s: %v", pendingDialInfo.OriginalElement.GetActionURL(), fetchErr) }
					}
				} else { callCtx.Log().Debugf("Received CHANNEL_ANSWER for non-pending B-leg UUID: %s", answeredUUID) }
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
					if pendingDialInfo.OriginalElement.ActionURL != "" && s.xmlProcessor != nil {
						params := url.Values{}; dialCallStatus := "completed"
						if mappedStatus, ok := localFreeswitchHangupCauseToDialStatus[hangupCause]; ok { dialCallStatus = mappedStatus
						} else if hangupCause != "NORMAL_CLEARING" { dialCallStatus = "failed" }
						params.Set("DialCallStatus", dialCallStatus); params.Set("DialCallSid", hangedUpUUID); params.Set("DialBlegUuid", hangedUpUUID); params.Set("ParentCallSid", pendingDialInfo.ParentAgbaraCallSID); params.Set("DialHangupCause", hangupCause)
						newElements, fetchErr := s.xmlProcessor.FetchAndParseXML(context.Background(), callCtx, pendingDialInfo.OriginalElement.GetActionURL(), pendingDialInfo.OriginalElement.GetMethod(), params)
						if fetchErr == nil && len(newElements) > 0 { // Check for newElements != nil
							if sendErr := callCtx.SendNextElements(newElements); sendErr != nil { callCtx.Log().Errorf("Failed to send new elements from Dial Hangup ActionURL: %v", sendErr) }
						} else if fetchErr != nil { callCtx.Log().Errorf("Error fetching/parsing XML from Dial Hangup ActionURL %s: %v", pendingDialInfo.OriginalElement.GetActionURL(), fetchErr) }
					}
				} else { callCtx.Log().Debugf("Received CHANNEL_HANGUP for non-pending B-leg UUID: %s", hangedUpUUID) }
			}
		case "CHANNEL_HANGUP_COMPLETE":
			if eventUUID == callCtx.GetFreeswitchUUID() {
				callCtx.Log().Infof("Received CHANNEL_HANGUP_COMPLETE for A-leg %s. Terminating event handler.", callCtx.GetUuid())
				callCtx.SetHangupInitiated(); return
			}
		case "DTMF":
			digit := event.Get("DTMF-Digit")
			callCtx.Log().Infof("Received DTMF: Digit='%s', Duration=%s", digit, event.Get("DTMF-Duration"))
			if recInfoInter, recExists := callCtx.GetPendingRecording(); recExists && recInfoInter != nil {
				recInfo := recInfoInter.(*callcontrol.PendingRecordInfo)
				if recInfo.OriginalElement.FinishOnKey != "" && strings.Contains(recInfo.OriginalElement.FinishOnKey, digit) {
					callCtx.Log().Infof("FinishOnKey '%s' received during recording. Stopping recording %s.", digit, recInfo.ExpectedFilePath)
					_, err := eslExecutor.Execute("uuid_record", callCtx.GetFreeswitchUUID(), "stop", recInfo.ExpectedFilePath)
					if err != nil { callCtx.Log().Errorf("Error stopping recording %s on FinishOnKey: %v", recInfo.ExpectedFilePath, err) }
				}
			}
		case "CONFERENCE_MAINTENANCE":
			confName := event.Get("Conference-Name")
			confUniqueID := event.Get("Conference-Unique-ID")
			memberID := event.Get("Member-ID")
			eventSubclass := event.Get("Event-Subclass")
			participantCallFsUUID := event.Get("Caller-Channel-UUID")

			currentConfSID, inConf := callCtx.GetCurrentConferenceSID()
			currentConfName, _ := callCtx.GetCurrentConferenceName()

			if !inConf || currentConfName != confName {
				callCtx.Log().Debugf("Ignoring CONFERENCE_MAINTENANCE for conf '%s', current context conf is '%s'", confName, currentConfName)
				continue
			}

			callCtx.Log().Infof("Conference Event: %s for %s (MemberID: %s, CallFSID: %s)", eventSubclass, confName, memberID, participantCallFsUUID)

			var participantCallAgbaraSID string
			if participantCallFsUUID == callCtx.GetFreeswitchUUID() {
				participantCallAgbaraSID = callCtx.GetUuid()
			} else {
				participantCallAgbaraSID = event.Get("Caller-Caller-ID-Number")
				if participantCallAgbaraSID == "" {  participantCallAgbaraSID = "unknown:" + participantCallFsUUID }
			}

			var dbConf *domain.Conference
			var err error
			if s.callService != nil { // Check if callService is available
				dbConf, err = s.callService.GetConferenceBySID(context.Background(), currentConfSID)
				if err != nil {
					callCtx.Log().Errorf("ConfEvent: Failed to get conference %s from DB: %v", currentConfSID, err)
					continue
				}
			} else {
				callCtx.Log().Error("ConfEvent: callService is nil, cannot process conference DB operations.")
				continue
			}


			params := url.Values{}
			params.Set("ConferenceSid", dbConf.SID)
			params.Set("ConferenceFriendlyName", dbConf.FriendlyName)
			params.Set("Timestamp", time.Now().UTC().Format(time.RFC3339))
			params.Set("EventMemberID", memberID) // Freeswitch Member-ID

			switch eventSubclass {
			case "conference::maintenance::add-member":
				params.Set("Event", "participant-join")
				var pSID string
				if participantCallFsUUID == callCtx.GetFreeswitchUUID() {
					pSID, _ = callCtx.GetCurrentConferenceParticipantSID()
				} else {
					part, pErr := s.callService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
					if pErr == nil && part != nil { pSID = part.SID
					} else { // Participant might not be in DB yet if joined via other means or this is the first event
						// Attempt to add this newly discovered participant.
						// This requires AccountSID. Assume it's the same as the current call's AccountSID for now.
						newPSID := utils.GenerateSID("CP")
						isMuted := event.Get("Speak") == "false" // Approximation
						isModerator := event.Get("Control") == "moderator" // Approximation
						_, addErr := s.callService.AddParticipant(context.Background(), dbConf.SID, participantCallAgbaraSID, newPSID, callCtx.GetAccountSid(), isMuted, isModerator)
						if addErr == nil { pSID = newPSID
						} else if errors.Is(addErr, domain.ErrConflict) { // Already added by another event/process
							partRetry, pErrRetry := s.callService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
							if pErrRetry == nil && partRetry != nil { pSID = partRetry.SID } else {pSID = "unknown_conflict"}
						} else {
							callCtx.Log().Errorf("Failed to add newly discovered participant %s to DB: %v", participantCallAgbaraSID, addErr)
							pSID = "unknown_error"
						}
					}
				}
				params.Set("ParticipantSid", pSID)
				params.Set("CallSid", participantCallAgbaraSID)

			case "conference::maintenance::del-member":
				params.Set("Event", "participant-leave")
				part, pErr := s.callService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
				if pErr == nil && part != nil {
					_ = s.callService.RemoveParticipant(context.Background(), part.SID, time.Now().UTC())
					params.Set("ParticipantSid", part.SID)
				} else {
					params.Set("ParticipantSid", "unknown")
				}
				params.Set("CallSid", participantCallAgbaraSID)
				if participantCallFsUUID == callCtx.GetFreeswitchUUID() { callCtx.LeaveConference() }

			case "conference::maintenance::mute-member", "conference::maintenance::unmute-member":
				params.Set("Event", "participant-mute-update")
				part, pErr := s.callService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
				if pErr == nil && part != nil {
					newMuteState := eventSubclass == "conference::maintenance::mute-member"
					_ = s.callService.UpdateParticipantMuteStatus(context.Background(), part.SID, newMuteState)
					params.Set("ParticipantSid", part.SID); params.Set("Muted", fmt.Sprintf("%t", newMuteState))
				} else {params.Set("ParticipantSid", "unknown")}
				params.Set("CallSid", participantCallAgbaraSID)

			case "conference::maintenance::start-talking", "conference::maintenance::stop-talking":
				params.Set("Event", "participant-talk-status")
				part, pErr := s.callService.GetParticipantByCallSID(context.Background(), participantCallAgbaraSID)
				if pErr == nil && part != nil { params.Set("ParticipantSid", part.SID)
				} else { params.Set("ParticipantSid", "unknown") }
				params.Set("CallSid", participantCallAgbaraSID)
				params.Set("Talking", fmt.Sprintf("%t", eventSubclass == "conference::maintenance::start-talking"))

			case "conference::maintenance::end":
				params.Set("Event", "conference-end")
				if s.callService != nil { _ = s.callService.EndConference(context.Background(), dbConf.SID, time.Now().UTC()) }
				callCtx.LeaveConference()

			default:
				callCtx.Log().Debugf("Unhandled conference event subclass: %s", eventSubclass)
				continue
			}

			cbURL, hasCB := callCtx.GetCurrentConferenceCallbackURL()
			cbMethod, _ := callCtx.GetCurrentConferenceCallbackMethod()
			if hasCB && cbURL != "" && s.xmlProcessor != nil { // Check s.xmlProcessor as well
				// Call sendConferenceCallback which is fire-and-forget for simple notifications
				// If ActionURL-like behavior (new XML) is desired for conference events,
				// then FetchAndParseXML and SendNextElements would be used here.
				// The prompt implies informational callbacks, so sendConferenceCallback is appropriate.
				go sendConferenceCallback(cbURL, cbMethod, params, callCtx.Log())
			}

		default:
			// Other events can be logged or handled as needed
		}
	}
}

// finalHangup now takes EslConnectionExecutor as it cannot be reliably obtained from MinimalCallContext
func (s *FSOutboundServer) finalHangup(callCtx domain.MinimalCallContext, reason string, eslExecutor domain.EslConnectionExecutor) {
	if callCtx == nil {
		s.logger.Error("finalHangup called with nil callCtx")
		return
	}
	if eslExecutor == nil {
		callCtx.Log().Error("finalHangup called with nil eslExecutor. Cannot send Hangup command.")
		if s.callService != nil {
			_ = s.callService.UpdateCallStatus(callCtx, string(domain.CallStatusFailed), "ESL_EXECUTOR_NIL_ON_HANGUP")
		}
		callCtx.SetHangupInitiated()
		return
	}

	if callCtx.IsHangupInitiated() && reason != "CHANNEL_UNBRIDGE_HANGUP" {
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
		finalStatus := domain.CallStatusCompleted
		if reason != "NORMAL_CLEARING" && reason != "NORMAL_TEMPORARY_FAILURE" {
			if _, ok := localFreeswitchHangupCauseToDialStatus[reason]; !ok { // Use local map
				finalStatus = domain.CallStatusFailed
			}
			// Potentially map to other statuses based on reason if localFreeswitchHangupCauseToDialStatus was more comprehensive
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

func NewESLConnectionAdapter(eslConn *eventsocket.Connection, logger *logrus.Entry) domain.EslConnectionExecutor { // Added logger
	// Use passed logger or a default if nil
	var adapterLogger *logrus.Entry
	if logger != nil {
		adapterLogger = logger.WithField("adapter", "ESLConnectionAdapter")
	} else {
		defaultLog := logrus.New()
		defaultLog.SetOutput(io.Discard) // Default to discard if no logger passed
		adapterLogger = logrus.NewEntry(defaultLog).WithField("adapter","ESLConnectionAdapter")
	}
	return &ESLConnectionAdapter{conn: eslConn, log: adapterLogger}
}

func (a *ESLConnectionAdapter) Execute(command string, args ...string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter Execute")}
	argStr := strings.Join(args, " ")
	a.log.Debugf("Adapter Executing: App='%s', Args='%s'", command, argStr)
	ev, err := a.conn.Execute(command, argStr, true)
	if err != nil { return "", err }
	if ev == nil { return "", errors.New("nil event received from ESL Execute") }
	return ev.Get("Reply-Text"), nil
}

func (a *ESLConnectionAdapter) ExecuteSofia(command string, args ...string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter ExecuteSofia")}
	fullCmd := fmt.Sprintf("sofia %s %s", command, strings.Join(args, " "))
	a.log.Debugf("Adapter ExecuteSofia (as API): %s", fullCmd)
	ev, err := a.conn.Send(fmt.Sprintf("api %s", fullCmd))
	if err != nil { return "", err}
	if ev == nil { return "", errors.New("nil event received")}
	return ev.Body, nil
}

func (a *ESLConnectionAdapter) SendMsg(msg map[string]string) (string, error) {
	if a.conn == nil { return "", errors.New("ESL connection is nil in adapter SendMsg")}
	if cmd, ok := msg["command"]; ok {
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
	if strings.Contains(replyText, "-ERR Already Answered") || strings.Contains(replyText, "UNKNOWN COMMAND") {
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
		escapedValue := strings.ReplaceAll(v, ",", "\\,")
		escapedValue = strings.ReplaceAll(escapedValue, "{", "\\{")
		escapedValue = strings.ReplaceAll(escapedValue, "}", "\\}")
		varList = append(varList, fmt.Sprintf("%s='%s'", k, escapedValue))
	}
	varsString := ""
	if len(varList) > 0 { varsString = "{" + strings.Join(varList, ",") + "}" }

	originateCmd := fmt.Sprintf("bgapi originate %s%s", varsString, dialString)
	a.log.Debugf("Adapter Originate (via bgapi): %s", originateCmd)

	ev, err := a.conn.Send(originateCmd)
	if err != nil { return "", err }
	if ev == nil { return "", errors.New("nil event from bgapi originate")}

	jobUUID := ev.Get("Job-UUID")
	if jobUUID == "" {
		if strings.HasPrefix(ev.Body, "+OK Job-UUID: ") {
			jobUUID = strings.TrimPrefix(ev.Body, "+OK Job-UUID: ")
		} else if strings.HasPrefix(ev.Body, "+OK") {
            return "OK_NO_JOB_UUID", nil
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
	fsUUID := ev.Get("Unique-ID")
	if fsUUID == "" { return nil, errors.New("Unique-ID not found in connect event") }

	agbaraCallSid := ev.Get("variable_agbara_call_sid")
	if agbaraCallSid == "" {
		agbaraCallSid = fsUUID
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
	vars["uuid"] = fsUUID
	vars["agbara_call_sid"] = agbaraCallSid
	vars["channel_state"] = ev.Get("Channel-State")
	vars["channel_call_state"] = ev.Get("Channel-Call-State")
	vars["caller_id_number"] = ev.Get("Caller-Caller-ID-Number")
	vars["destination_number"] = ev.Get("Caller-Destination-Number")
	vars["agbara_account_sid"] = accountSid
	vars["agbara_application_sid"] = appSid
	vars["agbara_answer_url"] = answerURL

	return callcontrol.NewCallContext(agbaraCallSid, fsUUID, accountSid, appSid, answerURL, vars, eslExecutor, baseLogger), nil
}

// Helper to generate a unique recording SID (moved from domain to avoid import cycle if domain needs utils)
// func GenerateRecordingSID() string {
//     return fmt.Sprintf("RE%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
// }

// Helper for logging nullable strings, can be moved to a common utils package
func LogNullString(ns *string) string {
    if ns == nil {
        return "<nil>"
    }
    return *ns
}
