using System;
using System.Collections.Generic;
using System.Collections.Specialized;
using System.Collections.Concurrent;
using System.Threading;
using System.Threading.Tasks;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{
    public abstract class EventSocket : Commands
    {
    #region Field
    internal bool _closing_state;
    internal bool _is_eventjson;
    internal string _filter;
    internal bool connected;
    internal bool _running;
    internal Transport transport;
    string EOL = "\n";
    int MAXLINES_PER_EVENT = 1000;
    internal readonly BlockingCollection<Event> _response_queue ;
    private object objlock = new object();
    internal readonly Dictionary<string, MessageHandler> _response_callbacks = new Dictionary<string, MessageHandler>();
    public Dictionary<string, Delegate> event_handlers;
    #endregion Field

    #region Constructor
    public EventSocket(string filter = "ALL", bool eventjson = true)
        {
            _response_queue = new BlockingCollection<Event>(1);
            _is_eventjson = eventjson;
            _closing_state = false;
            _filter = filter;
            connected = false;
            _running = true;
            if (_is_eventjson)
                _response_callbacks.Add("text/event-json", _event_json);
            else
            _response_callbacks.Add("text/event-plain", _event_plain);
            _response_callbacks.Add("command/reply", _command_reply);
            _response_callbacks.Add("api/response", _api_response);
            _response_callbacks.Add("auth/request", _auth_request);
            _response_callbacks.Add("text/disconnect-notice", _disconnect_notice);
            event_handlers = new Dictionary<string, Delegate>();
           LoadEventHandlers();
        }
    #endregion


    #region Private Methods
    private Event _auth_request(Event ev)
        {
        /*
        Receives auth/request callback.
         */
        _response_queue.Add(ev);
        return ev;
        }
    private Event _api_response(Event ev)
    {
     /*
        Receives api/response callback.
        Gets raw data for this event.*/
        string raw = read_raw(ev);
       // If raw was found, this is our Event body.
        if (!string.IsNullOrEmpty(raw))
            ev.SetBody(raw);
        // Pushes Event to response events queue and returns Event.
        _response_queue.Add(ev);
        return ev;
    }
    private Event _command_reply(Event ev)
    {
     /*
        Receives command/reply callback.
        Pushes Event to response events queue and returns Event.*/
        _response_queue.Add(ev);
        return ev;
    }
    private Event _event_plain(Event ev)
    {
     /*
        Receives text/event-plain callback.
        Gets raw data for this event*/
        string raw = read_raw(ev);
        // If raw was found drops current event and replaces with Event created from raw
        if (!string.IsNullOrEmpty(raw))
            ev = new Event(raw);
            // Gets raw response from Event Content-Length header and raw buffer
            string raw_response = read_raw_response(ev, raw);
            //If rawresponse was found, this is our Event body
            if(!string.IsNullOrEmpty(raw_response))
                ev.SetBody(raw_response);
       // Returns Event
        return ev;
    }
    private Event _event_json(Event ev)
    {
     /*
        Receives text/event-json callback.
        Gets json data for this event */
        JsonEvent jsonev;
        Event retev = new Event();
        string json_data = read_raw(ev);
        //If raw was found drops current event and replaces with JsonEvent created from json_data
        if (!string.IsNullOrEmpty(json_data))
        {
            jsonev = new JsonEvent(json_data);
            retev = (Event)jsonev;
        }
        return retev;
    }
    private Event _disconnect_notice(Event ev)
    {
        /*
           Receives text/disconnect-notice callback.
           */
        _closing_state = true;
        return ev;
    }
    private void _send(string cmd)
    {
        cmd += EOL;
        transport.write(cmd + EOL);
    }
    private void _sendmsg(string name, string arg = "", string uuid = "", bool islock = false, int loops = 1, bool async = false)
    {   
        string msg = string.Format("sendmsg {0}\ncall-command: execute\n", uuid);
        msg += string.Format("execute-app-name: {0}\n", name);
        if (islock)
            msg += "event-lock: true\n";
        if (loops > 1)
            msg += string.Format("loops: {0}\n", loops);
        if (async)
            msg += "async: true\n";
        if (!string.IsNullOrEmpty(arg))
        {
            int arglen = arg.Length;
            msg += string.Format("content-type: text/plain\ncontent-length: {0}\n\n{1}\n", arglen, arg);
        }
        transport.write(msg + EOL);
    }
    private void LoadEventHandlers()
    {
        event_handlers.Add("CUSTOM", null);
        event_handlers.Add("CLONE", null);
        event_handlers.Add("CHANNEL_CREATE", null);
        event_handlers.Add("CHANNEL_DESTROY", null);
        event_handlers.Add("CHANNEL_STATE", null);
        event_handlers.Add("CHANNEL_CALLSTATE", null);
        event_handlers.Add("CHANNEL_ANSWER", null);
        event_handlers.Add("CHANNEL_HANGUP", null);
        event_handlers.Add("CHANNEL_HANGUP_COMPLETE", null);
        event_handlers.Add("CHANNEL_EXECUTE", null);
        event_handlers.Add("CHANNEL_EXECUTE_COMPLETE", null);
        event_handlers.Add("CHANNEL_HOLD", null);
        event_handlers.Add("CHANNEL_UNHOLD", null);
        event_handlers.Add("CHANNEL_BRIDGE", null);
        event_handlers.Add("CHANNEL_UNBRIDGE", null);
        event_handlers.Add("CHANNEL_PROGRESS", null);
        event_handlers.Add("CHANNEL_PROGRESS_MEDIA", null);
        event_handlers.Add("CHANNEL_OUTGOING", null);
        event_handlers.Add("CHANNEL_PARK", null);
        event_handlers.Add("CHANNEL_UNPARK", null);
        event_handlers.Add("CHANNEL_APPLICATION", null);
        event_handlers.Add("CHANNEL_ORIGINATE", null);
        event_handlers.Add("CHANNEL_UUID", null);
        event_handlers.Add("API", null);
        event_handlers.Add("LOG", null);
        event_handlers.Add("INBOUND_CHAN", null);
        event_handlers.Add("OUTBOUND_CHAN", null);
        event_handlers.Add("STARTUP", null);
        event_handlers.Add("SHUTDOWN", null);
        event_handlers.Add("PUBLISH", null);
        event_handlers.Add("UNPUBLISH", null);
        event_handlers.Add("TALK", null);
        event_handlers.Add("NOTALK", null);
        event_handlers.Add("SESSION_CRASH", null);
        event_handlers.Add("MODULE_LOAD", null);
        event_handlers.Add("MODULE_UNLOAD", null);
        event_handlers.Add("DTMF", null);
        event_handlers.Add("MESSAGE", null);
        event_handlers.Add("PRESENCE_IN", null);
        event_handlers.Add("NOTIFY_IN", null);
        event_handlers.Add("PRESENCE_OUT", null);
        event_handlers.Add("PRESENCE_PROBE", null);
        event_handlers.Add("MESSAGE_WAITING", null);
        event_handlers.Add("MESSAGE_QUERY", null);
        event_handlers.Add("ROSTER", null);
        event_handlers.Add("CODEC", null);
        event_handlers.Add("BACKGROUND_JOB", null);
        event_handlers.Add("DETECTED_SPEECH", null);
        event_handlers.Add("DETECTED_TONE", null);
        event_handlers.Add("PRIVATE_COMMAND", null);
        event_handlers.Add("HEARTBEAT", null);
        event_handlers.Add("TRAP", null);
        event_handlers.Add("ADD_SCHEDULE", null);
        event_handlers.Add("DEL_SCHEDULE", null);
        event_handlers.Add("EXE_SCHEDULE", null);
        event_handlers.Add("RE_SCHEDULE", null);
        event_handlers.Add("RELOADXML", null);
        event_handlers.Add("NOTIFY", null);
        event_handlers.Add("SEND_MESSAGE", null);
        event_handlers.Add("RECV_MESSAGE", null);
        event_handlers.Add("REQUEST_PARAMS", null);
        event_handlers.Add("CHANNEL_DATA", null);
        event_handlers.Add("GENERAL", null);
        event_handlers.Add("COMMAND", null);
        event_handlers.Add("SESSION_HEARTBEAT", null);
        event_handlers.Add("CLIENT_DISCONNECTED", null);
        event_handlers.Add("SERVER_DISCONNECTED", null);
        event_handlers.Add("SEND_INFO", null);
        event_handlers.Add("RECV_INFO", null);
        event_handlers.Add("CALL_SECURE", null);
        event_handlers.Add("NAT", null);
        event_handlers.Add("RECORD_START", null);
        event_handlers.Add("RECORD_STOP", null);
        event_handlers.Add("PLAYBACK_START", null);
        event_handlers.Add("PLAYBACK_STOP", null);
        event_handlers.Add("CALL_UPDATE", null);
        event_handlers.Add("FAILURE", null);
        event_handlers.Add("SOCKET_DATA", null);
        event_handlers.Add("MEDIA_BUG_START", null);
        event_handlers.Add("MEDIA_BUG_STOP", null);
        event_handlers.Add("ALL", null);
    }
    #endregion

    #region Public Methods
    public bool is_connected()
    {
        /*
        Checks if connected and authenticated to eventsocket.
        Returns True or False.
        */
        return connected;
    }
    public void start_event_handler()
    {
     /*
        Starts Event handler in background.
       */
        Task.Factory.StartNew(()=> handle_events());
        
    }
    public void handle_events()
    {
        /*Gets and Dispatches events in an endless loop using gevent spawn.*/
        while (true)
        {
            try
            {
                //   # Gets event and dispatches to handler.
                get_event();
                Thread.Sleep(10);
            }
            catch (Exception ex)
            {
                throw ex;
            }
            finally
            {
                connected = false;
            }
        }
            
    }
    public  void get_event()
    {
        //Gets complete Event, and processes response callback.
        Event ev = read_event();
       //Gets callback response for this event
        MessageHandler handler;
        if (!_response_callbacks.TryGetValue(ev.GetContentType(), out handler))
        {
            return;
        }
        try
        {
            ev = handler(ev);
        }
        catch (Exception ex)
        {
            throw ex;
        }
        
        // # Only dispatches event if Event-Name header found.
        if (!string.IsNullOrEmpty(ev.GetHeader("Event-Name")))
        {
            Task.Factory.StartNew(() => dispatch_event(ev));
        }
    }
    public Event read_event()
    {
     /*
        Reads one Event from socket until EOL.
        Returns Event instance.
        Raises LimitExceededError if MAXLINES_PER_EVENT is reached.
        */
        string buff = "";
        for (int i = 0; i < MAXLINES_PER_EVENT; i++)
        {
            string line = transport.read_line();
            if (line == "")
                // # When matches EOL, creates Event and returns it.
                return new Event(buff);
            else
            {
                //# Else appends line to current buffer.
                line += "\n";
                buff += line;
            }
        }
        throw new LimitExceededError(string.Format("max lines per event {0} reached",MAXLINES_PER_EVENT));
    }
    public string read_raw(Event ev)
    {
     /*
        Reads raw data based on Event Content-Length.
        Returns raw string or None if not found.
        */
        int length = ev.GetContentLength();
        //# Reads length bytes if length > 0
        if (length >0)
            return transport.read(length);
        return string.Empty;
    }
    public string read_raw_response(Event ev,string raw)
    {
     /*
        Extracts raw response from raw buffer and length based on Event Content-Length.
        Returns raw string or None if not found.
        */
        int length = ev.GetContentLength();
        if (length > 0 && !string.IsNullOrEmpty(raw))
            return raw.Substring(raw.Length - length, length);
        return raw;
    }
    private void dispatch_event(Event ev)
    {
        EventHandlers handler;
        if (null != (handler = (EventHandlers)event_handlers[ev.EventName]))
        {
            handler(ev);
        }

    }
    public virtual void connect()
    {
     /*
        Connects to eventsocket.
       */
        _closing_state = false;
    }
    public void disconnect()
    {
        /*
           Disconnect and release socket and finally kill event handler.
           '''*/
        connected = false;
        transport.close();
    }
    public override Event ProtocolSend(string commands, string args = "")
    {
        Event ev = new Event();
        if (_closing_state)
            return ev;

        lock (objlock)
        {
            if (_response_queue.Count != 0)
            {
                _response_queue.Take();
            }
            _send(string.Format("{0} {1}", commands, args));
            ev = _response_queue.Take();
            
        }
        
        // Casts Event to appropriate event type :
        // Casts to ApiResponse, if event is api
        if (commands == "api")
        {
            ev = APIResponse.Cast(ev);
        }
        //Casts to BgapiResponse, if event is bgapi
        else if (commands == "bgapi")
        {
             ev = BgApiResponse.Cast(ev);
        }
        //Casts to CommandResponse by default
        else
        {
            ev = CommandResponse.Cast(ev);
        }
        return ev;
    }
    public override Event ProtocolSendMsg(string name, string args, string uuid = "", bool Lock = false, int loop = 1, bool async=false)
    {
        Event ev = new Event();
        if (_closing_state)
            return ev;
        lock (objlock)
        {
            if (_response_queue.Count != 0)
            {
                _response_queue.Take();
            }
            _sendmsg(name, args, uuid, Lock, loop, async);
             ev = _response_queue.Take();
        }
       // _resetEvent.Reset();
              ev = CommandResponse.Cast(ev);
            // Always casts Event to CommandResponse
            return ev;
        
    }
    #endregion

    internal delegate Event MessageHandler(Event message);
    public delegate void EventHandlers(Event message);
    #region Event
    public event EventHandlers OnCUSTOM
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CUSTOM"] = (EventHandlers)event_handlers["CUSTOM"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CUSTOM"] = (EventHandlers)event_handlers["CUSTOM"] - value;
            }
        }
    }
	public event EventHandlers OnCLONE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CLONE"] = (EventHandlers)event_handlers["CLONE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CLONE"] = (EventHandlers)event_handlers["CLONE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_CREATE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_CREATE"] = (EventHandlers)event_handlers["CHANNEL_CREATE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_CREATE"] = (EventHandlers)event_handlers["CHANNEL_CREATE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_DESTROY
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_DESTROY"] = (EventHandlers)event_handlers["CHANNEL_DESTROY"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_DESTROY"] = (EventHandlers)event_handlers["CHANNEL_DESTROY"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_STATE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_STATE"] = (EventHandlers)event_handlers["CHANNEL_STATE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_STATE"] = (EventHandlers)event_handlers["CHANNEL_STATE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_CALLSTATE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_CALLSTATE"] = (EventHandlers)event_handlers["CHANNEL_CALLSTATE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_CALLSTATE"] = (EventHandlers)event_handlers["CHANNEL_CALLSTATE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_ANSWER
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_ANSWER"] = (EventHandlers)event_handlers["CHANNEL_ANSWER"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_ANSWER"] = (EventHandlers)event_handlers["CHANNEL_ANSWER"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_HANGUP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HANGUP"] = (EventHandlers)event_handlers["CHANNEL_HANGUP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HANGUP"] = (EventHandlers)event_handlers["CHANNEL_HANGUP"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_HANGUP_COMPLETE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HANGUP_COMPLETE"] = (EventHandlers)event_handlers["CHANNEL_HANGUP_COMPLETE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HANGUP_COMPLETE"] = (EventHandlers)event_handlers["CHANNEL_HANGUP_COMPLETE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_EXECUTE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_EXECUTE"] = (EventHandlers)event_handlers["CHANNEL_EXECUTE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_EXECUTE"] = (EventHandlers)event_handlers["CHANNEL_EXECUTE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_EXECUTE_COMPLETE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_EXECUTE_COMPLETE"] = (EventHandlers)event_handlers["CHANNEL_EXECUTE_COMPLETE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_EXECUTE_COMPLETE"] = (EventHandlers)event_handlers["CHANNEL_EXECUTE_COMPLETE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_HOLD
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HOLD"] = (EventHandlers)event_handlers["CHANNEL_HOLD"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_HOLD"] = (EventHandlers)event_handlers["CHANNEL_HOLD"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_UNHOLD
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNHOLD"] = (EventHandlers)event_handlers["CHANNEL_UNHOLD"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNHOLD"] = (EventHandlers)event_handlers["CHANNEL_UNHOLD"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_BRIDGE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_BRIDGE"] = (EventHandlers)event_handlers["CHANNEL_BRIDGE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_BRIDGE"] = (EventHandlers)event_handlers["CHANNEL_BRIDGE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_UNBRIDGE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNBRIDGE"] = (EventHandlers)event_handlers["CHANNEL_UNBRIDGE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNBRIDGE"] = (EventHandlers)event_handlers["CHANNEL_UNBRIDGE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_PROGRESS
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PROGRESS"] = (EventHandlers)event_handlers["CHANNEL_PROGRESS"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PROGRESS"] = (EventHandlers)event_handlers["CHANNEL_PROGRESS"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_PROGRESS_MEDIA
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PROGRESS_MEDIA"] = (EventHandlers)event_handlers["CHANNEL_PROGRESS_MEDIA"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PROGRESS_MEDIA"] = (EventHandlers)event_handlers["CHANNEL_PROGRESS_MEDIA"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_OUTGOING
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_OUTGOING"] = (EventHandlers)event_handlers["CHANNEL_OUTGOING"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_OUTGOING"] = (EventHandlers)event_handlers["CHANNEL_OUTGOING"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_PARK
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PARK"] = (EventHandlers)event_handlers["CHANNEL_PARK"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_PARK"] = (EventHandlers)event_handlers["CHANNEL_PARK"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_UNPARK
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNPARK"] = (EventHandlers)event_handlers["CHANNEL_UNPARK"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UNPARK"] = (EventHandlers)event_handlers["CHANNEL_UNPARK"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_APPLICATION
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_APPLICATION"] = (EventHandlers)event_handlers["CHANNEL_APPLICATION"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_APPLICATION"] = (EventHandlers)event_handlers["CHANNEL_APPLICATION"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_ORIGINATE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_ORIGINATE"] = (EventHandlers)event_handlers["CHANNEL_ORIGINATE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_ORIGINATE"] = (EventHandlers)event_handlers["CHANNEL_ORIGINATE"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_UUID
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UUID"] = (EventHandlers)event_handlers["CHANNEL_UUID"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_UUID"] = (EventHandlers)event_handlers["CHANNEL_UUID"] - value;
            }
        }
    }
	public event EventHandlers OnAPI
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["API"] = (EventHandlers)event_handlers["API"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["API"] = (EventHandlers)event_handlers["API"] - value;
            }
        }
    }
	public event EventHandlers OnLOG
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["LOG"] = (EventHandlers)event_handlers["LOG"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["LOG"] = (EventHandlers)event_handlers["LOG"] - value;
            }
        }
    }
	public event EventHandlers OnINBOUND_CHAN
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["INBOUND_CHAN"] = (EventHandlers)event_handlers["INBOUND_CHAN"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["INBOUND_CHAN"] = (EventHandlers)event_handlers["INBOUND_CHAN"] - value;
            }
        }
    }
	public event EventHandlers OnOUTBOUND_CHAN
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["OnOUTBOUND_CHAN"] = (EventHandlers)event_handlers["OnOUTBOUND_CHAN"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["OnOUTBOUND_CHAN"] = (EventHandlers)event_handlers["OnOUTBOUND_CHAN"] - value;
            }
        }
    }
	public event EventHandlers OnSTARTUP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["STARTUP"] = (EventHandlers)event_handlers["STARTUP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["STARTUP"] = (EventHandlers)event_handlers["STARTUP"] - value;
            }
        }
    }
	public event EventHandlers OnSHUTDOWN
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SHUTDOWN"] = (EventHandlers)event_handlers["SHUTDOWN"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SHUTDOWN"] = (EventHandlers)event_handlers["SHUTDOWN"] - value;
            }
        }
    }
	public event EventHandlers OnPUBLISH
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PUBLISH"] = (EventHandlers)event_handlers["PUBLISH"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PUBLISH"] = (EventHandlers)event_handlers["PUBLISH"] - value;
            }
        }
    }
	public event EventHandlers OnUNPUBLISH
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["UNPUBLISH"] = (EventHandlers)event_handlers["UNPUBLISH"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["UNPUBLISH"] = (EventHandlers)event_handlers["UNPUBLISH"] - value;
            }
        }
    }
	public event EventHandlers OnTALK
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["TALK"] = (EventHandlers)event_handlers["TALK"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["TALK"] = (EventHandlers)event_handlers["TALK"] - value;
            }
        }
    }
	public event EventHandlers OnNOTALK
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["NOTALK"] = (EventHandlers)event_handlers["NOTALK"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["NOTALK"] = (EventHandlers)event_handlers["NOTALK"] - value;
            }
        }
    }
	public event EventHandlers OnSESSION_CRASH
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SESSION_CRASH"] = (EventHandlers)event_handlers["SESSION_CRASH"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SESSION_CRASH"] = (EventHandlers)event_handlers["SESSION_CRASH"] - value;
            }
        }
    }
	public event EventHandlers OnMODULE_LOAD
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MODULE_LOAD"] = (EventHandlers)event_handlers["MODULE_LOAD"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MODULE_LOAD"] = (EventHandlers)event_handlers["MODULE_LOAD"] - value;
            }
        }
    }
	public event EventHandlers OnMODULE_UNLOAD
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MODULE_UNLOAD"] = (EventHandlers)event_handlers["MODULE_UNLOAD"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MODULE_UNLOAD"] = (EventHandlers)event_handlers["MODULE_UNLOAD"] - value;
            }
        }
    }
	public event EventHandlers OnDTMF
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["DTMF"] = (EventHandlers)event_handlers["DTMF"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["DTMF"] = (EventHandlers)event_handlers["DTMF"] - value;
            }
        }
    }
	public event EventHandlers OnMESSAGE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE"] = (EventHandlers)event_handlers["MESSAGE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE"] = (EventHandlers)event_handlers["MESSAGE"] - value;
            }
        }
    }
	public event EventHandlers OnPRESENCE_IN
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_IN"] = (EventHandlers)event_handlers["PRESENCE_IN"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_IN"] = (EventHandlers)event_handlers["PRESENCE_IN"] - value;
            }
        }
    }
	public event EventHandlers OnNOTIFY_IN
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["NOTIFY_IN"] = (EventHandlers)event_handlers["NOTIFY_IN"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["NOTIFY_IN"] = (EventHandlers)event_handlers["NOTIFY_IN"] - value;
            }
        }
    }
	public event EventHandlers OnPRESENCE_OUT
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_OUT"] = (EventHandlers)event_handlers["PRESENCE_OUT"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_OUT"] = (EventHandlers)event_handlers["PRESENCE_OUT"] - value;
            }
        }
    }
	public event EventHandlers OnPRESENCE_PROBE
        {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_PROBE"] = (EventHandlers)event_handlers["PRESENCE_PROBE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PRESENCE_PROBE"] = (EventHandlers)event_handlers["PRESENCE_PROBE"] - value;
            }
        }
    }
	public event EventHandlers OnMESSAGE_WAITING
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE_WAITING"] = (EventHandlers)event_handlers["MESSAGE_WAITING"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE_WAITING"] = (EventHandlers)event_handlers["MESSAGE_WAITING"] - value;
            }
        }
    }
	public event EventHandlers OnMESSAGE_QUERY
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE_QUERY"] = (EventHandlers)event_handlers["MESSAGE_QUERY"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MESSAGE_QUERY"] = (EventHandlers)event_handlers["MESSAGE_QUERY"] - value;
            }
        }
    }
	public event EventHandlers OnROSTER
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["ROSTER"] = (EventHandlers)event_handlers["ROSTER"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["ROSTER"] = (EventHandlers)event_handlers["ROSTER"] - value;
            }
        }
    }
	public event EventHandlers OnCODEC
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CODEC"] = (EventHandlers)event_handlers["CODEC"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CODEC"] = (EventHandlers)event_handlers["CODEC"] - value;
            }
        }
    }
    public event EventHandlers OnBACKGROUND_JOB
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["BACKGROUND_JOB"] = (EventHandlers)event_handlers["BACKGROUND_JOB"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["BACKGROUND_JOB"] = (EventHandlers)event_handlers["BACKGROUND_JOB"] - value;
            }
        }
    }
	public event EventHandlers OnDETECTED_SPEECH
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["DETECTED_SPEECH"] = (EventHandlers)event_handlers["DETECTED_SPEECH"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["DETECTED_SPEECH"] = (EventHandlers)event_handlers["DETECTED_SPEECH"] - value;
            }
        }
    }
	public event EventHandlers OnDETECTED_TONE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["DETECTED_TONE"] = (EventHandlers)event_handlers["DETECTED_TONE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["DETECTED_TONE"] = (EventHandlers)event_handlers["DETECTED_TONE"] - value;
            }
        }
    }
	public event EventHandlers OnPRIVATE_COMMAND
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PRIVATE_COMMAND"] = (EventHandlers)event_handlers["PRIVATE_COMMAND"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PRIVATE_COMMAND"] = (EventHandlers)event_handlers["PRIVATE_COMMAND"] - value;
            }
        }
    }
	public event EventHandlers OnHEARTBEAT
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["HEARTBEAT"] = (EventHandlers)event_handlers["HEARTBEAT"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["HEARTBEAT"] = (EventHandlers)event_handlers["HEARTBEAT"] - value;
            }
        }
    }
	public event EventHandlers OnTRAP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["TRAP"] = (EventHandlers)event_handlers["TRAP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["TRAP"] = (EventHandlers)event_handlers["TRAP"] - value;
            }
        }
    }
	public event EventHandlers OnADD_SCHEDULE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["ADD_SCHEDULE"] = (EventHandlers)event_handlers["ADD_SCHEDULE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["ADD_SCHEDULE"] = (EventHandlers)event_handlers["ADD_SCHEDULE"] - value;
            }
        }
    }
	public event EventHandlers OnDEL_SCHEDULE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["DEL_SCHEDULE"] = (EventHandlers)event_handlers["DEL_SCHEDULE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["DEL_SCHEDULE"] = (EventHandlers)event_handlers["DEL_SCHEDULE"] - value;
            }
        }
    }
	public event EventHandlers OnEXE_SCHEDULE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["EXE_SCHEDULE"] = (EventHandlers)event_handlers["EXE_SCHEDULE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["EXE_SCHEDULE"] = (EventHandlers)event_handlers["EXE_SCHEDULE"] - value;
            }
        }
    }
	public event EventHandlers OnRE_SCHEDULE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RE_SCHEDULE"] = (EventHandlers)event_handlers["RE_SCHEDULE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RE_SCHEDULE"] = (EventHandlers)event_handlers["RE_SCHEDULE"] - value;
            }
        }
    }
	public event EventHandlers OnRELOADXML
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RELOADXML"] = (EventHandlers)event_handlers["RELOADXML"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RELOADXML"] = (EventHandlers)event_handlers["RELOADXML"] - value;
            }
        }
    }
	public event EventHandlers OnNOTIFY
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["NOTIFY"] = (EventHandlers)event_handlers["NOTIFY"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["NOTIFY"] = (EventHandlers)event_handlers["NOTIFY"] - value;
            }
        }
    }
	public event EventHandlers OnSEND_MESSAGE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SEND_MESSAGE"] = (EventHandlers)event_handlers["SEND_MESSAGE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SEND_MESSAGE"] = (EventHandlers)event_handlers["SEND_MESSAGE"] - value;
            }
        }
    }
	public event EventHandlers OnRECV_MESSAGE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RECV_MESSAGE"] = (EventHandlers)event_handlers["RECV_MESSAGE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RECV_MESSAGE"] = (EventHandlers)event_handlers["RECV_MESSAGE"] - value;
            }
        }
    }
	public event EventHandlers OnREQUEST_PARAMS
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["REQUEST_PARAMS"] = (EventHandlers)event_handlers["REQUEST_PARAMS"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["REQUEST_PARAMS"] = (EventHandlers)event_handlers["REQUEST_PARAMS"] - value;
            }
        }
    }
	public event EventHandlers OnCHANNEL_DATA
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_DATA"] = (EventHandlers)event_handlers["CHANNEL_DATA"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CHANNEL_DATA"] = (EventHandlers)event_handlers["CHANNEL_DATA"] - value;
            }
        }
    }
	public event EventHandlers OnGENERAL
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["GENERAL"] = (EventHandlers)event_handlers["GENERAL"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["GENERAL"] = (EventHandlers)event_handlers["GENERAL"] - value;
            }
        }
    }
	public event EventHandlers OnCOMMAND
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["COMMAND"] = (EventHandlers)event_handlers["COMMAND"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["COMMAND"] = (EventHandlers)event_handlers["COMMAND"] - value;
            }
        }
    }
	public event EventHandlers OnSESSION_HEARTBEAT
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SESSION_HEARTBEAT"] = (EventHandlers)event_handlers["SESSION_HEARTBEAT"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SESSION_HEARTBEAT"] = (EventHandlers)event_handlers["SESSION_HEARTBEAT"] - value;
            }
        }
    }
	public event EventHandlers OnCLIENT_DISCONNECTED
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CLIENT_DISCONNECTED"] = (EventHandlers)event_handlers["CLIENT_DISCONNECTED"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CLIENT_DISCONNECTED"] = (EventHandlers)event_handlers["CLIENT_DISCONNECTED"] - value;
            }
        }
    }
	public event EventHandlers OnSERVER_DISCONNECTED
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SERVER_DISCONNECTED"] = (EventHandlers)event_handlers["SERVER_DISCONNECTED"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SERVER_DISCONNECTED"] = (EventHandlers)event_handlers["SERVER_DISCONNECTED"] - value;
            }
        }
    }
	public event EventHandlers OnSEND_INFO
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SEND_INFO"] = (EventHandlers)event_handlers["SEND_INFO"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SEND_INFO"] = (EventHandlers)event_handlers["SEND_INFO"] - value;
            }
        }
    }
	public event EventHandlers OnRECV_INFO
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RECV_INFO"] = (EventHandlers)event_handlers["RECV_INFO"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RECV_INFO"] = (EventHandlers)event_handlers["RECV_INFO"] - value;
            }
        }
    }
	public event EventHandlers OnCALL_SECURE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CALL_SECURE"] = (EventHandlers)event_handlers["CALL_SECURE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CALL_SECURE"] = (EventHandlers)event_handlers["CALL_SECURE"] - value;
            }
        }
    }
	public event EventHandlers OnNAT
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["NAT"] = (EventHandlers)event_handlers["NAT"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["NAT"] = (EventHandlers)event_handlers["NAT"] - value;
            }
        }
    }
	public event EventHandlers OnRECORD_START
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RECORD_START"] = (EventHandlers)event_handlers["RECORD_START"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RECORD_START"] = (EventHandlers)event_handlers["RECORD_START"] - value;
            }
        }
    }
	public event EventHandlers OnRECORD_STOP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["RECORD_STOP"] = (EventHandlers)event_handlers["RECORD_STOP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["RECORD_STOP"] = (EventHandlers)event_handlers["RECORD_STOP"] - value;
            }
        }
    }
	public event EventHandlers OnPLAYBACK_START
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PLAYBACK_START"] = (EventHandlers)event_handlers["PLAYBACK_START"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PLAYBACK_START"] = (EventHandlers)event_handlers["PLAYBACK_START"] - value;
            }
        }
    }
	public event EventHandlers OnPLAYBACK_STOP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["PLAYBACK_STOP"] = (EventHandlers)event_handlers["PLAYBACK_STOP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["PLAYBACK_STOP"] = (EventHandlers)event_handlers["PLAYBACK_STOP"] - value;
            }
        }
    }
	public event EventHandlers OnCALL_UPDATE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["CALL_UPDATE"] = (EventHandlers)event_handlers["CALL_UPDATE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["CALL_UPDATE"] = (EventHandlers)event_handlers["CALL_UPDATE"] - value;
            }
        }
    }
	public event EventHandlers OnFAILURE
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["FAILURE"] = (EventHandlers)event_handlers["FAILURE"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["FAILURE"] = (EventHandlers)event_handlers["FAILURE"] - value;
            }
        }
    }
	public event EventHandlers OnSOCKET_DATA
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["SOCKET_DATA"] = (EventHandlers)event_handlers["SOCKET_DATA"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["SOCKET_DATA"] = (EventHandlers)event_handlers["SOCKET_DATA"] - value;
            }
        }
    }
	public event EventHandlers OnMEDIA_BUG_START
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MEDIA_BUG_START"] = (EventHandlers)event_handlers["MEDIA_BUG_START"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MEDIA_BUG_START"] = (EventHandlers)event_handlers["MEDIA_BUG_START"] - value;
            }
        }
    }
	public event EventHandlers OnMEDIA_BUG_STOP
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["MEDIA_BUG_STOP"] = (EventHandlers)event_handlers["MEDIA_BUG_STOP"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["MEDIA_BUG_STOP"] = (EventHandlers)event_handlers["MEDIA_BUG_STOP"] - value;
            }
        }
    }
    public event EventHandlers OnALL
    {
        add
        {
            lock (event_handlers)
            {
                event_handlers["ALL"] = (EventHandlers)event_handlers["ALL"] + value;
            }
        }
        remove
        {
            lock (event_handlers)
            {
                event_handlers["ALL"] = (EventHandlers)event_handlers["ALL"] - value;
            }
        }
    }
    #endregion
    }

    
}
