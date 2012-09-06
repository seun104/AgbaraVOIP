using System;
using System.Collections.Generic;
using System.Collections.Specialized;
using System.Linq;
using System.Text;
using System.IO;
using fastJSON;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{
    public class Event
    {

        #region Field

        private string stCoreUUID;
        private int inEventCallingLineNumber;
        private DateTime dtEventDateGMT;
        private DateTime dtEventDateLocal;
        internal Dictionary<string,string> _headers = new Dictionary<string,string>();
        internal string _raw_body = string.Empty;

        #endregion Field

        #region Property

        #region Minimum event information

        public string EventName
        {
            get { return GetHeader("Event-Name"); }
        }

        public string CoreUUID
        {
            get { return stCoreUUID; }
        }

        public DateTime EventDateLocal
        {
            get { return dtEventDateLocal; }
        }

        public DateTime EventDateGMT
        {
            get { return dtEventDateGMT; }
        }

        public string EventCallingFile
        {
            get { return GetHeader("Event-Calling-File"); }
        }

        public string EventCallingFunction
        {
            get { return GetHeader("Event-Calling-Function"); }
        }

        public int EventCallingLineNumber
        {
            get { return inEventCallingLineNumber; }
        }
        #endregion Minimum event information

        #endregion Property

        #region Constructor
        public Event(string buffer = "")
        {
            if (!string.IsNullOrEmpty(buffer))
            {
                foreach (string data in buffer.Split(new Char[] { '\n' }))
                {
                    int pos = data.IndexOf(':');
                    if (pos != -1)
                    {
                        string name = data.Substring(0, pos).Trim();
                        string value = data.Substring(pos + 1).Trim();
                        SetHeader(name, value);
                    }
                }
            }
        }
        #endregion Constructor

        #region Public Member

        public int GetContentLength()
        {
            /*
               Gets Content-Length header as integer.

               Returns 0 If length not found.
               */
            int ret = 0;
            string length = GetHeader("Content-Length");
            if (!string.IsNullOrEmpty(length))
            {
                try
                {
                    int.TryParse(length, out ret);
                }
                catch
                {
                    return ret;
                }
            }
            return ret;

        }

        public string GetReplyText()
        {
            /*
               Gets Reply-Text header as string.
               Returns None if header not found.
              */
            string reply = GetHeader("Reply-Text");

            return reply;
        }

        public bool IsReplyTextSuccess()
        {
            /*
               Returns True if ReplyText header contains OK.
               Returns False otherwise.
              */
            return GetReplyText().Contains("OK");
        }

        public string GetContentType()
        {
            /*
               Gets Content-Type header as string.
               Returns None if header not found.
              */
            return GetHeader("Content-Type");
        }

        public Dictionary<string,string> GetHeaders()
        {
            /*
               Gets all headers as a python dict.
              */
            return _headers;
        }

        public void SetHeaders(Dictionary<string,string> headers)
        {
            /*
             Sets all headers from dict.
             */
            foreach(string key in headers.Keys)
            {
                _headers.Add(key, headers[key]);
            }
        }

        public string GetHeader(string key, string defaultvalue = "")
        {
            /*
               Gets a specific header as string.
               Returns None if header not found.
              */
            string value = "";
            
                try
                {
                    value = _headers[key];
                }
                catch(Exception ex)
                {
                    return defaultvalue;
                }
            
            return value;
        }

        public void SetHeader(string key, string value)
        {
            /*
               Sets a specific header.
              */
            _headers.Add(key, value);
        }

        public string GetBody()
        {
            /*
               Gets raw Event body.
              */
            return _raw_body;
        }

        public void SetBody(string data)
        {
            /*
               Sets raw Event body.
              */
            _raw_body = data;
        }

        public override string ToString()
        {
            return string.Format("{0} headers ={1} , body ={2}", "Event", _headers.ToString(), _raw_body.ToString());
        }
        #endregion Public Member

        #region Private Member

        private void LoadDefaultProperties()
        {
            stCoreUUID = GetHeader("core-UUID");
            string stTemp = GetHeader("event-date-local");
            dtEventDateLocal = String.IsNullOrEmpty(stTemp) ? DateTime.Now : DateTime.Parse(Uri.UnescapeDataString(stTemp));

            stTemp = GetHeader("event-date-GMT");
            dtEventDateGMT = String.IsNullOrEmpty(stTemp) ? DateTime.Now : DateTime.Parse(Uri.UnescapeDataString(stTemp));

            stTemp = GetHeader("event-calling-line-number");
            inEventCallingLineNumber = String.IsNullOrEmpty(stTemp) ? -1 : int.Parse(stTemp);
        }
        
        #endregion Private Member
    }

    public class APIResponse : Event
    {
        public APIResponse(string buffer)
        {
            base._headers = (Dictionary<string, string>)JSON.Instance.ToObject(buffer);
        }
        public static APIResponse Cast(Event evnt)
        {
            APIResponse cls = new APIResponse("");
                cls._headers = evnt._headers;
                cls._raw_body = evnt._raw_body;
            return cls;
        }
        public string GetResponse()
        {
            string Message = string.Empty;

            string replyText = base.GetHeader("reply-text") ?? "+OK";
            string body = base.GetBody().Trim();
            if (!string.IsNullOrEmpty(body))
            {
                Message = body;
            }
            else
            {
                int pos = replyText.IndexOf(' ');
                if (pos != -1)
                    Message = replyText.Substring(pos + 1);
            }

            return Message;
        }
        public bool IsSuccess()
        {
            return base.IsReplyTextSuccess();
        }
        public override string ToString()
        {
            return string.Format("{0} headers ={1} , body ={2}", "APIResponse", base._headers.ToString(), base._raw_body.ToString());
        }
    }

    public class BgApiResponse : Event
    {
        public BgApiResponse(string buffer)
        {
            
            base._headers = (Dictionary<string, string>)JSON.Instance.ToObject(buffer);
        }
        public static BgApiResponse Cast(Event evnt)
        {
            BgApiResponse cls = new BgApiResponse("");
            
                cls._headers = evnt._headers;
                cls._raw_body = evnt._raw_body;

            return cls;
        }
        public string GetResponse()
        {
            string Message = string.Empty;

            string replyText = base.GetHeader("reply-text") ?? "+OK";
            string body = base.GetBody().Trim();
            if (!string.IsNullOrEmpty(body))
            {
                Message = body;
            }
            else
            {
                int pos = replyText.IndexOf(' ');
                if (pos != -1)
                    Message = replyText.Substring(pos + 1);
            }

            return Message;
        }
        public bool IsSuccess()
        {
            return base.IsReplyTextSuccess();
        }
        public string GetJobUUID()
        {
            return base.GetHeader("Job-UUID");
        }
        public override string ToString()
        {
            return string.Format("{0} headers ={1} , body ={2}", "BgApiResponse", base._headers.ToString(), base._raw_body.ToString());
        }
    }

    public class CommandResponse : Event
    {
        public CommandResponse(string buffer)
        {
            base._headers = (Dictionary<string, string>)JSON.Instance.ToObject(buffer);
        }
        public static CommandResponse Cast(Event evnt)
        {
            CommandResponse cls = new CommandResponse("");
           
                cls._headers = evnt._headers;
                cls._raw_body = evnt._raw_body;
            return cls;
        }

        public string GetResponse()
        {

            string Message = string.Empty;

            string replyText = base.GetHeader("reply-text") ?? "+OK";
            string body = base.GetBody().Trim();
            if (!string.IsNullOrEmpty(body))
            {
                Message = body;
            }
            else
            {
                int pos = replyText.IndexOf(' ');
                if (pos != -1)
                    Message = replyText.Substring(pos + 1);
            }

            return Message;
        }
        public bool IsSuccess()
        {
            return base.IsReplyTextSuccess();
        }
        public override string ToString()
        {
            return string.Format("{0} headers ={1} , body ={2}", "CommandResponse", base._headers.ToString(), base._raw_body.ToString());
        }
    }

    public class JsonEvent :Event
    {
        #region Constructor
        public JsonEvent(string buffer)
        {
            try
            {
                _headers =(Dictionary<string,string>) JSON.Instance.ToObject(buffer);
                    //JsonSerializer.DeserializeFromReader<Dictionary<string, string>>(new StringReader( buffer));
            }
            catch (Exception ex)
            {
            }
            _raw_body = _headers["_body"];
        }
        #endregion Constructor

        #region Private Member
        public override string ToString()
        {
            return string.Format("{0} headers ={1} , body ={2}", "JsonEvent", _headers.ToString(), _raw_body.ToString());
        }
        #endregion Private Member
    }
}
