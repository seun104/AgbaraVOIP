using System;
using System.Collections.Generic;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface ICallService 
    {
        Call AddCallLog(Call Log);
        Call UpdateCallStatus(string CallSId, string Status);
        Call UpdateCallLog(Call Log);
        Call GetCallDetail(string CallId);
        IEnumerable<Call> GetAllCalls(string AccountSid);
    }
}
