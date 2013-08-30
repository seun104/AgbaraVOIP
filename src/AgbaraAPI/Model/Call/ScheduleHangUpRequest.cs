using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class ScheduleHangUpRequest
    {
        public string Time { get; set; }
        public string CallUUID { get; set; }
    }

    public class ScheduleHangUpResponse
    {
        public string Message { get; set; }
        public string ScheduleId { get; set; }
        public bool Result { get; set; }
    }
}
