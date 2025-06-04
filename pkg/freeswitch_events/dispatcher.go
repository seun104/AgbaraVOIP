package freeswitch_events

import (
	"agbara-go/pkg/models" // For a potential structured event model later
	"fmt"
	"log"
	"strconv" // Added for timestamp parsing
	"strings"
	"sync"

	"github.com/percipia/eslgo" // To handle *eslgo.Message
)

// ParsedEvent is a simpler structure for internal use after initial parsing from *eslgo.Message
type ParsedEvent struct {
	Name        string            // Event-Name
	Subclass    string            // Event-Subclass, if any
	UniqueID    string            // Unique-ID (channel UUID)
	Headers     map[string]string // All headers
	Body        string            // Raw body
	Timestamp   int64             // Event-Date-Timestamp or similar (Unix seconds)
}

// EventHandlerFunc defines the signature for functions that handle specific FreeSWITCH events.
type EventHandlerFunc func(event *ParsedEvent)

// EventDispatcher manages subscriptions and routes incoming FreeSWITCH events.
type EventDispatcher struct {
	eventSource  <-chan *eslgo.Message // Receives raw events from ESLConnection
	listeners    map[string][]EventHandlerFunc
	mu           sync.RWMutex
	shutdownChan chan struct{}
	wg           sync.WaitGroup
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher(eventSource <-chan *eslgo.Message) *EventDispatcher {
	return &EventDispatcher{
		eventSource:  eventSource,
		listeners:    make(map[string][]EventHandlerFunc),
		shutdownChan: make(chan struct{}),
	}
}

// Subscribe allows a component to register an EventHandlerFunc for specific event names.
func (d *EventDispatcher) Subscribe(eventName string, handler EventHandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	normalizedEventName := strings.ToUpper(eventName) 
	d.listeners[normalizedEventName] = append(d.listeners[normalizedEventName], handler)
	log.Printf("EventDispatcher: Handler registered for event '%s'\n", normalizedEventName)
}

// Start begins listening for events from the eventSource and dispatches them.
func (d *EventDispatcher) Start() {
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		log.Println("EventDispatcher started.")
		for {
			select {
			case rawEvent, ok := <-d.eventSource:
				if !ok {
					log.Println("EventDispatcher: Event source channel closed. Shutting down.")
					return
				}
				if rawEvent == nil {
					continue
				}
				d.dispatchEvent(rawEvent)
			case <-d.shutdownChan:
				log.Println("EventDispatcher: Shutdown signal received.")
				return
			}
		}
	}()
}

// Stop signals the EventDispatcher to stop processing events.
func (d *EventDispatcher) Stop() {
	log.Println("EventDispatcher: Stopping...")
	close(d.shutdownChan)
	d.wg.Wait() 
	log.Println("EventDispatcher: Stopped.")
}

// dispatchEvent parses the raw event and routes it to interested subscribers.
func (d *EventDispatcher) dispatchEvent(rawEvent *eslgo.Message) {
	headers := rawEvent.Headers() 
	eventName := strings.ToUpper(headers["Event-Name"])
	if eventName == "" {
		return
	}

	parsed := &ParsedEvent{
		Name:     eventName,
		Subclass: headers["Event-Subclass"],
		UniqueID: headers["Unique-ID"], 
		Headers:  headers,
		Body:     rawEvent.Body(),
	}
    if tsStr, ok := headers["Event-Date-Timestamp"]; ok {
        if ts, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
            parsed.Timestamp = ts / 1000000 // Convert microseconds to Unix seconds
        }
    }

	log.Printf("EventDispatcher: Dispatching event: Name: %s, Subclass: %s, UniqueID: %s\n",
		parsed.Name, parsed.Subclass, parsed.UniqueID)

	d.mu.RLock()
	// Dispatch to specific event listeners
	if handlers, found := d.listeners[parsed.Name]; found {
		for _, handler := range handlers {
			go handler(parsed) 
		}
	}
	// Dispatch to "ALL" listeners
	if handlers, found := d.listeners["ALL"]; found {
		for _, handler := range handlers {
			go handler(parsed) 
		}
	}
	d.mu.RUnlock() // Moved RUnlock to after both dispatches
}

// Helper to convert *eslgo.Message to models.FreeSwitchEvent (if we define a more detailed model)
// func ToModelFreeSwitchEvent(msg *eslgo.Message) *models.FreeSwitchEvent { ... }
