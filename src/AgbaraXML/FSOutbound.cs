using System;
using System.Collections.Generic;
using System.Collections;
using System.Collections.Concurrent;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Net;
using System.Net.Sockets;
using System.Xml.Linq;
using Emmanuel.AgbaraVOIP.Freeswitch;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.AgbaraXML
{
    public class FSOutbound :OutboundClient
    {
    /*An instance of this class is created every time an incoming call is received.
    The instance requests for a XML element set to execute the call and acts as a
    bridge between Event_Socket and the web application*/
    private const int MAX_REDIRECT = 999;
    private string[] WAIT_FOR_ACTIONS = new string[8] {"playback","record","play_and_get_digits","bridge","sleep","speak","conference","park"};
    private string[] NO_ANSWER_ELEMENTS =new string[4]{"Pause","PreAnswer","Reject","Dial"};
    private string[] VALID_ELEMENTS = new string[13] { "Pause", "PreAnswer", "Hangup", "Dial","Conference","Number","Gather","Say","Record","Play","Redirect","SMS","Reject" };
    private readonly Dictionary<string, Type> _elementTypes = new Dictionary<string, Type>();
    private string default_hangup_url;
    private string default_http_method;
    private string extra_fs_vars;
    private bool answered;
    private string xml_response;
    private List<Element> parsed_element = new List<Element>();
    private List<XElement> lexed_xml_response = new List<XElement>();
    private string target_url ;
    private string hangup_url ;
    public string CallSid;
    public string AccountSid;
    public string ApiVersion;
    public string CurrentElement ;
    public string RecordingPath;
    private string default_answer_url;
    public SortedList session_params = new SortedList();
    private BlockingCollection<Event> _actionEventQueue;
    private string _hangup_cause;
    private string channel_uuid;
    public FSOutbound(
        Socket socket,string address,string default_answer_url,string default_hangup_url,string extra_fs_vars,string default_http_method="POST", int request_id=0
                 ):base(socket)
        {
        //# the request id
        int _request_id = request_id;
        
        // set all settings empty
        this.xml_response = "";
        //this.parsed_element = new List<Element>();
        //this.lexed_xml_response = new List<XElement>();
        this.target_url = "";
        this.hangup_url = "";
        this._hangup_cause = "";
        // flag to track current element
        this.CurrentElement = "";
        //# create queue for waiting actions
        this._actionEventQueue = new BlockingCollection<Event>(1);
        // set default answer url
        this.default_answer_url = default_answer_url;
        //# set default hangup_url
        this.default_hangup_url = default_hangup_url;
        //# set default http method POST or GET
        this.default_http_method = default_http_method;
       // # identify the extra FS variables to be passed along
        this.extra_fs_vars = extra_fs_vars;
        //# set answered flag
        this.answered = false;
        
        base.OnCHANNEL_EXECUTE_COMPLETE += new EventHandlers(OnChannelExecuteComplete);
        base.OnCUSTOM += new EventHandlers(OnCustomEvent);
        base.OnCHANNEL_UNBRIDGE += new EventHandlers(OnChannelUnabridgeEvent);
        base.OnCHANNEL_HANGUP_COMPLETE += new EventHandlers(OnChannelHangupCompleteEvent);
        ElementTypeLoader.ScanForElements(GetType().Assembly, (name, type) => _elementTypes.Add(name, type));
        try
        {
            Run();
        }
        catch (Exception ex)
        {
            throw ex;
        }
        finally
        {
            Disconnect();
        }
        }

    public Event ActionReturnedEvent()
    {
     //   Wait until an action is over and return action event.
        return this._actionEventQueue.Take();
    }
    
        //In order to "block" the execution of our service until the
    //command is finished, we use a synchronized queue from gevent
    //and wait for such event to come. The on_channel_execute_complete
    //method will put that event in the queue, then we may continue working.
    //However, other events will still come, like for instance, DTMF.
    public void OnChannelExecuteComplete(Event evt)
    {
        string app = evt.GetHeader("Application");
        if  (WAIT_FOR_ACTIONS.Contains(app))
            // If transfer has begun, put empty event to break current action
            if (evt.GetHeader("variable_agbara_transfer_progress") == "true")
            {
                this._actionEventQueue.Add(new Event());
            }
            else
            {
               this._actionEventQueue.Add(evt);
            }
    }
    public void OnChannelHangupCompleteEvent(Event evt)
    {
     
     //   Capture Channel Hangup Complete
        this._hangup_cause = evt.GetHeader("Hangup-Cause");
        this.session_params["HangupCause"] = this._hangup_cause;
        this.session_params["CallStatus"] = CallStatus.completed;
        // Prevent command to be stuck while waiting response
        this._actionEventQueue.Add(new Event());
    }
    public void OnChannelUnabridgeEvent(Event evt)
    {
       // special case to get bleg uuid for Dial
        if (CurrentElement == "Dial")
        {
            this._actionEventQueue.Add(evt);
         }
    }
    public void OnCustomEvent(Event evt)
    {
        Console.WriteLine("Subclass {0}",evt.GetHeader("Event-Subclass"));
        Console.WriteLine("Action Performed {0}", evt.GetHeader("Action"));
        Console.WriteLine("Member Id {0}", evt.GetHeader("Member-ID"));
        // case conference event
        if(CurrentElement == "Conference")
        {
            //special case to get Member-ID for Conference
            if (evt.GetHeader("Event-Subclass") == "conference::maintenance" && evt.GetHeader("Action") == "add-member" && evt.GetHeader("Unique-ID") == channel_uuid)
            {
                Console.WriteLine("Subclass {0}", evt.GetHeader("Event-Subclass"));
                Console.WriteLine("Action Performed {0}", evt.GetHeader("Action"));
                Console.WriteLine("Member Id {0}", evt.GetHeader("Member-ID"));
               // this._actionEventQueue.Add(evt);
            }
            // special case for hangupOnStar in Conference
            else if(evt.GetHeader("Event-Subclass") == "conference::maintenance" && evt.GetHeader("Action") == "kick" && evt.GetHeader("Unique-ID") == channel_uuid)
            {
              string  room = evt.GetHeader("Conference-Name");
               string member_id = evt.GetHeader("Member-ID");
                if (!string.IsNullOrEmpty(room)&& !string.IsNullOrEmpty(member_id))
                {
                    BgAPICommand(string.Format("conference {0} kick {1}",room, member_id));
                }
            }
            // special case to send callback for Conference
            else if(evt.GetHeader("Event-Subclass") == "conference::maintenance" && evt.GetHeader("Action") == "digits-match" && evt.GetHeader("Unique-ID") == channel_uuid)
            {
               
                string digits_action = evt.GetHeader("Callback-Url");
                string digits_method = evt.GetHeader("Callback-Method");
                if (!string.IsNullOrEmpty(digits_action)&& !string.IsNullOrEmpty(digits_method))
                {
                    SortedList param = new SortedList();
                    param["ConferenceMemberID"] = evt.GetHeader("Member-ID");
                    param["ConferenceUUID"] = evt.GetHeader("Conference-Unique-ID");
                    param["ConferenceName"] = evt.GetHeader("Conference-Name");
                    param["ConferenceDigitsMatch"] = evt.GetHeader("Digits-Match");
                    param["ConferenceAction"] = "digits";
                    Task.Factory.StartNew(() => this.SendToUrl(digits_action, param, digits_method));
                }
            }
        }
        // case dial event
        else if(CurrentElement == "Dial")
            {
            if (evt.GetHeader("Event-Subclass") == "agbara::dial" && evt.GetHeader("Action") == "digits-match" && evt.GetHeader("Unique-ID") == channel_uuid)
            {
                string digits_action = evt.GetHeader("Callback-Url");
                string digits_method = evt.GetHeader("Callback-Method");
                if (!string.IsNullOrEmpty(digits_action)&& !string.IsNullOrEmpty(digits_method))
                {
                    SortedList param = new SortedList();
                    param["DialDigitsMatch"] = evt.GetHeader("Digits-Match");
                    param["DialAction"] = "digits";
                    param["DialALegUUID"] = evt.GetHeader("Unique-ID");
                    param["DialBLegUUID"] = evt.GetHeader("variable_bridge_uuid");
                    Task.Factory.StartNew(() => this.SendToUrl(digits_action, param, digits_method));
                }
            }
        }
    }
    public bool HasHangup()
    {
        if (!string.IsNullOrEmpty(_hangup_cause))
        {
            return true;
        }
        return false;
    }
    public bool HasAnswered()
    {
        return answered;
    }
    public string GetHangupCause()
    {
        return _hangup_cause;
    }
    public  void Disconnect()
    {
        //Prevent command to be stuck while waiting response
        try
        {
            this._actionEventQueue.Add(new Event());
        }
        catch(Exception ex)
        {
        }
        
        base.disconnect();
    }
    public override void  Run()
    {
        try
        {
            _run();
        }
        catch(Exception ex)
        {
        }
    }
    private void _run()
    {
        connect();
        myevents();
        Resume();
        //Linger to get all remaining events before closing
        linger();
        myevents();
        if (_iseventjson)
        {
         EventJson("CUSTOM conference::maintenance agbara::dial");
        }
        else
        {
            EventPlain("CUSTOM conference::maintenance agbara::dial");
        }

        //get channel unique Id
        this.channel_uuid = get_channel_unique_id();
        AccountSid = get_channel().GetHeader("variable_agbara_accountsid");
        CallSid = get_channel().GetHeader("variable_agbara_callsid");
        ApiVersion = get_channel().GetHeader("variable_agbara_apiversion");
        //Set agbara app flag
        set("agbara_app=true");
        //Don"t hangup after bridge
        set("hangup_after_bridge=false");
        CommandResponse channel = get_channel();
        string call_uuid = get_channel_unique_id();
        string called_no = channel.GetHeader("Caller-Destination-Number");
        string from_no = channel.GetHeader("Caller-Caller-ID-Number");
        
        //Set To to Session Params
        session_params.Add("To", called_no.TrimStart(new Char['+']));
        //Set From to Session Params
        session_params.Add("From", from_no.TrimStart(new Char['+']));
        
        string aleg_uuid = "";
        string aleg_request_uuid = "";
        string sched_hangup_id = channel.GetHeader("variable_agbara_sched_hangup_id");
        string forwarded_from;
        string variable_sip_h_Diversion = channel.GetHeader("variable_sip_h_Diversion");
        if (string.IsNullOrEmpty(variable_sip_h_Diversion))
        {
            forwarded_from = "";
        }
        else
        {
            int startindex = variable_sip_h_Diversion.IndexOf(':');
            int length = variable_sip_h_Diversion.IndexOf('@') - startindex;
            forwarded_from = variable_sip_h_Diversion.Substring(startindex, length);
        }
        var direction = channel.GetHeader("Call-Direction");
        if (direction == "outbound")
        {
            //# Look for variables in channel headers
            aleg_uuid = channel.GetHeader("Caller-Unique-ID");
            aleg_request_uuid = channel.GetHeader("variable_agbara_request_uuid");
            // Look for target url in order below :
            // get agbara_transfer_url from channel var
            //  get agbara_answer_url from channel var
            string xfer_url = channel.GetHeader("variable_agbara_transfer_url");
            string answer_url = channel.GetHeader("variable_agbara_answer_url");
            if (CurrentElement == "Dial")
            {
                session_params.Add("Direction", CallDirection.outbounddial);
            }
            else
            {
                session_params.Add("Direction",CallDirection.outboundapi);
            }
            
            if (!string.IsNullOrEmpty(xfer_url))
            {
                this.target_url = xfer_url;

            }
            else if(!string.IsNullOrEmpty(answer_url))
            {
                this.target_url = answer_url;
            }
            else
            {
                return;
            }
            //Look for a sched_hangup_id
            
            //Don"t post hangup in outbound direction because it is handled by inboundsocket
            default_hangup_url = "";
            hangup_url = "";
            //Set CallStatus to Session Params
            session_params.Add("CallStatus",CallStatus.inprogress);
        }
        else
        {
            session_params.Add("Direction", CallDirection.inbound);
            //Look for target url in order below :
            //get agbara_transfer_url from channel var
            //get agbara_answer_url from channel var
            //get default answer_url from config
            string xfer_url = this.GetVar("agbara_transfer_url");
            string answer_url = this.GetVar("agbara_answer_url");
            if (!string.IsNullOrEmpty(xfer_url))
            {
                this.target_url = xfer_url;
            }
            else if(!string.IsNullOrEmpty(answer_url))
            {
                this.target_url = answer_url;
            }
            else if(!string.IsNullOrEmpty(default_answer_url))
            {
                this.target_url = default_answer_url;
            }
            else
            {
                return;
            }
        }
            
            //Look for a sched_hangup_id
            sched_hangup_id = GetVar("agbara_sched_hangup_id",get_channel_unique_id());
            //Set CallStatus to Session Params
            //this.session_params.Add("CallStatus", CallStatus.ringing);

        if (string.IsNullOrEmpty(sched_hangup_id))
                sched_hangup_id = "";

        //Add more Session Params 
        session_params.Add("AccountSid", channel.GetHeader("variable_agbara_accountsid"));
        session_params.Add("CallSid",channel.GetHeader("variable_agbara_callsid"));
        if (!string.IsNullOrEmpty(forwarded_from))
            session_params.Add("ForwardedFrom", forwarded_from.TrimStart(new Char[]{'+'}));

        // Remove sched_hangup_id from channel vars
        if (!string.IsNullOrEmpty(sched_hangup_id))
            unset("agbara_sched_hangup_id");

        // Run application
        try
        {
            Console.WriteLine("Processing call");
           ProcessCall();
        }
        catch(Exception ex)
        {
        //except RESTHangup:
        //    this.log.warn("Channel has hung up, breaking Processing Call")
        //except Exception, e:
        //    this.log.error("Processing Call Failure !")
        //    # If error occurs during xml parsing
        //    # log exception and break
        //    this.log.error(str(e))
        //    [ this.log.error(line) for line in \
        //                traceback.format_exc().splitlines() ]
        //this.log.info("Processing Call Ended")
        }

    }
    private void ProcessCall()
    {
        //Method to proceed on the call. This will fetch the XML, validate the response. Parse the XML and Execute it
       for (int x=0; x<999;x++)
        {
            try
            {
                // update call status if needed
                if (HasHangup())
                    session_params["CallStatus"] = "completed";
                // case answer url, add extra vars to http request :
                
                //fetch remote response
                FetchResponse("GET");
                // check hangup
                if (HasHangup())
                //    raise RESTHangup();
                if (!string.IsNullOrEmpty(xml_response))
                {
                    return;
                }
                // parse and execute restxml
                LexXml();
                ParseXml();
                ExecuteResponse();
                return;
            }
            catch(RedirectException redirect)
            {
                //if has_hangup()
                //    raise RESTHangup()
                //Set target URL to Redirect URL
                // Set method to Redirect method
                // Set additional params to Redirect params
                this.target_url = redirect.GetUrl();
               string  fetch_method = redirect.GetMethod();
               SortedList param = redirect.GetParam();
                if (string.IsNullOrEmpty(fetch_method))
                    fetch_method = "POST";
                // Reset all the previous response and element
                this.xml_response = "";
                this.parsed_element =new List<Element>();
                this.lexed_xml_response = new List<XElement>();
                // If transfer is in progress, break redirect
                string xfer_progress = GetVar("agbara_transfer_progress",get_channel_unique_id()) ;
                if (xfer_progress =="true")
                {
                    //Transfer in progress, breaking redirect
                    return;
                }
                System.Threading.Thread.Sleep(1000);
                continue;
            }
            }
    }
    public void FetchResponse(string method = "")
    {
        //This method will retrieve the xml from the answer_url
        //The url result expected is an XML content which will be stored in xml_response
        //Hashtable param = new Hashtable();
        xml_response = SendToUrl(target_url, session_params, method);
    }
    public string SendToUrl(string url, SortedList param, string method = "")
    {
     //   This method will do an http POST or GET request to the Url
        if(method == "")
        {
            method = default_http_method;
        }

        if (string.IsNullOrEmpty(url))
        {
            return "";
        }

        HTTPRequest http_obj = new  HTTPRequest(this.AccountSid, this.AccountSid);
        try
        {
            
            string data = http_obj.FetchResponse(url, method,session_params );
            return data;
        }
        catch(Exception ex)
        {
            return "";
        }
      }
    private void LexXml()
    {
        XElement doc = null;
     //   Validate the XML document and make sure we recognize all Element
        try
        {
            //convert the string into an Element instance
           doc = XElement.Parse(xml_response);
        }
        catch
        {
            // stop processing
        }

        //Make sure the document has a <AgbaraResponse> root
        if (doc.Name != "Response")
        {
            return;
        }

        // Make sure we recognize all the Element in the xml
        foreach(XElement element in doc.Nodes())
        {
          if(VALID_ELEMENTS.ToList().Contains(element.Name.ToString()))
          {
              lexed_xml_response.Add(element);
          }
            
        }
        }
    private void ParseXml()
    {
        //This method will parse the XML and add the Elements into parsed_element
        //Check all Elements tag name
        foreach(XElement element in lexed_xml_response)
        {
            Type type;
            Element _element = null;
			if (!_elementTypes.TryGetValue(element.Name.ToString(), out type))
			{
				return;
			}

			try
			{
				_element = (Element) Activator.CreateInstance(type);
                
				_element.ParseElement(element,this.target_url);
                parsed_element.Add(_element);
				
			}
			catch (Exception err)
			{
			}
            //Validate, Parse & Store the nested children
            //inside the main element element
            this.ValidateElement(element, _element);
        }
    }
    private void ValidateElement(XElement element,Element element_instance)
    {
        IEnumerable<XElement> children = element.Elements();
        if (children != null && element_instance.Nestables.Count > 0)
        {

            foreach (XElement child in children)
            {
                ParseChildren(child, element_instance);
            }

        }
    }
    private void ParseChildren(XElement child_element,Element parent_instance)
    {
        Type type;
        Element _element = null;
        if (!_elementTypes.TryGetValue(child_element.Name.ToString(), out type))
        {
            return;
        }

        try
        {
            _element = (Element)Activator.CreateInstance(type);

            _element.ParseElement(child_element, this.target_url);
            parent_instance.Children.Add(_element);

        }
        catch (Exception ex)
        {
        }
    }
    private void ExecuteResponse()
    {
        foreach(Element element in parsed_element)
        {
            if (element.GetType().GetMethods().Where(m=>m.Name.ToLower() =="Prepare").Count()>0)
            {
                //Task.Factory.StartNew(()=>element_instance.Prepare()
            }
            
            //Check if it"s an inbound call
            if (session_params["Direction"].ToString() == "inbound")
            {
                //Answer the call if element need it
                if (!answered && NO_ANSWER_ELEMENTS.ToList().Contains(element.GetType().Name))
                {
                    answer();
                    answered = true;
                    //After answer, update callstatus to "in-progress"
                    session_params["CallStatus"] = "in-progress";
                }
            }
            //execute Element
            element.Run(this);
        }
        // If transfer is in progress, don"t hangup call
        if (!HasHangup())
        {
            string xfer_progress = GetVar("agbara_transfer_progress");
            if (string.IsNullOrEmpty(xfer_progress))
            {
                session_params["CallStatus"] = "completed";
                session_params["HangupCause"] = "NORMAL_CLEARING";
                hangup();
            }
        }
    }
    }

}
