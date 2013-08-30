using System;
using System.Linq;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface ISMSService
    {
        SMS CreateSMSEntry(string CallSid, string AccountSid, int Duration);
        IQueryable<SMS> GetAllSMSs(string AccountSid);
        SMS GetSMS(string SMSSid);
    }
}
