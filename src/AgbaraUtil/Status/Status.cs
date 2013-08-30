using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    
    public class CallStatus
    {
        public const string queued= "queued";
        public const string ringing= "ringing";
        public const string inprogress= "in-progress";
        public const string completed= "completed";
        public const string busy= "busy";
        public const string noanswer = "no answer";
        public const string failed = "failed";
    }

    public class AccountStatus
    {
         public const string active = "active";
         public const string suspended = "suspended";
         public const string trial = "trial";
         public const string closed = "closed";
    }

    public class ConferenceStatus
    {
         public const string init = "init";
         public const string inprogress = "in-progress";
         public const string completed = "completed";
    }

    public class AccountType
    {
         public const string trial = "trial";
         public const string full = "full";
    }
    public class CallDirection
    {
         public const string inbound = "inbound";
         public const string outboundapi = "outbound-api";
         public const string outbounddial = "outbound-dial";
    }
}
