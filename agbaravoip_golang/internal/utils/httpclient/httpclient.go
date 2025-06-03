package httpclient

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/user/agbaravoip_golang/internal/domain" // Corrected import path
	"github.com/sirupsen/logrus"                            // Or your logging library
)

const defaultRequestTimeout = 15 * time.Second

// FetchXML fetches an XML document from the given URL using the specified method, parameters, and headers.
// It unmarshals the response into a domain.ResponseElement.
// The provided http.Client will be used for the request. If nil, a default client is used.
func FetchXML(
	ctx context.Context,
	httpClient *http.Client,
	urlStr string,
	method string,
	params url.Values, // For POST form data or GET query params
	headers map[string]string,
	log *logrus.Entry, // Pass a logger entry for contextual logging
) (*domain.ResponseElement, error) {

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultRequestTimeout}
	}

	// Prepare request body for POST/PUT, or append query params for GET
	var reqBody io.Reader
	if method == http.MethodPost || method == http.MethodPut {
		if params != nil && len(params) > 0 { // Ensure params are not empty before creating reader
			reqBody = strings.NewReader(params.Encode())
			if headers == nil {
				headers = make(map[string]string)
			}
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	} else if method == http.MethodGet {
		if params != nil && len(params) > 0 {
			parsedURL, err := url.Parse(urlStr)
			if err != nil {
				log.Errorf("Error parsing URL '%s' for GET params: %v", urlStr, err)
				return nil, fmt.Errorf("parsing URL for GET params: %w", err)
			}
			query := parsedURL.Query()
			for k, v := range params {
				for _, val := range v {
					query.Add(k, val)
				}
			}
			parsedURL.RawQuery = query.Encode()
			urlStr = parsedURL.String()
		}
	}

	// Use httpClient.Timeout if set, otherwise defaultRequestTimeout for the context.
	// If httpClient.Timeout is 0 (meaning no timeout), then we should not create a context with timeout
	// or use a default one. For simplicity here, we assume httpClient.Timeout is the effective timeout.
	// If httpClient.Timeout is zero, this means the request can run indefinitely, which might be
	// intended for specific clients. The WithTimeout below will use this value.
	// If httpClient.Timeout is not set (is 0), use defaultRequestTimeout for the context.
	requestSpecificTimeout := httpClient.Timeout
	if requestSpecificTimeout <= 0 {
		requestSpecificTimeout = defaultRequestTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, requestSpecificTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, strings.ToUpper(method), urlStr, reqBody)
	if err != nil {
		log.Errorf("Error creating HTTP request to '%s': %v", urlStr, err)
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	// Add custom headers
	if headers != nil {
		for key, value := range headers {
			req.Header.Set(key, value)
		}
	}
	// Add a default User-Agent
	req.Header.Set("User-Agent", "AgbaraVOIP-GoLang/1.0")
	// Ensure Host header is set if not automatically done by client. Usually it is.
	// req.Host = parsedURL.Host (if needed and parsedURL is available)

	log.Infof("Fetching XML from URL: %s, Method: %s", urlStr, method)

	resp, err := httpClient.Do(req)
	if err != nil {
		// Check for context deadline exceeded specifically
		if ue, ok := err.(*url.Error); ok && ue.Err == context.DeadlineExceeded {
			log.Errorf("HTTP request to '%s' timed out: %v", urlStr, err)
			return nil, fmt.Errorf("request to %s timed out: %w", urlStr, err)
		}
		log.Errorf("Error performing HTTP request to '%s': %v", urlStr, err)
		return nil, fmt.Errorf("performing HTTP request to %s: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body) // Read body for error message
		log.Errorf("HTTP request to '%s' failed with status %d: %s", urlStr, resp.StatusCode, string(bodyBytes))
		return nil, fmt.Errorf("request to %s failed with status %d: %s", urlStr, resp.StatusCode, string(bodyBytes))
	}

	// Check content type, should ideally be application/xml or text/xml
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/xml") && !strings.HasPrefix(contentType, "text/xml") {
		log.Warnf("Response from '%s' has non-XML Content-Type: %s. Proceeding with parsing.", urlStr, contentType)
		// Continue processing anyway, as some servers might misconfigure Content-Type
	}

	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading response body from '%s': %v", urlStr, err)
		return nil, fmt.Errorf("reading response body from %s: %w", urlStr, err)
	}

	log.Debugf("Received XML response body from %s: %s", urlStr, string(responseBodyBytes))

	var responseElement domain.ResponseElement
	if err := xml.Unmarshal(responseBodyBytes, &responseElement); err != nil {
		log.Errorf("Error unmarshaling XML response from '%s': %v. Body: %s", urlStr, err, string(responseBodyBytes))
		// Include the body in the error for easier debugging
		return nil, fmt.Errorf("unmarshaling XML response from %s: %w. Body: %s", urlStr, err, string(responseBodyBytes))
	}

	log.Infof("Successfully fetched and parsed XML from '%s'", urlStr)
	return &responseElement, nil
}
