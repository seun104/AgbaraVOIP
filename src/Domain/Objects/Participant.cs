using System;
namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Participant
    {
        public string CallSid { get; set; }
        public  string AccountSid { get; set; }
        public  string ConferenceSid { get; set; }
        public  string FriendlyName { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  string Muted {get; set;}
        public  string StartConferenceOnEnter { get; set; }
        public  string EndConferenceOnExit { get; set; }
        public Participant()
        {
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
        }
    }
    
}
