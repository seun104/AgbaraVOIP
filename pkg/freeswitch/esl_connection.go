package freeswitch

import (
	"fmt"
	"log" // For logging connection errors, etc.
	"strings" // Added for strings.TrimSpace and strings.HasPrefix
	"time"

	"github.com/percipia/eslgo" // Assuming this is the correct import path
)

// ESLConnection manages a connection to FreeSWITCH Event Socket Layer.
type ESLConnection struct {
	client *eslgo.ESL // Assuming eslgo.ESL is the connection client type
	host   string
	port   string
	pass   string
}

// NewESLConnection creates and establishes a new ESL connection.
// It will attempt to connect upon creation.
func NewESLConnection(host, port, password string, connectionTimeout time.Duration, maxRetries int) (*ESLConnection, error) {
	conn := &ESLConnection{
		host: host,
		port: port,
		pass: password,
	}

	var err error
	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to FreeSWITCH ESL (%s:%s) - attempt %d/%d\n", host, port, i+1, maxRetries)
		// Based on percipia/eslgo docs, it's simpler:
		c, errConnect := eslgo.NewConnection(host, port, password, int(connectionTimeout.Seconds()))
		err = errConnect // Store the last error
		if err == nil {
			conn.client = c
			log.Printf("Successfully connected to FreeSWITCH ESL at %s:%s\n", host, port)
			// Optionally, subscribe to some events or check status if needed upon connection.
			// Example: _, err = c.Send("events plain ALL")
			return conn, nil
		}
		log.Printf("ESL Connection attempt %d failed: %v\n", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second) // Wait before retrying
		}
	}
	return nil, fmt.Errorf("failed to connect to FreeSWITCH ESL after %d retries: %w", maxRetries, err)
}

// Close closes the ESL connection.
func (ec *ESLConnection) Close() {
	if ec.client != nil {
		log.Println("Closing FreeSWITCH ESL connection.")
		// For percipia/eslgo, it's ec.client.CloseConnection()
		ec.client.CloseConnection()
	}
}

// SendBgApiCommand sends a command to FreeSWITCH using bgapi and returns the job UUID.
// Example: command = "originate", args = "{...}sofia/gateway/mygw/123&echo()"
func (ec *ESLConnection) SendBgApiCommand(command string, args string) (string, error) {
	if ec.client == nil {
		return "", fmt.Errorf("not connected to FreeSWITCH ESL")
	}

	// For percipia/eslgo: msg, err := ec.client.Send("bgapi " + command + " " + args)
	msg, err := ec.client.Send(fmt.Sprintf("bgapi %s %s", command, args))
	if err != nil {
		return "", fmt.Errorf("failed to send bgapi command '%s': %w", command, err)
	}

	// Parse Job-UUID from response. Example response: "+OK Job-UUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	// Assuming msg.Response is the string.
	responseStr := msg.Response
	if len(responseStr) > 12 && strings.HasPrefix(responseStr, "+OK Job-UUID:") { // More robust prefix check
		jobUUID := strings.TrimSpace(responseStr[13:]) // Adjusted index for "+OK Job-UUID:"
		return jobUUID, nil
	} else if strings.HasPrefix(responseStr, "-ERR") {
        return "", fmt.Errorf("bgapi command '%s' failed: %s", command, responseStr)
    }
	
	log.Printf("Unexpected bgapi response for '%s': %s\n", command, responseStr)
	return "", fmt.Errorf("unexpected response from bgapi command '%s': %s", command, responseStr)
}

// SendApiCommand sends a command to FreeSWITCH using api (foreground) and returns the full response.
// This is less common for originate but useful for other commands.
func (ec *ESLConnection) SendApiCommand(command string, args string) (string, error) {
    if ec.client == nil {
        return "", fmt.Errorf("not connected to FreeSWITCH ESL")
    }
    // For percipia/eslgo: msg, err := ec.client.Send("api " + command + " " + args)
    msg, err := ec.client.Send(fmt.Sprintf("api %s %s", command, args))
    if err != nil {
        return "", fmt.Errorf("failed to send api command '%s': %w", command, err)
    }
    // The response for api is the direct output of the command.
    return msg.Response, nil
}

// TODO: Add methods for event handling if needed later (e.g., SubscribeEvents, ReadEvents in a loop)
