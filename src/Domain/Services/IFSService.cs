using System;
using System.Linq;
using System.Collections.Generic;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface IFSService
    {
        FSServer GetFSServer();
        IQueryable<Gateway> GetGatewaysForNumber(string Number, string FSSid);
    }
}
