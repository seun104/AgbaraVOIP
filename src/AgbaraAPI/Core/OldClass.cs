using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Core
{
    class OldClass
    {
        private HangUpResponse HangUpCall(HangUpRequest request)
        {
            HangUpResponse response = new HangUpResponse();

            if (string.IsNullOrEmpty(request.CallUUID) && string.IsNullOrEmpty(request.CallSid))
            {
                response.Message = "CallUUID or RequestUUID Parameter must be present";
                response.Result = false;
                return response;
            }

            if (fsInbound.HangupCall(request.CallUUID, request.CallSid))
            {
                response.Message = "Hangup Call Executed";
                response.Result = true;
                return response;
            }
            else
            {
                response.Message = "Hangup Call Failed";
                response.Result = false;
                return response;
            }

        }
        private HangUpAllResponse HangUpAllCall()
        {
            HangUpAllResponse response = new HangUpAllResponse();

            //Hangup All Live Calls in the system

            if (fsInbound.HangupAllCalls())
            {
                response.Message = "All Calls Hungup Successfuly";
                response.Result = true;
            }
            else
            {
                response.Message = "Calls Hangup failed!";
                response.Result = false;
            }


            return response;
        }
        private ScheduleHangUpResponse ScheduleHangUpCall(ScheduleHangUpRequest request)
        {
            ScheduleHangUpResponse response = new ScheduleHangUpResponse();

            if (string.IsNullOrEmpty(request.CallUUID))
            {
                response.Message = "CallUUID Parameter must be present";
                response.Result = false;
            }
            else if (string.IsNullOrEmpty(request.Time))
            {
                response.Message = "Time Parameter must be present";
            }
            else
            {
                try
                {
                    int time = 0;
                    if (int.TryParse(request.Time, out time))
                    {
                        if (time <= 0)
                        {
                            response.Message = "Time Parameter must be > 0 !";
                            response.Result = false;
                        }
                        else
                        {
                            string sched_id = Guid.NewGuid().ToString();
                            APIResponse res = (APIResponse)fsInbound.APICommand(string.Format("sched_api {0} +{1} uuid_kill {2} ALLOTTED_TIMEOUT", sched_id, time, request.CallUUID));
                            if (res.IsSuccess())
                            {
                                response.Message = string.Format("Scheduled Hangup Done with SchedHangupId {0}", sched_id);
                                response.Result = true;
                                response.ScheduleId = sched_id;
                            }
                            else
                            {
                                response.Message = string.Format("Scheduled Hangup Failed: {0}", res.GetResponse());
                                response.Result = false;
                            }
                        }
                    }
                }
                catch (Exception ex)
                {
                    response.Message = "Invalid Time Parameter !";
                    response.Result = false;
                }

            }
            return response;
        }
        private CancelScheduledHangUpResponse CancelScheduleHangUpCall(CancelScheduledHangUpRequest request)
        {
            CancelScheduledHangUpResponse response = new CancelScheduledHangUpResponse();
            if (!string.IsNullOrEmpty(request.ScheduleId))
            {
                response.Message = "Id Parameter must be present";
                response.Result = false;
            }

            else
            {
                APIResponse res = (APIResponse)fsInbound.APICommand(string.Format("sched_del {0}", request.ScheduleId));
                if (res.IsSuccess())
                {
                    response.Message = "Scheduled Hangup Canceled";
                    response.Result = true;
                }
                else
                {
                    response.Message = string.Format("Scheduled Hangup Cancelation Failed: {0}", res.GetResponse());
                    response.Result = false;
                }
            }
            return response;
        }
    }
}
