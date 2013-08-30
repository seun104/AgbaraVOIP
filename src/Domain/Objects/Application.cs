using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Application 
    {
        public string Sid { get; set; }
        public  string AccountSid { get; set; }
        public  string FriendlyName { get; set; }
        public  string VoiceUrl { get; set; }
        public  string VoiceMethod { get; set; }
        public  string VoiceFallbackUrl { get; set; }
        public  string VoiceFallbackMethod { get; set; }
        public  string StatusCallback { get; set; }
        public  string StatusCallbackMethod { get; set; }
        public  string SmsUrl { get; set; }
        public  string SmsMethod { get; set; }
        public  string SmsFallbackUrl { get; set; }
        public  string SmsFallbackMethod { get; set; }
        public  string SmsStatusCallback { get; set; }
        public  string SmsStatusCallbackMethod { get; set; }
        public  string HeartbeatUrl { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public Application()
        {
            Sid = "AP" + Guid.NewGuid().ToString("N");
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
        }

        
    }
}
