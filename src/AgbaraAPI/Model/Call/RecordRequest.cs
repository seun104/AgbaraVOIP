using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CallRecordRequest
    {
        public string AccountSid { get; set; }
        public string CallSid { get; set; }
    }

    public class CallRecordResponse
    {
    }

    public class StopCallRecordRequest
    {
        public string AccountSid { get; set; }
        public string CallSid { get; set; }
    }

    public class StopCallRecordResponse
    {
    }
}
