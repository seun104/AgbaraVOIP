using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class DialString
    {
        public string callSid { get; set; }
        public string to { get; set; }
        public string gw { get; set; }
        public string codecs { get; set; }
        public string timeout { get; set; }
        public string extra_dial_string { get; set; }

        public DialString(string CallSid, string to, string gw, string codec, string timeout, string extra_dial_string)
        {
            this.callSid = CallSid;
            this.to = to;
            this.gw = gw;
            this.codecs = codec;
            this.timeout = timeout;
            this.extra_dial_string = extra_dial_string;
        }

    }
}
