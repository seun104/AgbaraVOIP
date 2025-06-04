package freeswitch

import (
	// "bufio" // Not needed with eslgo.ReadMessage()
	"fmt"
	"log"
	"strings"
	"sync" // For managing goroutine shutdown
	"time"

	"github.com/percipia/eslgo" 
)

type ESLConnection struct {
	client *eslgo.ESL
	host   string
	port   string
	pass   string

	eventSubscriptions string              // e.g., "ALL", "CHANNEL_CREATE CHANNEL_HANGUP"
	eventChannel       chan *eslgo.Message // Channel to send parsed events out
	shutdownChan       chan struct{}       // To signal event listener goroutine to stop
	wg                 sync.WaitGroup      // To wait for goroutine to finish
}

func NewESLConnection(host, port, password string,
	connectionTimeout time.Duration, maxRetries int,
	eventSubs string, eventChan chan *eslgo.Message) (*ESLConnection, error) {
	conn := &ESLConnection{
		host:               host,
		port:               port,
		pass:               password,
		eventSubscriptions: eventSubs,
		eventChannel:       eventChan, 
		shutdownChan:       make(chan struct{}),
	}

	var lastErr error // Store the last error for returning after retries
	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to FreeSWITCH ESL (%s:%s) for commands/events - attempt %d/%d\n", host, port, i+1, maxRetries)
		c, clientErr := eslgo.NewConnection(host, port, password, int(connectionTimeout.Seconds()))
		if clientErr == nil {
			conn.client = c
			log.Printf("Successfully connected to FreeSWITCH ESL at %s:%s\n", host, port)

			if conn.eventSubscriptions != "" {
				eventCmd := fmt.Sprintf("events plain %s", conn.eventSubscriptions)
				log.Printf("Sending event subscription: %s\n", eventCmd)
				_, errSub := conn.client.Send(eventCmd)
				if errSub != nil {
					log.Printf("ERROR: Failed to subscribe to events ('%s'): %v. Closing connection and will retry.", eventCmd, errSub)
					conn.client.CloseConnection()
					conn.client = nil
					lastErr = errSub // Store this error
					// Continue to the next retry iteration
					if i < maxRetries-1 { time.Sleep(3 * time.Second) }
					continue
				}
				log.Printf("Successfully subscribed to events: %s\n", conn.eventSubscriptions)
			}

			conn.wg.Add(1)
			go conn.listenForEvents()

			return conn, nil // Successful connection and setup
		}
		log.Printf("ESL Connection attempt %d failed: %v\n", i+1, clientErr)
		lastErr = clientErr // Store this error
		if i < maxRetries-1 {
			time.Sleep(3 * time.Second) 
		}
	}
	return nil, fmt.Errorf("failed to connect to FreeSWITCH ESL after %d retries: %w", maxRetries, lastErr)
}

func (ec *ESLConnection) listenForEvents() {
	defer ec.wg.Done()
	log.Println("ESL event listener started.")

	for {
		select {
		case <-ec.shutdownChan:
			log.Println("ESL event listener shutting down (received shutdown signal).")
			return
		default:
			// Ensure client is not nil before attempting ReadMessage
			if ec.client == nil {
				log.Println("ESL event listener: client is nil, attempting to reconnect...")
				// Trigger reconnection logic directly
                if !ec.performReconnect() { // If reconnect fails permanently after its retries
                    log.Println("ESL event listener: Permanent reconnection failure. Listener stopping.")
                    return
                }
                // If reconnected, continue to next iteration to ReadMessage
				continue
			}

			msg, err := ec.client.ReadMessage()
			if err != nil {
				select {
				case <-ec.shutdownChan:
					log.Println("ESL event listener: ReadMessage error due to shutdown.")
					return 
				default:
					log.Printf("ERROR: Failed to read ESL message: %v. Attempting to reconnect...\n", err)
					ec.client.CloseConnection() 
					ec.client = nil            
                    if !ec.performReconnect() { // If reconnect fails permanently after its retries
                        log.Println("ESL event listener: Permanent reconnection failure after read error. Listener stopping.")
                        return
                    }
                    // If reconnected, continue to next iteration to ReadMessage
					continue 
				}
			}

			if msg == nil {
				continue
			}

			contentType := msg.GetHeader("Content-Type")
			if contentType == "text/event-plain" || contentType == "text/event-json" {
				// log.Printf("Received FreeSWITCH Event: Name: %s, Subclass: %s, UUID: %s\nHeaders: %+v\nBody: %s\n",
				// 	msg.GetHeader("Event-Name"),
				// 	msg.GetHeader("Event-Subclass"),
				// 	msg.GetHeader("Unique-ID"), 
				// 	msg.Headers,
				// 	msg.Body,
				// )
				if ec.eventChannel != nil {
					select {
					case ec.eventChannel <- msg:
						// Event sent
					case <-ec.shutdownChan:
						log.Println("ESL event listener: Shutdown while trying to send event to channel.")
						return
					default:
						log.Printf("WARN: ESL event channel is full. Event for %s might be dropped or delayed.", msg.GetHeader("Event-Name"))
					}
				}
			} else if contentType == "text/disconnect-notice" {
                log.Println("ESL event listener: Received disconnect notice. Attempting to reconnect...")
                ec.client.CloseConnection()
                ec.client = nil
                if !ec.performReconnect() {
                    log.Println("ESL event listener: Permanent reconnection failure after disconnect notice. Listener stopping.")
                    return
                }
                continue
            } else if contentType == "auth/request" {
                log.Println("WARN: Received unexpected auth/request in event listener. Should be handled by NewConnection.")
			} else {
				// command/reply, api/response are handled by Send methods directly.
				// log.Printf("DEBUG: Received ESL message of Content-Type '%s': Headers: %+v\n", contentType, msg.Headers)
			}
		}
	}
}

// performReconnect attempts to re-establish the ESL connection and re-subscribe to events.
// Returns true if successful, false if shutdown is signaled or retries exhausted.
func (ec *ESLConnection) performReconnect() bool {
    const reconnectMaxRetries = 5 // Or make configurable
    const reconnectSleep = 5 * time.Second

    for i := 0; i < reconnectMaxRetries; i++ {
        select {
        case <-ec.shutdownChan:
            log.Println("ESL event listener: Reconnect aborted due to shutdown signal.")
            return false
        default:
            log.Printf("ESL event listener: Reconnect attempt %d/%d...\n", i+1, reconnectMaxRetries)
            newClient, reconnErr := eslgo.NewConnection(ec.host, ec.port, ec.pass, 10) // 10s timeout
            if reconnErr == nil {
                ec.client = newClient
                log.Println("ESL event listener: Reconnected successfully.")
                if ec.eventSubscriptions != "" {
                    eventCmd := fmt.Sprintf("events plain %s", ec.eventSubscriptions)
                    log.Printf("Re-subscribing to events: %s\n", eventCmd)
                    _, errSub := ec.client.Send(eventCmd)
                    if errSub != nil {
                        log.Printf("ERROR: Failed to re-subscribe to events: %v. Closing this new connection.", errSub)
                        ec.client.CloseConnection()
                        ec.client = nil
                        // Fall through to sleep and retry connection loop
                    } else {
                        log.Println("Successfully re-subscribed to events.")
                        return true // Successfully reconnected and re-subscribed
                    }
                } else {
                    return true // Successfully reconnected, no subscriptions needed
                }
            }
            log.Printf("ESL event listener: Reconnect attempt %d failed: %v\n", i+1, reconnErr)
            
            // Wait before next reconnect attempt, checking for shutdown
            select {
            case <-time.After(reconnectSleep):
            case <-ec.shutdownChan:
                log.Println("ESL event listener: Reconnect sleep interrupted by shutdown.")
                return false
            }
        }
    }
    log.Println("ESL event listener: Exhausted reconnect retries.")
    return false // Exhausted retries
}


func (ec *ESLConnection) Close() {
	if ec.client != nil || len(ec.shutdownChan) == 0 { // Check if shutdownChan is already closed
		log.Println("Closing FreeSWITCH ESL connection and stopping event listener.")
		close(ec.shutdownChan) 
		if ec.client != nil {
			ec.client.CloseConnection()
		}
		ec.wg.Wait() 
		log.Println("ESL Connection and listener fully stopped.")
	} else {
        log.Println("ESL Connection already closed or never fully initialized.")
    }
}

func (ec *ESLConnection) SendBgApiCommand(command string, args string) (string, error) {
	if ec.client == nil { return "", fmt.Errorf("not connected to FreeSWITCH ESL") }
	msg, err := ec.client.Send(fmt.Sprintf("bgapi %s %s", command, args))
	if err != nil { return "", fmt.Errorf("failed to send bgapi command '%s': %w", command, err) }
	responseStr := msg.Response
	if strings.HasPrefix(responseStr, "+OK Job-UUID:") {
		return strings.TrimSpace(responseStr[len("+OK Job-UUID:"):]), nil
	} else if strings.HasPrefix(responseStr, "-ERR") {
        return "", fmt.Errorf("bgapi command '%s' failed: %s", command, responseStr)
    }
	log.Printf("Unexpected bgapi response for '%s': %s\n", command, responseStr)
	return "", fmt.Errorf("unexpected response from bgapi command '%s': %s", command, responseStr)
}

func (ec *ESLConnection) SendApiCommand(command string, args string) (string, error) {
    if ec.client == nil { return "", fmt.Errorf("not connected to FreeSWITCH ESL") }
    msg, err := ec.client.Send(fmt.Sprintf("api %s %s", command, args))
    if err != nil { return "", fmt.Errorf("failed to send api command '%s': %w", command, err) }
    return msg.Response, nil
}
