using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class HangUpRequest
    {
        public string CallUUID { get; set; }
        public string CallSid { get; set; }
    }

    public class HangUpResponse
    {
        public string Message { get; set; }
        public bool Result { get; set; }
    }
}
