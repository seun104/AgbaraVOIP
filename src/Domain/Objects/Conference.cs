using System;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Conference 
    {
        public string Sid { get; set; }
        public  string AccountSid { get; set; }
        public  string FriendlyName { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  string Status { get; set; }
        public Conference()
        {
            Sid = "CO" + Guid.NewGuid().ToString("N");
            Status = ConferenceStatus.init;
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
        }
    }
}
