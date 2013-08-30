using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CallSpeakRequest
    {
        public string AccountSid { get; set; }
        public string CallSid { get; set; }
        public string Text { get; set; }
        public string Loop { get; set; }
    }

    public class CallSpeakResponse
    {
    }
}
