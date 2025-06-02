package callcontrol

import (
	"encoding/xml"
	"fmt"
	"reflect" // To help map parsed <Verb> tags to actual CallControlElement structs

	"github.com/user/agbaravoip_golang/internal/domain"
	"github.com/sirupsen/logrus" // For logging
)

// AgbaraXMLParser holds dependencies for parsing AgbaraXML.
type AgbaraXMLParser struct {
	logger *logrus.Entry
	// verbRegistry can map XML tag names (string) to reflect.Type of the corresponding struct
	verbRegistry map[string]reflect.Type
}

// NewAgbaraXMLParser creates a new parser.
func NewAgbaraXMLParser(logger *logrus.Logger) *AgbaraXMLParser {
	logEntry := logger.WithField("component", "xml_parser")
	
	registry := make(map[string]reflect.Type)
	// Register known verb elements. This links the XML tag name to the Go struct type.
	registry["Say"] = reflect.TypeOf(domain.SayElement{})
	registry["Play"] = reflect.TypeOf(domain.PlayElement{})
	registry["Hangup"] = reflect.TypeOf(domain.HangupElement{})
	registry["Pause"] = reflect.TypeOf(domain.PauseElement{})
	registry["Redirect"] = reflect.TypeOf(domain.RedirectElement{})
	// TODO: Register other elements like Gather, Record, Dial as they are defined.

	return &AgbaraXMLParser{
		logger:       logEntry,
		verbRegistry: registry,
	}
}

// Parse takes an XML byte array and unmarshals it into a domain.ResponseElement,
// then populates its .Elements field with concrete CallControlElement implementations.
func (p *AgbaraXMLParser) Parse(xmlData []byte) (*domain.ResponseElement, error) {
	var response domain.ResponseElement
	
	// First, unmarshal into ResponseElement. This will populate response.Verbs
	// with a slice of xml.Name and chardata, or actual structs if direct mapping occurs.
	// Using response.Verbs []interface{} `xml:",any"` is a common way to capture mixed content.
	// However, a more direct approach is to use a custom UnmarshalXML or rely on how `encoding/xml`
	// handles slices of a specific type if all children are known.
	// For now, we will use a temporary struct that mirrors ResponseElement but has specific verb fields.
	// This is more robust for `encoding/xml`.

	// Temp struct for unmarshalling to handle specific verb types directly
	// This list needs to be exhaustive for all supported verbs.
	tempResponse := struct {
		XMLName  xml.Name                `xml:"Response"`
		Say      []*domain.SayElement      `xml:"Say"`
		Play     []*domain.PlayElement     `xml:"Play"`
		Hangup   []*domain.HangupElement   `xml:"Hangup"`
		Pause    []*domain.PauseElement    `xml:"Pause"`
		Redirect []*domain.RedirectElement `xml:"Redirect"`
		// Add other verb slices here: Gather, Record, Dial etc.
		// The order of elements in the XML is preserved by iterating through the fields
		// of tempResponse in the order they are defined, or by a more complex custom unmarshaller.
		// For simplicity now, we assume the order of parsing based on field iteration is acceptable,
		// OR we use a custom unmarshaller for ResponseElement.
		// For now, let's stick to `xml:",any"` and post-process.
	}{}


	// Unmarshal into the temporary structure that has specific fields for each verb.
	// This doesn't work as desired because `encoding/xml` won't preserve order across different verb types.
	// The `xml:",any"` approach with post-processing is more flexible for ordered mixed content.
	// So, we stick to the original ResponseElement and process `response.Verbs`.

	err := xml.Unmarshal(xmlData, &response)
	if err != nil {
		p.logger.Errorf("XML unmarshal error: %v", err)
		return nil, fmt.Errorf("failed to unmarshal AgbaraXML: %w", err)
	}

	if response.XMLName.Local != "Response" {
		return nil, fmt.Errorf("invalid AgbaraXML: root element must be <Response>, got <%s>", response.XMLName.Local)
	}

	// Post-process response.Verbs to populate response.Elements
	// response.Verbs will contain a mix of xml.Token (like xml.StartElement, xml.CharData, xml.EndElement)
	// This is complex. A simpler way for `xml:",any"` is if it produces a slice of structs
	// that can then be type-asserted.
	// The `encoding/xml` package, when using `xml:",any"`, actually populates the slice
	// with pointers to structs of the types that were unmarshalled, if they are known types
	// (i.e. if their XML names match registered types or struct field names).
	// Let's test this behavior. If `response.Verbs` contains actual structs, we can type assert.

	response.Elements = make([]domain.CallControlElement, 0, len(response.Verbs))
	for _, verbInterface := range response.Verbs {
		if elem, ok := verbInterface.(domain.CallControlElement); ok {
			response.Elements = append(response.Elements, elem)
		} else {
			// This might happen if there's chardata between elements that isn't captured
			// into a specific field, or if an unknown verb is encountered.
			// For now, we only add elements that successfully unmarshal into a known CallControlElement type.
			// This requires that the structs (SayElement, PlayElement, etc.) are directly unmarshalled
			// by `xml:",any"`. This happens if their `xml.Name` (from struct tag or type name) matches.
			// For example, if we have `<Say>...</Say>`, `xml:",any"` should put a `*SayElement` in `response.Verbs`.
			p.logger.Debugf("Skipping non-CallControlElement item from XML parse: Type %T, Value: %v", verbInterface, verbInterface)
		}
	}
	
	if len(response.Elements) == 0 && len(xmlData) > 50 { // Arbitrary length to check if it was non-empty XML
		p.logger.Warnf("XML parsed into zero CallControlElements. XML data might not match known verb structs or was empty. Verbs slice content: %+v", response.Verbs)
	}


	p.logger.Infof("Parsed %d AgbaraXML elements from response.", len(response.Elements))
	return &response, nil
}
