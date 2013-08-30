using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class Request
    {
        public string CallSid { get; set; }
        public string answer_url { get; set; }
        public string ring_url { get; set; }
        public string hangup_url { get; set; }
        public string state_flag { get; set; }
        public Queue<DialString> gateways { get; set; }

        public Request(string CallSid, Queue<DialString> gateways, string answer_url, string ring_url, string hangup_url)
        {
            this.CallSid = CallSid;
            this.gateways = gateways;
            this.answer_url = answer_url;
            this.ring_url = ring_url;
            this.hangup_url = hangup_url;
            this.state_flag = "";
        }
    }
}
