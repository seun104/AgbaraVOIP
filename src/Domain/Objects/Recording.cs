using System;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Recording
    {
        public string Sid { get; set; }
        public  string AccountSid { get; set; }
        public  string CallSid { get; set; }
        public  int Duration { get; set; }
        public string RecordUrl { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }

        public Recording()
        {
            Sid = "RE" + Guid.NewGuid().ToString("N");
            Duration = 0;
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
        }
    }
}

