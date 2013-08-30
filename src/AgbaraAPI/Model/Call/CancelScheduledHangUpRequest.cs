using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CancelScheduledHangUpRequest
    {
        public string ScheduleId { get; set; }
    }

    public class CancelScheduledHangUpResponse
    {
        public string Message { get; set; }
        public bool Result { get; set; }
        public string RequestUUID { get; set; }
    }
}
