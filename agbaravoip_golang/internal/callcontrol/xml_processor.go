package callcontrol
import ( "encoding/xml"; "errors"; "fmt"; "io/ioutil"; "net/http"; "net/url";
	"github.com/user/agbaravoip_golang/internal/domain"; "github.com/sirupsen/logrus" )
type XMLProcessor struct { logger *logrus.Entry; httpClient *http.Client }
func NewXMLProcessor(logger *logrus.Logger, client *http.Client) *XMLProcessor {
	return &XMLProcessor{ logger: logger.WithField("component", "xml_processor"), httpClient: client } }
func (p *XMLProcessor) FetchAndParseXML(ctx *CallContext) ([]domain.CallControlElement, error) {
	if ctx.AnswerURL == "" { return nil, errors.New("answer URL is empty") }
	queryParams := url.Values{}; queryParams.Add("CallSid", ctx.AgbaraCallSID); queryParams.Add("AccountSid", ctx.AgbaraAccountSID); queryParams.Add("From", ctx.FromNum); queryParams.Add("To", ctx.ToNum)
	if dir, ok := ctx.Variables["direction"]; ok { queryParams.Add("Direction", dir) } // Corrected key
	if callStatus, ok := ctx.Variables["call_status"]; ok { queryParams.Add("CallStatus", callStatus) } // Corrected key
	requestURL := ctx.AnswerURL; if len(queryParams) > 0 { requestURL += "?" + queryParams.Encode() }
	p.logger.Infof("Fetching XML from %s for %s", requestURL, ctx.AgbaraCallSID)
	req, err := http.NewRequest("GET", requestURL, nil); if err != nil { return nil, fmt.Errorf("create http req failed: %w", err) }
	resp, err := p.httpClient.Do(req); if err != nil { return nil, fmt.Errorf("http request failed: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { bodyBytes, _ := ioutil.ReadAll(resp.Body); p.logger.Debugf("Error body: %s", string(bodyBytes)); return nil, fmt.Errorf("http status %d", resp.StatusCode) }
	xmlData, err := ioutil.ReadAll(resp.Body); if err != nil { return nil, fmt.Errorf("read body failed: %w", err) }
	p.logger.Debugf("Received XML for %s: %s", ctx.AgbaraCallSID, string(xmlData))
	var responsePayload domain.ResponseElement
	if err := xml.Unmarshal(xmlData, &responsePayload); err != nil { return nil, fmt.Errorf("xml unmarshal failed: %w", err) }
	if responsePayload.XMLName.Local != "Response" { return nil, errors.New("xml root not <Response>") }
	parsedElements := make([]domain.CallControlElement, 0, len(responsePayload.Verbs))
	for _, verbInterface := range responsePayload.Verbs {
		var elem domain.CallControlElement
		switch v := verbInterface.(type) {
		case domain.SayElement: el := v; elem = &el; case domain.PlayElement: el := v; elem = &el
		case domain.HangupElement: el := v; elem = &el; case domain.PauseElement: el := v; elem = &el
		case domain.RedirectElement: el := v; elem = &el
		default: p.logger.Warnf("Unsupported XML element: %T for %s", v, ctx.AgbaraCallSID); continue
		}
		parsedElements = append(parsedElements, elem)
	}
	responsePayload.Elements = parsedElements
	if len(responsePayload.Elements) == 0 && len(xmlData) > 50 { p.logger.Warnf("XML parsed to zero elements. XML: %s", string(xmlData)) }
	p.logger.Infof("Fetched & parsed %d verbs for %s", len(responsePayload.Elements), ctx.AgbaraCallSID)
	return responsePayload.Elements, nil
}

