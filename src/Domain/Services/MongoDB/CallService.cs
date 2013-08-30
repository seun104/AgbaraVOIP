using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.AgbaraCommon;
using DreamSongs.MongoRepository;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class CallService :ICallService
    {
        private  MongoRepository<Call> callRepo = new MongoRepository<Call>();

        public IEnumerable<Call> GetAllCalls(string AccountSid)
        {
            return callRepo.All(a => a.AccountSid == AccountSid);
        }

        public Call GetCallDetail(string CallId)
        {
            throw new NotImplementedException();
        }


        public Call AddCallLog(Call call)
        {
            
            try
            {
                callRepo.Add(call);
            }
            catch (Exception ex)
            {
                call = new Call();
            }

            return call;
        }

        public Call UpdateCallStatus(string CallSId, String Status)
        {
            using (callRepo.RequestStart())
            {
                var call = callRepo.GetSingle(a => a.Sid == CallSId);

                try
                {
                    call.Status = Status;
                    callRepo.Update(call);
                }
                catch (Exception ex)
                {
                    call = new Call();
                }
                return call;
            }
        }

        public Call UpdateCallLog(Call call)
        {
                  using (callRepo.RequestStart())
            {

                try
                {
                    callRepo.Update(call);
                }
                catch (Exception ex)
                {
                    call = new Call();
                }
                return call;
            }
        
        }
    }
}
