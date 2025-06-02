package callcontrol_test

import (
	"bytes"
	"io" // For io.NopCloser
	"io/ioutil"
	"net/http"
	"strings" // For mock.MatchedBy
	"testing"
	"net/url" 

	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/fiorix/go-eventsocket/eventsocket" 
)

type MockRoundTripper struct { mock.Mock }
func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*http.Response), args.Error(1)
}

func TestXMLProcessor_FetchAndParseXML_Success(t *testing.T) {
	logger := logrus.New(); logger.SetLevel(logrus.PanicLevel) 
	mockRT := new(MockRoundTripper)
	mockHTTPClient := &http.Client{Transport: mockRT}
	xmlProcessor := callcontrol.NewXMLProcessor(logger, mockHTTPClient)

	dummyEslConn := &eventsocket.Connection{} 

	testCallCtx := &callcontrol.CallContext{
		AgbaraCallSID: "CAtest123", AgbaraAccountSID: "ACtestacc", FromNum: "1000", ToNum: "2000",
		AnswerURL: "http://example.com/test.xml", FreeswitchUUID: "fs-uuid-123",
		Variables: map[string]string{"variable_direction": "outbound", "variable_callstatus": "ringing"},
		ESLConnection: dummyEslConn, Logger: logger.WithField("test", "fetchparse"),
	}

	// Corrected XML string
	xmlResponseBody := `<Response><Say voice="alice">Hello</Say><Play loop="2">http://example.com/audio.wav</Play><Hangup/></Response>`
	
	r := ioutil.NopCloser(bytes.NewReader([]byte(xmlResponseBody))) // Use io.NopCloser for Go 1.16+
	mockHTTPClientResponse := &http.Response{ StatusCode: http.StatusOK, Body: r, Header: make(http.Header) }
	
	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		expectedURLBase := "http://example.com/test.xml"
		assert.Contains(t, req.URL.String(), expectedURLBase)
		q := req.URL.Query(); assert.Equal(t, "CAtest123", q.Get("CallSid"))
		return strings.HasPrefix(req.URL.String(), expectedURLBase)
	})).Return(mockHTTPClientResponse, nil).Once()

	elements, err := xmlProcessor.FetchAndParseXML(testCallCtx)
	
	assert.NoError(t, err); assert.NotNil(t, elements); assert.Len(t, elements, 3)
	if len(elements) == 3 {
		sayElem, ok := elements[0].(*domain.SayElement); assert.True(t, ok); assert.Equal(t, "Hello", sayElem.Text)
		playElem, ok := elements[1].(*domain.PlayElement); assert.True(t, ok); assert.Equal(t, "http://example.com/audio.wav", playElem.URL)
		_, okHangup := elements[2].(*domain.HangupElement); assert.True(t, okHangup)
	}
	mockRT.AssertExpectations(t)
}
func TestXMLProcessor_FetchAndParseXML_HttpError(t *testing.T) { t.Skip("HTTP error test not fully implemented yet") }
func TestXMLProcessor_FetchAndParseXML_InvalidXML(t *testing.T) { t.Skip("Invalid XML test not fully implemented yet") }
func TestXMLProcessor_FetchAndParseXML_NoResponseTag(t *testing.T) { t.Skip("No Response tag test not fully implemented yet") }

