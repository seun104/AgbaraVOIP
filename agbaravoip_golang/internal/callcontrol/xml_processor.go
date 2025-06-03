package callcontrol

import (
	"context"
	// "encoding/xml" // No longer needed directly here
	"fmt"
	// "io" // No longer needed directly here
	"net/http"
	"net/url"
	// "strings" // No longer needed directly here

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils/httpclient"
	"github.com/sirupsen/logrus"
)

// XMLProcessor handles fetching and parsing of AgbaraXML documents.
type XMLProcessor struct {
	httpClient *http.Client
	// callService domain.CallServicerForESL // Might not be needed directly here if status updates are in esl/outbound.go
}

// NewXMLProcessor creates a new XMLProcessor.
// Pass a configured http.Client (e.g., with connection pooling). If nil, FetchXML will use its default.
func NewXMLProcessor(client *http.Client) *XMLProcessor {
	return &XMLProcessor{
		httpClient: client,
	}
}

// FetchAndParseXML fetches an XML document from the given URL and parses it into a slice of CallControlElements.
// It now uses the new httpclient.FetchXML function.
func (p *XMLProcessor) FetchAndParseXML(
	ctx context.Context, // Pass context for httpclient
	callCtx domain.MinimalCallContext, // Used for logging and potentially other details
	urlStr string,
	method string, // "GET" or "POST"
	requestParams url.Values, // Parameters for the HTTP request
) ([]domain.CallControlElement, error) {
	log := callCtx.Log().WithFields(logrus.Fields{
		"service": "XMLProcessor",
		"url":     urlStr,
		"method":  method,
	})

	log.Info("Fetching and parsing XML")

	// Prepare headers if needed - for now, passing nil
	var headers map[string]string = nil
	// Example: headers = map[string]string{"X-Custom-Header": "value"}

	responseElement, err := httpclient.FetchXML(ctx, p.httpClient, urlStr, method, requestParams, headers, log)
	if err != nil {
		log.Errorf("Failed to fetch or parse XML: %v", err)
		// Specific error handling or status updates might occur in the caller (e.g., FSOutboundServer)
		return nil, fmt.Errorf("httpclient.FetchXML failed: %w", err)
	}

	if responseElement == nil {
		log.Error("Received nil ResponseElement from FetchXML without error, this should not happen.")
		return nil, fmt.Errorf("received nil ResponseElement from FetchXML without error from URL %s", urlStr)
	}

	// Directly use the Verbs field, which should be populated by httpclient.FetchXML
	if responseElement.Verbs == nil {
		log.Warn("ResponseElement.Verbs is nil. No verbs to process?")
		return []domain.CallControlElement{}, nil // Return empty slice if no verbs
	}

    log.Infof("Successfully fetched and parsed %d XML elements from response via httpclient.", len(responseElement.Verbs))
	return responseElement.Verbs, nil
}
