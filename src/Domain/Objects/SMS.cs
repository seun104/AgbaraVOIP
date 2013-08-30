using System;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class SMS 
    {
        public string Sid { get; set; }
        public  string AccountSid { get; set; }
        public  string From { get; set; }
        public  string To { get; set; }
        public  string Body { get; set; }
        public  string Status { get; set; }
        public  string Direction { get; set; }
        public  int Duration { get; set; }
        public  decimal Price { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  DateTime DateSent { get; set; }
        public SMS()
        {
            Status = CallStatus.queued.ToString();
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
            DateSent = DateTime.Now;
            Sid = "SM" + Guid.NewGuid().ToString("N");
        }
    }
}
