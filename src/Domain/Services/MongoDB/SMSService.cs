using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class SMSService: ISMSService
    {
        public Objects.SMS CreateSMSEntry(string CallSid, string AccountSid, int Duration)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.SMS> GetAllSMSs(string AccountSid)
        {
            throw new NotImplementedException();
        }

        public Objects.SMS GetSMS(string SMSSid)
        {
            throw new NotImplementedException();
        }
    }
}
