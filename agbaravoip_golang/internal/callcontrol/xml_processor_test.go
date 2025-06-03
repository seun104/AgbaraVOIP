package callcontrol_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/user/agbaravoip_golang/internal/callcontrol"
	"github.com/user/agbaravoip_golang/internal/domain"
)

// --- MockMinimalCallContext for xml_processor tests ---
type MockMinimalCallContext struct {
	mock.Mock
}

func (m *MockMinimalCallContext) Log() *logrus.Entry {
	args := m.Called()
	if args.Get(0) == nil {
		entry := logrus.NewEntry(logrus.New())
		entry.Logger.SetOutput(io.Discard)
		return entry
	}
	return args.Get(0).(*logrus.Entry)
}
func (m *MockMinimalCallContext) GetUuid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAccountSid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetApplicationSid() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAnswerURL() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetVariable(varName string) string { return m.Called(varName).String(0) }
func (m *MockMinimalCallContext) IsHangupInitiated() bool { return m.Called().Bool(0) }
func (m *MockMinimalCallContext) SetHangupInitiated() { m.Called() }

// --- MockRoundTripper for http.Client ---
type MockRoundTripper struct {
	mock.Mock
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// --- Test Setup Helper ---
func setupProcessorTest(t *testing.T) (*callcontrol.XMLProcessor, *MockMinimalCallContext, *MockRoundTripper, *logrus.Entry) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetOutput(io.Discard)

	mockCtx := new(MockMinimalCallContext)
	mockRT := new(MockRoundTripper)
	mockHTTPClient := &http.Client{Transport: mockRT}

	xmlProcessor := callcontrol.NewXMLProcessor(mockHTTPClient) // Updated constructor

	mockCtx.On("Log").Return(logger)
	mockCtx.On("GetUuid").Return("test-proc-uuid")

	return xmlProcessor, mockCtx, mockRT, logger
}

func TestXMLProcessor_FetchAndParseXML_Success(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)

	testURL := "http://example.com/test.xml"
	xmlResponseBody := `<Response><Say voice="alice">Hello</Say><Play loop="2">http://example.com/audio.wav</Play><Hangup/></Response>`
	
	r := io.NopCloser(bytes.NewReader([]byte(xmlResponseBody)))
	mockHTTPResponse := &http.Response{StatusCode: http.StatusOK, Body: r, Header: make(http.Header)}
	mockHTTPResponse.Header.Set("Content-Type", "application/xml") // Ensure content type

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL && req.Method == http.MethodGet
	})).Return(mockHTTPResponse, nil).Once()

	// FetchAndParseXML(ctx context.Context, callCtx domain.MinimalCallContext, urlStr string, method string, requestParams url.Values)
	elements, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, elements)
	assert.Len(t, elements, 3)
	if len(elements) == 3 {
		sayElem, ok := elements[0].(*domain.SayElement)
		assert.True(t, ok)
		assert.Equal(t, "Hello", sayElem.Text)
		assert.Equal(t, "alice", sayElem.Voice)

		playElem, ok := elements[1].(*domain.PlayElement)
		assert.True(t, ok)
		assert.Equal(t, "http://example.com/audio.wav", playElem.URL)
		assert.Equal(t, 2, playElem.Loop)

		_, okHangup := elements[2].(*domain.HangupElement)
		assert.True(t, okHangup)
	}
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestXMLProcessor_FetchAndParseXML_HttpError(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)
	testURL := "http://example.com/error.xml"

	mockHTTPResponse := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(bytes.NewReader([]byte("Internal Server Error"))),
		Header:     make(http.Header),
	}
	mockHTTPResponse.Header.Set("Content-Type", "text/plain")

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL
	})).Return(mockHTTPResponse, nil).Once() // httpclient.FetchXML expects error to be nil if resp is not nil

	_, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "httpclient.FetchXML failed") // Error from xml_processor
	assert.Contains(t, err.Error(), "failed with status 500")    // Error from httpclient
	assert.Contains(t, err.Error(), "Internal Server Error")
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestXMLProcessor_FetchAndParseXML_RoundTripError(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)
	testURL := "http://example.com/network-error.xml"
	expectedError := errors.New("network broke")

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL
	})).Return(nil, expectedError).Once()

	_, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "httpclient.FetchXML failed")
	assert.Contains(t, err.Error(), expectedError.Error())
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}


func TestXMLProcessor_FetchAndParseXML_InvalidXML(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)
	testURL := "http://example.com/invalid.xml"
	invalidXmlBody := `<Response><Say>Hello</Say><Play>audio.wav</Play></Response` // Missing closing Play

	r := io.NopCloser(bytes.NewReader([]byte(invalidXmlBody)))
	mockHTTPResponse := &http.Response{StatusCode: http.StatusOK, Body: r, Header: make(http.Header)}
	mockHTTPResponse.Header.Set("Content-Type", "application/xml")

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL
	})).Return(mockHTTPResponse, nil).Once()

	_, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "httpclient.FetchXML failed")
	assert.Contains(t, err.Error(), "unmarshaling XML response") // Error from httpclient.FetchXML
	assert.Contains(t, err.Error(), "Body:") // httpclient.FetchXML includes body in error
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestXMLProcessor_FetchAndParseXML_NoResponseTag(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)
	testURL := "http://example.com/noresponse.xml"
	xmlBodyNoResponse := `<Say>Just a Say verb, no Response wrapper</Say>`

	r := io.NopCloser(bytes.NewReader([]byte(xmlBodyNoResponse)))
	mockHTTPResponse := &http.Response{StatusCode: http.StatusOK, Body: r, Header: make(http.Header)}
	mockHTTPResponse.Header.Set("Content-Type", "application/xml")

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL
	})).Return(mockHTTPResponse, nil).Once()

	_, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "httpclient.FetchXML failed")
	assert.Contains(t, err.Error(), "unmarshaling XML response")
	// The error from xml.Unmarshal will be something like "expected element type <Response> but have <Say>"
	assert.Contains(t, err.Error(), "expected element type <Response>")
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}

func TestXMLProcessor_FetchAndParseXML_EmptyVerbs(t *testing.T) {
	xmlProcessor, mockCtx, mockRT, _ := setupProcessorTest(t)
	testURL := "http://example.com/emptyverbs.xml"
	xmlResponseBody := `<Response></Response>` // Empty but valid

	r := io.NopCloser(bytes.NewReader([]byte(xmlResponseBody)))
	mockHTTPResponse := &http.Response{StatusCode: http.StatusOK, Body: r, Header: make(http.Header)}
	mockHTTPResponse.Header.Set("Content-Type", "application/xml")

	mockRT.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
		return req.URL.String() == testURL
	})).Return(mockHTTPResponse, nil).Once()

	elements, err := xmlProcessor.FetchAndParseXML(context.Background(), mockCtx, testURL, http.MethodGet, nil)

	assert.NoError(t, err)
	assert.NotNil(t, elements)
	assert.Len(t, elements, 0) // Expect empty slice for Verbs
	mockRT.AssertExpectations(t)
	mockCtx.AssertExpectations(t)
}
