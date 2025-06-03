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
	// Extended XML to include more Dial verb variations and a top-level Conference
	serverXML := `
<Response>
    <Say>Test Say</Say>
    <Play>test.wav</Play>
    <Gather action="/gather_action" timeout="10" numDigits="5" finishOnKey="#">
        <Play>prompt.wav</Play>
    </Gather>
    <Record action="/record_action" maxLength="60" format="mp3" playBeep="true"/>
    <Dial action="/dial_action_simple_chardata">1234567890</Dial>
    <Dial action="/dial_action_number">
        <Number sendDigits="ww123">5551112222</Number>
    </Dial>
    <Dial action="/dial_action_conference" callerId="confCaller">
        <Conference muted="true" beep="true">meetingRoomAlpha</Conference>
    </Dial>
    <Dial action="/dial_action_sip">
        <Sip>sip:alice@example.com</Sip>
    </Dial>
    <Conference callbackUrl="/conf_events" muted="false" maxMembers="20" beep="true">myMainConference</Conference>
    <Hangup/>
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
		t.FailNow()
	}
	assert.Len(t, respElement.Verbs, 10) // Say, Play, Gather, Record, Dial(chardata), Dial(Number), Dial(Conference), Dial(Sip), Conference, Hangup

	// Index 0: Say
	sayElem, ok := respElement.Verbs[0].(*domain.SayElement)
	assert.True(t, ok, "Expected SayElement at index 0"); if ok { assert.Equal(t, "Test Say", sayElem.Text) }

	// Index 1: Play
	playElem, ok := respElement.Verbs[1].(*domain.PlayElement)
	assert.True(t, ok, "Expected PlayElement at index 1"); if ok { assert.Equal(t, "test.wav", playElem.URL) }

	// Index 2: Gather
	gatherElem, ok := respElement.Verbs[2].(*domain.GatherElement)
	assert.True(t, ok, "Expected GatherElement at index 2")
	if ok {
		assert.Equal(t, "/gather_action", gatherElem.ActionURL)
		assert.Equal(t, 10, gatherElem.TimeoutSeconds)
		assert.Equal(t, 5, gatherElem.NumDigits)
		assert.Equal(t, "#", gatherElem.FinishOnKey)
		assert.NotNil(t, gatherElem.Play); if gatherElem.Play != nil { assert.Equal(t, "prompt.wav", gatherElem.Play.URL) }
	}

	// Index 3: Record
	recordElem, ok := respElement.Verbs[3].(*domain.RecordElement)
	assert.True(t, ok, "Expected RecordElement at index 3")
	if ok {
		assert.Equal(t, "/record_action", recordElem.ActionURL)
		assert.Equal(t, 60, recordElem.MaxLengthSeconds)
		assert.Equal(t, "mp3", recordElem.FileFormat)
		assert.True(t, recordElem.PlayBeep)
	}

	// Index 4: Dial (chardata)
	dialSimple, ok := respElement.Verbs[4].(*domain.DialElement)
	assert.True(t, ok, "Expected DialElement (simple chardata) at index 4")
	if ok {
		assert.Equal(t, "/dial_action_simple_chardata", dialSimple.ActionURL)
		assert.Equal(t, "1234567890", dialSimple.CalleeIDToDial)
		assert.Nil(t, dialSimple.Number)
		assert.Nil(t, dialSimple.NestedConference)
		assert.Nil(t, dialSimple.Sip)
	}

	// Index 5: Dial (Number)
	dialWithNumber, ok := respElement.Verbs[5].(*domain.DialElement)
	assert.True(t, ok, "Expected DialElement (with Number) at index 5")
	if ok {
		assert.Equal(t, "/dial_action_number", dialWithNumber.ActionURL)
		assert.NotNil(t, dialWithNumber.Number, "Dial.Number should not be nil")
		if dialWithNumber.Number != nil {
			assert.Equal(t, "5551112222", dialWithNumber.Number.PhoneNumber)
			assert.Equal(t, "ww123", dialWithNumber.Number.SendDigits)
		}
		assert.Empty(t, dialWithNumber.CalleeIDToDial, "CalleeIDToDial should be empty when Number is present")
		assert.Nil(t, dialWithNumber.NestedConference)
		assert.Nil(t, dialWithNumber.Sip)
	}

	// Index 6: Dial (Conference)
	dialWithConf, ok := respElement.Verbs[6].(*domain.DialElement)
	assert.True(t, ok, "Expected DialElement (with Conference) at index 6")
	if ok {
		assert.Equal(t, "/dial_action_conference", dialWithConf.ActionURL)
		assert.Equal(t, "confCaller", dialWithConf.CallerID)
		assert.NotNil(t, dialWithConf.NestedConference, "Dial.NestedConference should not be nil")
		if dialWithConf.NestedConference != nil {
			assert.Equal(t, "meetingRoomAlpha", dialWithConf.NestedConference.RoomName)
			assert.True(t, dialWithConf.NestedConference.Muted)
			assert.True(t, dialWithConf.NestedConference.Beep)
		}
		assert.Empty(t, dialWithConf.CalleeIDToDial)
		assert.Nil(t, dialWithConf.Number)
		assert.Nil(t, dialWithConf.Sip)
	}

	// Index 7: Dial (Sip)
	dialWithSip, ok := respElement.Verbs[7].(*domain.DialElement)
	assert.True(t, ok, "Expected DialElement (with Sip) at index 7")
	if ok {
		assert.Equal(t, "/dial_action_sip", dialWithSip.ActionURL)
		assert.NotNil(t, dialWithSip.Sip, "Dial.Sip should not be nil")
		if dialWithSip.Sip != nil {
			assert.Equal(t, "sip:alice@example.com", dialWithSip.Sip.URI)
		}
		assert.Empty(t, dialWithSip.CalleeIDToDial)
		assert.Nil(t, dialWithSip.Number)
		assert.Nil(t, dialWithSip.NestedConference)
	}

	// Index 8: Top-level Conference
	confElem, ok := respElement.Verbs[8].(*domain.ConferenceElement)
	assert.True(t, ok, "Expected ConferenceElement at index 8")
	if ok {
		assert.Equal(t, "myMainConference", confElem.RoomName)
		assert.Equal(t, "/conf_events", confElem.CallbackURL)
	}

	// Index 9: Hangup
	_, okHangup := respElement.Verbs[9].(*domain.HangupElement)
	assert.True(t, okHangup, "Expected HangupElement at index 9")
}
