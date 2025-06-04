package twiml

import (
	"strings"
	"testing"
)

func TestNewResponse(t *testing.T) {
	response := NewResponse()
	if response == nil {
		t.Fatal("NewResponse() returned nil")
	}
	if len(response.Verbs) != 0 {
		t.Errorf("Expected empty verbs slice, got %d verbs", len(response.Verbs))
	}
}

func TestResponseAdd(t *testing.T) {
	response := NewResponse()
	say := &Say{Text: "Hello World"}
	
	response.Add(say)
	
	if len(response.Verbs) != 1 {
		t.Errorf("Expected 1 verb, got %d", len(response.Verbs))
	}
}

func TestResponseRender(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Response
		contains []string
	}{
		{
			name: "empty response",
			setup: func() *Response {
				return NewResponse()
			},
			contains: []string{"<?xml version=\"1.0\" encoding=\"UTF-8\"?>", "<Response>", "</Response>"},
		},
		{
			name: "response with say verb",
			setup: func() *Response {
				response := NewResponse()
				response.Add(&Say{Text: "Hello World"})
				return response
			},
			contains: []string{"<Response>", "<Say>Hello World</Say>", "</Response>"},
		},
		{
			name: "response with multiple verbs",
			setup: func() *Response {
				response := NewResponse()
				response.Add(&Say{Text: "Hello"})
				response.Add(&Hangup{})
				return response
			},
			contains: []string{"<Say>Hello</Say>", "<Hangup></Hangup>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := tt.setup()
			xml, err := response.Render()
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			
			for _, expected := range tt.contains {
				if !strings.Contains(xml, expected) {
					t.Errorf("Expected XML to contain %q, got:\n%s", expected, xml)
				}
			}
		})
	}
}

func TestSayVerb(t *testing.T) {
	tests := []struct {
		name     string
		say      Say
		expected []string
	}{
		{
			name: "basic say",
			say:  Say{Text: "Hello World"},
			expected: []string{"<Say>Hello World</Say>"},
		},
		{
			name: "say with voice",
			say:  Say{Text: "Hello", Voice: "alice"},
			expected: []string{"<Say voice=\"alice\">Hello</Say>"},
		},
		{
			name: "say with language and loop",
			say:  Say{Text: "Hola", Language: "es-MX", Loop: 2},
			expected: []string{"<Say language=\"es-MX\" loop=\"2\">Hola</Say>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml, err := ToXML(&tt.say)
			if err != nil {
				t.Fatalf("ToXML() error = %v", err)
			}
			
			for _, expected := range tt.expected {
				if !strings.Contains(xml, expected) {
					t.Errorf("Expected XML to contain %q, got:\n%s", expected, xml)
				}
			}
		})
	}
}

func TestSayGetVerbName(t *testing.T) {
	say := &Say{}
	if say.GetVerbName() != "Say" {
		t.Errorf("Expected verb name 'Say', got %q", say.GetVerbName())
	}
}

func TestPlayVerb(t *testing.T) {
	tests := []struct {
		name     string
		play     Play
		expected []string
	}{
		{
			name: "basic play",
			play: Play{URL: "http://example.com/audio.mp3"},
			expected: []string{"<Play>http://example.com/audio.mp3</Play>"},
		},
		{
			name: "play with loop",
			play: Play{URL: "http://example.com/audio.mp3", Loop: 3},
			expected: []string{"<Play loop=\"3\">http://example.com/audio.mp3</Play>"},
		},
		{
			name: "play with digits",
			play: Play{URL: "http://example.com/audio.mp3", Digits: "1234"},
			expected: []string{"<Play digits=\"1234\">http://example.com/audio.mp3</Play>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xml, err := ToXML(&tt.play)
			if err != nil {
				t.Fatalf("ToXML() error = %v", err)
			}
			
			for _, expected := range tt.expected {
				if !strings.Contains(xml, expected) {
					t.Errorf("Expected XML to contain %q, got:\n%s", expected, xml)
				}
			}
		})
	}
}

func TestRecordVerb(t *testing.T) {
	record := Record{
		Action:      "http://example.com/record",
		Method:      "POST",
		MaxLength:   30,
		FinishOnKey: "#",
		PlayBeep:    true,
	}
	
	xml, err := ToXML(&record)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Record",
		"action=\"http://example.com/record\"",
		"method=\"POST\"",
		"maxLength=\"30\"",
		"finishOnKey=\"#\"",
		"playBeep=\"true\"",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestHangupVerb(t *testing.T) {
	hangup := &Hangup{}
	
	if hangup.GetVerbName() != "Hangup" {
		t.Errorf("Expected verb name 'Hangup', got %q", hangup.GetVerbName())
	}
	
	xml, err := ToXML(hangup)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	if !strings.Contains(xml, "<Hangup></Hangup>") {
		t.Errorf("Expected XML to contain <Hangup></Hangup>, got:\n%s", xml)
	}
}

func TestRedirectVerb(t *testing.T) {
	redirect := Redirect{
		URL:    "http://example.com/redirect",
		Method: "POST",
	}
	
	xml, err := ToXML(&redirect)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Redirect method=\"POST\">http://example.com/redirect</Redirect>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestDialVerb(t *testing.T) {
	dial := Dial{
		Action:       "http://example.com/dial",
		Method:       "POST",
		Timeout:      30,
		CallerId:     "+1234567890",
		HangupOnStar: true,
	}
	
	xml, err := ToXML(&dial)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Dial",
		"action=\"http://example.com/dial\"",
		"method=\"POST\"",
		"timeout=\"30\"",
		"callerId=\"+1234567890\"",
		"hangupOnStar=\"true\"",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestDialWithTargets(t *testing.T) {
	dial := &Dial{}
	number := &Number{PhoneNumber: "+1234567890"}
	sip := &Sip{SipURI: "sip:user@example.com"}
	
	dial.AddTarget(number)
	dial.AddTarget(sip)
	
	if len(dial.Targets) != 2 {
		t.Errorf("Expected 2 targets, got %d", len(dial.Targets))
	}
	
	xml, err := ToXML(dial)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Number>+1234567890</Number>",
		"<Sip>sip:user@example.com</Sip>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestNumberVerb(t *testing.T) {
	number := Number{
		PhoneNumber:  "+1234567890",
		SendDigits:   "1234",
		Url:          "http://example.com/number",
		Method:       "POST",
	}
	
	xml, err := ToXML(&number)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Number",
		"sendDigits=\"1234\"",
		"url=\"http://example.com/number\"",
		"method=\"POST\"",
		">+1234567890</Number>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestSipVerb(t *testing.T) {
	sip := Sip{
		SipURI:   "sip:user@example.com",
		Username: "testuser",
		Password: "testpass",
		Headers:  "X-Custom=value",
	}
	
	xml, err := ToXML(&sip)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Sip",
		"username=\"testuser\"",
		"password=\"testpass\"",
		"headers=\"X-Custom=value\"",
		">sip:user@example.com</Sip>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestConferenceVerb(t *testing.T) {
	conference := Conference{
		Name:                   "MyConference",
		Muted:                  true,
		Beep:                   "onEnter",
		StartConferenceOnEnter: true,
		MaxParticipants:        10,
	}
	
	xml, err := ToXML(&conference)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Conference",
		"muted=\"true\"",
		"beep=\"onEnter\"",
		"startConferenceOnEnter=\"true\"",
		"maxParticipants=\"10\"",
		">MyConference</Conference>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestGatherVerb(t *testing.T) {
	gather := Gather{
		Action:      "http://example.com/gather",
		Method:      "POST",
		Timeout:     10,
		NumDigits:   4,
		FinishOnKey: "#",
	}
	
	xml, err := ToXML(&gather)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Gather",
		"action=\"http://example.com/gather\"",
		"method=\"POST\"",
		"timeout=\"10\"",
		"numDigits=\"4\"",
		"finishOnKey=\"#\"",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestGatherWithNestedVerbs(t *testing.T) {
	gather := &Gather{
		Action: "http://example.com/gather",
	}
	
	say := &Say{Text: "Please enter your PIN"}
	pause := &Pause{Length: 1}
	
	gather.AddNestedVerb(say)
	gather.AddNestedVerb(pause)
	
	if len(gather.NestedVerbs) != 2 {
		t.Errorf("Expected 2 nested verbs, got %d", len(gather.NestedVerbs))
	}
	
	xml, err := ToXML(gather)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	expected := []string{
		"<Gather",
		"<Say>Please enter your PIN</Say>",
		"<Pause length=\"1\"></Pause>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestPauseVerb(t *testing.T) {
	pause := Pause{Length: 5}
	
	if pause.GetVerbName() != "Pause" {
		t.Errorf("Expected verb name 'Pause', got %q", pause.GetVerbName())
	}
	
	xml, err := ToXML(&pause)
	if err != nil {
		t.Fatalf("ToXML() error = %v", err)
	}
	
	if !strings.Contains(xml, "<Pause length=\"5\"></Pause>") {
		t.Errorf("Expected XML to contain <Pause length=\"5\"></Pause>, got:\n%s", xml)
	}
}

func TestRenderVerbs(t *testing.T) {
	say := &Say{Text: "Hello"}
	hangup := &Hangup{}
	
	xml, err := RenderVerbs(say, hangup)
	if err != nil {
		t.Fatalf("RenderVerbs() error = %v", err)
	}
	
	expected := []string{
		"<Say>Hello</Say>",
		"<Hangup></Hangup>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestComplexTwiMLDocument(t *testing.T) {
	// Test a complex TwiML document with nested structures
	response := NewResponse()
	
	// Add a Say verb
	response.Add(&Say{Text: "Welcome to our service", Voice: "alice"})
	
	// Add a Gather with nested verbs
	gather := &Gather{
		Action:      "http://example.com/process",
		Method:      "POST",
		NumDigits:   1,
		FinishOnKey: "#",
	}
	gather.AddNestedVerb(&Say{Text: "Press 1 for sales, 2 for support"})
	gather.AddNestedVerb(&Pause{Length: 2})
	response.Add(gather)
	
	// Add a Dial with multiple targets
	dial := &Dial{Timeout: 30}
	dial.AddTarget(&Number{PhoneNumber: "+1234567890"})
	dial.AddTarget(&Sip{SipURI: "sip:support@example.com"})
	response.Add(dial)
	
	// Add a final hangup
	response.Add(&Hangup{})
	
	xml, err := response.Render()
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	
	// Verify the structure
	expected := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<Response>",
		"<Say voice=\"alice\">Welcome to our service</Say>",
		"<Gather action=\"http://example.com/process\" method=\"POST\" numDigits=\"1\" finishOnKey=\"#\">",
		"<Say>Press 1 for sales, 2 for support</Say>",
		"<Pause length=\"2\"></Pause>",
		"</Gather>",
		"<Dial timeout=\"30\">",
		"<Number>+1234567890</Number>",
		"<Sip>sip:support@example.com</Sip>",
		"</Dial>",
		"<Hangup></Hangup>",
		"</Response>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestVerbNames(t *testing.T) {
	tests := []struct {
		verb     Verb
		expected string
	}{
		{&Say{}, "Say"},
		{&Play{}, "Play"},
		{&Record{}, "Record"},
		{&Hangup{}, "Hangup"},
		{&Redirect{}, "Redirect"},
		{&Dial{}, "Dial"},
		{&Number{}, "Number"},
		{&Sip{}, "Sip"},
		{&Conference{}, "Conference"},
		{&Gather{}, "Gather"},
		{&Pause{}, "Pause"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.verb.GetVerbName(); got != tt.expected {
				t.Errorf("GetVerbName() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRenderVerbsWithError(t *testing.T) {
	// Test RenderVerbs with multiple verbs to increase coverage
	say1 := &Say{Text: "First message"}
	say2 := &Say{Text: "Second message"}
	say3 := &Say{Text: "Third message"}
	
	xml, err := RenderVerbs(say1, say2, say3)
	if err != nil {
		t.Fatalf("RenderVerbs() error = %v", err)
	}
	
	expected := []string{
		"<Say>First message</Say>",
		"<Say>Second message</Say>",
		"<Say>Third message</Say>",
	}
	
	for _, exp := range expected {
		if !strings.Contains(xml, exp) {
			t.Errorf("Expected XML to contain %q, got:\n%s", exp, xml)
		}
	}
}

func TestToXMLErrorHandling(t *testing.T) {
	// Test ToXML with different verb types to increase coverage
	verbs := []Verb{
		&Say{Text: "Test"},
		&Play{URL: "http://example.com/test.mp3"},
		&Record{Action: "http://example.com/record"},
		&Hangup{},
		&Redirect{URL: "http://example.com/redirect"},
	}
	
	for _, verb := range verbs {
		xml, err := ToXML(verb)
		if err != nil {
			t.Errorf("ToXML() error = %v for verb %s", err, verb.GetVerbName())
		}
		if xml == "" {
			t.Errorf("ToXML() returned empty string for verb %s", verb.GetVerbName())
		}
	}
}