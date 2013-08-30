using System;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Account 
    {
        public string Sid { get; set; }
        public  string ParentSid { get; set; }
        public  string FriendlyName { get; set; }
        public string PhoneNumber { get; set; }
        public  DateTime DateCreated { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  string Type { get; set; }
        public  string Status { get; set; }
        public  string AuthToken { get; set; }

        public Account()
        {
            Sid = "AC" + Guid.NewGuid().ToString("N");
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
            Status = AccountStatus.active;
            Type = AccountType.trial;
        }
    }
}
