package domain_test 
import ( "errors"; "testing"; "strings"; "io"; "net/textproto";
	"github.com/user/agbaravoip_golang/internal/domain";
	"github.com/fiorix/go-eventsocket/eventsocket";
	"github.com/sirupsen/logrus"; "github.com/stretchr/testify/assert"; "github.com/stretchr/testify/mock" )
type MockMinimalCallContext struct { mock.Mock }
func (m *MockMinimalCallContext) GetFreeswitchUUID() string { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetAgbaraCallSID() string  { return m.Called().String(0) }
func (m *MockMinimalCallContext) GetLoggerEntry() *logrus.Entry { ret := m.Called().Get(0); if ret == nil { return nil }; return ret.(*logrus.Entry) }

// Define an interface that MockESLConnection and *eventsocket.Connection both satisfy for Execute methods
type EslConnectionExecutor interface {
    Execute(app string, arg string, lock bool) (*eventsocket.Event, error)
    Send(cmd string, args ...string) (*eventsocket.Event, error)
    // Add other methods from eventsocket.Connection that your Execute methods use
}
type MockESLConnection struct { mock.Mock }
func (m *MockESLConnection) Send(cmd string, cmdArgs ...string) (*eventsocket.Event, error) { var iargs []interface{}; iargs = append(iargs, cmd); for _,a := range cmdArgs {iargs = append(iargs,a)}; ca := m.Called(iargs...); if ca.Get(0) == nil { return nil, ca.Error(1) }; return ca.Get(0).(*eventsocket.Event), ca.Error(1) }
func (m *MockESLConnection) Execute(app string, arg string, lock bool) (*eventsocket.Event, error) { args := m.Called(app, arg, lock); if args.Get(0) == nil { return nil, args.Error(1) }; return args.Get(0).(*eventsocket.Event), args.Error(1) }

type MockCallService struct { mock.Mock } // Implements services.ICallService
func (m *MockCallService) OriginateCall(a string,b *string,c,d,e string,f *int) (*domain.Call,error) { return nil,nil }
func (m *MockCallService) GetCallBySID(a,b string) (*domain.Call,error) { return nil,nil }
func (m *MockCallService) ListCalls(a string,f map[string]interface{}) ([]*domain.Call,error) { return nil,nil }
func (m *MockCallService) UpdateCallStatus(a string,b domain.CallStatus,c string,d int) (*domain.Call,error) { return nil,nil }

func setupTestMocks(t *testing.T) (*logrus.Entry, *MockMinimalCallContext, EslConnectionExecutor, interface{}) { // Return EslConnectionExecutor
	logger := logrus.NewEntry(logrus.New()); logger.Logger.SetOutput(io.Discard) 
	mockCtx := new(MockMinimalCallContext); mockEsl := new(MockESLConnection); mockCallSvc := new(MockCallService)
	mockCtx.On("GetAgbaraCallSID").Return("CAtest123"); mockCtx.On("GetLoggerEntry").Return(logger) 
	return logger, mockCtx, mockEsl, mockCallSvc
}
func TestSayElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	say := domain.SayElement{Text: "Hello", Loop: 1}
	mockEsl.(*MockESLConnection).On("Execute", "speak", "flite|slt|Hello", true).Return(&eventsocket.Event{Header: textproto.MIMEHeader{"Reply-Text":{"ok"}}}, nil).Once()
	result := say.Execute(mockCtx, mockEsl, mockCallSvc); assert.Equal(t, domain.ActionContinue, result.Action); mockEsl.(*MockESLConnection).AssertExpectations(t)
}
// ... (other tests similarly changed to use mockEsl as EslConnectionExecutor, and Header: textproto.MIMEHeader) ...
func TestHangupElement_Execute(t *testing.T) {
	_, mockCtx, mockEsl, mockCallSvc := setupTestMocks(t)
	hangup := domain.HangupElement{}
	mockEsl.(*MockESLConnection).On("Execute", "hangup", "NORMAL_CLEARING", true).Return(&eventsocket.Event{Header: textproto.MIMEHeader{"Reply-Text":{"+OK"}}}, nil).Once()
	result := hangup.Execute(mockCtx, mockEsl, mockCallSvc); assert.Equal(t, domain.ActionHangup, result.Action); mockEsl.(*MockESLConnection).AssertExpectations(t)
}


