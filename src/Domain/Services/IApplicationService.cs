using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface IApplicationService
    {
        Application CreateApplication(string FriendlyName, string AccountSid);
        IQueryable<Application> GetAllApplications(string AccountSid);
        Application GetApplication(string AccountSid);
        Application ModifyApplication(string AccountSid, string Status);
    }
}

