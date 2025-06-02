package callcontrol

import (
	"fmt" 
	"github.com/fiorix/go-eventsocket/eventsocket"
	"github.com/sirupsen/logrus" 
)

type CallContext struct {
	FreeswitchUUID      string 
	AgbaraAccountSID    string 
	AgbaraCallSID       string 
	AnswerURL           string 
	FromNum             string
	ToNum               string 
	ChannelState        string
	Variables           map[string]string 
	ESLConnection       *eventsocket.Connection 
	Logger              *logrus.Entry 
}

func NewCallContext(connectEvent *eventsocket.Event, eslConn *eventsocket.Connection, baseLogger *logrus.Logger) (*CallContext, error) {
	fsUUID := connectEvent.Get("Channel-Call-UUID"); if fsUUID == "" { fsUUID = connectEvent.Get("Unique-ID") }
	if fsUUID == "" { return nil, fmt.Errorf("FS UUID missing") }
	
	agbaraCallSID := connectEvent.Get("variable_agbara_call_sid")
	callLogger := baseLogger.WithFields(logrus.Fields{ "fs_uuid": fsUUID, "call_sid": agbaraCallSID })

	ctx := &CallContext{
		FreeswitchUUID:   fsUUID, AgbaraAccountSID: connectEvent.Get("variable_agbara_account_sid"),
		AgbaraCallSID:    agbaraCallSID, AnswerURL: connectEvent.Get("variable_agbara_answer_url"),
		FromNum:          connectEvent.Get("Caller-Caller-ID-Number"), ToNum: connectEvent.Get("Caller-Destination-Number"),
		ChannelState:     connectEvent.Get("Channel-State"), Variables: make(map[string]string),
		ESLConnection:    eslConn, Logger: callLogger,
	}
	
	// Simplified header processing using connectEvent.Get(key)
	if connectEvent.Header != nil {
		for key := range connectEvent.Header {
			// Get() returns the first value for a header key, which is usually what's needed.
			// This avoids the compiler confusion with iterating over map[string][]string values directly.
			ctx.Variables[key] = connectEvent.Get(key)
		}
	}
	
	return ctx, nil
}
func (c *CallContext) GetFreeswitchUUID() string { return c.FreeswitchUUID }
func (c *CallContext) GetAgbaraCallSID() string  { return c.AgbaraCallSID }
func (c *CallContext) GetLoggerEntry() *logrus.Entry { return c.Logger }
