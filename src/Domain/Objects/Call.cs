using System;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Call 
    {
        public string Sid { get; set; }
        public  string AccountSid { get; set; }
        public  string CallerId { get; set; }
        public  string CallTo { get; set; }
        public  string AnswerUrl { get; set; }
        public  string Status { get; set; }
        public  string Timeout { get; set; }
        public  string Direction { get; set; }
        public  int Duration { get; set; }
        public  decimal Price { get; set; }
        public  DateTime StartTime { get; set; }
        public  DateTime EndTime { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  string AnsweredBy { get; set; }
        public Call()
        {
            Duration = 0;
            StartTime = DateTime.Now;
            EndTime = DateTime.Now;
            Status = CallStatus.queued.ToString();
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
            Sid = "CA" + Guid.NewGuid().ToString("N");
        }
    }
}
