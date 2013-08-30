using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class ApplicationService : IApplicationService
    {
        public Objects.Application CreateApplication(string FriendlyName, string AccountSid)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.Application> GetAllApplications(string AccountSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Application GetApplication(string AccountSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Application ModifyApplication(string AccountSid, string Status)
        {
            throw new NotImplementedException();
        }
    }
}
