package httpclient_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/user/agbaravoip_golang/internal/utils/httpclient"
)

// Helper to create a logger for tests
func newTestLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetOutput(io.Discard) // Don't show logs during tests
	return logrus.NewEntry(logger)
}

func TestFetchXML_SuccessGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "value1", r.URL.Query().Get("param1"))
		fmt.Fprintln(w, `<Response><Say>Hello</Say></Response>`)
	}))
	defer server.Close()

	params := url.Values{}
	params.Add("param1", "value1")

	respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, params, nil, newTestLogger())

	assert.NoError(t, err)
	assert.NotNil(t, respElement)
	assert.Len(t, respElement.Verbs, 1)
	sayElement, ok := respElement.Verbs[0].(*domain.SayElement)
	assert.True(t, ok)
	assert.Equal(t, "Hello", sayElement.Text)
}

func TestFetchXML_SuccessPOST(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "valuePost", r.FormValue("paramPost"))

		fmt.Fprintln(w, `<Response><Play>audio.wav</Play></Response>`)
	}))
	defer server.Close()

	params := url.Values{}
	params.Add("paramPost", "valuePost")
	headers := map[string]string{"X-Custom-Header": "Test"}

	respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodPost, params, headers, newTestLogger())

	assert.NoError(t, err)
	assert.NotNil(t, respElement)
	assert.Len(t, respElement.Verbs, 1)
	playElement, ok := respElement.Verbs[0].(*domain.PlayElement)
	assert.True(t, ok)
	assert.Equal(t, "audio.wav", playElement.URL)
}

func TestFetchXML_HTTPErrorNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, "Not Found Here")
	}))
	defer server.Close()

	_, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed with status 404")
	assert.Contains(t, err.Error(), "Not Found Here")
}

func TestFetchXML_HTTPErrorInternalServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed with status 500")
}

func TestFetchXML_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Sleep longer than client timeout
		fmt.Fprintln(w, `<Response><Say>Too late</Say></Response>`)
	}))
	defer server.Close()

	customClient := &http.Client{Timeout: 20 * time.Millisecond} // Short timeout
	_, err := httpclient.FetchXML(context.Background(), customClient, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.Error(t, err)
	// The error message from httpclient.FetchXML includes "timed out"
	// and also the context.DeadlineExceeded error is wrapped.
	assert.Contains(t, err.Error(), "timed out")
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), context.DeadlineExceeded.Error()), "Error should wrap context.DeadlineExceeded")
}


func TestFetchXML_InvalidXMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `<Response><Say>Hello</Say><Play>audio.wav</Play></Response`) // Missing closing </Response>
	}))
	defer server.Close()

	_, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshaling XML response")
	// The actual error from xml.Unmarshal for this case is "XML syntax error on line 1: unexpected EOF"
	// or similar depending on exact parser behavior with partial input.
	// Checking for "Body:" is good as FetchXML includes it.
	assert.Contains(t, err.Error(), "Body:")
}

func TestFetchXML_NonXMLContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json") // Incorrect content type
		fmt.Fprintln(w, `<Response><Hangup/></Response>`)  // But valid XML body
	}))
	defer server.Close()

	// Logger with a buffer to check warnings could be used here if needed.
	// For now, just check successful parsing.
	respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.NoError(t, err) // Should still parse successfully
	assert.NotNil(t, respElement)
	assert.Len(t, respElement.Verbs, 1)
	_, ok := respElement.Verbs[0].(*domain.HangupElement)
	assert.True(t, ok)
}

func TestFetchXML_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Returns empty body, which is invalid XML for ResponseElement
		fmt.Fprint(w, "")
	}))
	defer server.Close()

	_, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshaling XML response")
	// xml.Unmarshal on empty string into a struct results in "EOF" error.
	assert.Contains(t, err.Error(), "EOF")
}

func TestFetchXML_OnlyXMLDeclaration(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`) // No Response tag
    }))
    defer server.Close()

    _, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "unmarshaling XML response")
    // Error will likely be about expecting <Response> but finding EOF or similar.
}

func TestFetchXML_NoVerbsInResponse(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, `<Response></Response>`) // Empty Response
    }))
    defer server.Close()

    respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())
    assert.NoError(t, err)
    assert.NotNil(t, respElement)
    assert.Empty(t, respElement.Verbs) // Verbs slice should be empty or nil
}

func TestFetchXML_POST_NoParams(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, http.MethodPost, r.Method)
        // For POST with nil params, Content-Type might not be set by httpclient, or body is empty.
        // The httpclient.FetchXML sets Content-Type only if params are present.
        // Standard http.Client might send Content-Length: 0.
        // This test ensures it doesn't fail.
        body, _ := io.ReadAll(r.Body)
        assert.Empty(t, body)
        fmt.Fprintln(w, `<Response><Say>POST No Params</Say></Response>`)
    }))
    defer server.Close()

    respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodPost, nil, nil, newTestLogger())

    assert.NoError(t, err)
    assert.NotNil(t, respElement)
    assert.Len(t, respElement.Verbs, 1)
    sayElement, ok := respElement.Verbs[0].(*domain.SayElement)
    assert.True(t, ok)
    assert.Equal(t, "POST No Params", sayElement.Text)
}

func TestFetchXML_WithAllVerbs(t *testing.T) {
	serverXML := `<Response>
	<Say>Test</Say>
	<Gather action="/gather_action" method="POST" timeout="10" finishOnKey="#" numDigits="5">
		<Play>enter_digits.wav</Play>
	</Gather>
	<Record action="/record_action" maxLength="30" playBeep="true" format="mp3"/>
	<Dial action="/dial_action" callerId="12345" timeoutSeconds="45" hangupOnStar="true">1234567890</Dial>
</Response>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintln(w, serverXML)
	}))
	defer server.Close()

	respElement, err := httpclient.FetchXML(context.Background(), nil, server.URL, http.MethodGet, nil, nil, newTestLogger())

	assert.NoError(t, err)
	assert.NotNil(t, respElement)
	if respElement == nil {
		t.FailNow() // Avoid nil pointer dereference below
	}
	assert.Len(t, respElement.Verbs, 4) // Say, Gather, Record, Dial

	// 1. SayElement
	sayElem, ok := respElement.Verbs[0].(*domain.SayElement)
	assert.True(t, ok, "Expected SayElement at index 0")
	if ok {
		assert.Equal(t, "Test", sayElem.Text)
	}

	// 2. GatherElement
	gatherElem, ok := respElement.Verbs[1].(*domain.GatherElement)
	assert.True(t, ok, "Expected GatherElement at index 1")
	if ok {
		assert.Equal(t, "/gather_action", gatherElem.ActionURL)
		assert.Equal(t, "POST", gatherElem.Method)
		assert.Equal(t, 10, gatherElem.TimeoutSeconds)
		assert.Equal(t, "#", gatherElem.FinishOnKey)
		assert.Equal(t, 5, gatherElem.NumDigits) // Assert numDigits
		assert.NotNil(t, gatherElem.Play, "Gather should have a nested Play element")
		if gatherElem.Play != nil {
			assert.Equal(t, "enter_digits.wav", gatherElem.Play.URL)
		}
		assert.Nil(t, gatherElem.Say, "Gather should not have a nested Say element in this test case")
	}

	// 3. RecordElement
	recordElem, ok := respElement.Verbs[2].(*domain.RecordElement)
	assert.True(t, ok, "Expected RecordElement at index 2")
	if ok {
		assert.Equal(t, "/record_action", recordElem.ActionURL)
		assert.Equal(t, 30, recordElem.MaxLengthSeconds)
		assert.True(t, recordElem.PlayBeep)
		assert.Equal(t, "mp3", recordElem.FileFormat) // Assert format
	}

	// 4. DialElement
	dialElem, ok := respElement.Verbs[3].(*domain.DialElement)
	assert.True(t, ok, "Expected DialElement at index 3")
	if ok {
		assert.Equal(t, "/dial_action", dialElem.ActionURL)
		assert.Equal(t, "1234567890", dialElem.CalleeIDToDial)
		assert.Equal(t, "12345", dialElem.CallerID)          // Assert callerId
		assert.Equal(t, 45, dialElem.TimeoutSeconds)     // Assert timeoutSeconds
		assert.True(t, dialElem.HangupOnStar)             // Assert hangupOnStar
	}
}
