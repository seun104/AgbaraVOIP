using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CallPlayRequest
    {
        public string AccountSid { get; set; }
        public string CallSid { get; set; }
        public string PlayUrl { get; set; }
        public string Loop { get; set; }
        public string Legs { get; set; }
    }

    public class CallPlayResponse
    {
        public string CallSid { get; set; }
        public string Mesage { get; set; }
    }

    public class StopCallPlayRequest
    {
        public string AccountSid { get; set; }
        public string CallSid { get; set; }
    }

    public class StopCallPlayResponse
    {
        public string CallSid { get; set; }
        public string Mesage { get; set; }
    }
}
