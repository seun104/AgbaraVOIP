using System;
using System.Configuration;
using System.Collections.Generic;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
using Emmanuel.AgbaraVOIP.Freeswitch;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Core
{
    public class ApiServer
    {
        internal FSInbound fsInbound;
        private string gw = string.Empty;
        private string gw_codecs = string.Empty;
        private string gw_timeouts = string.Empty;
        private string gw_retries = string.Empty;
        private bool _run;
        private ICallService _callService;
        private IFSService _fsService;
        private static FSServer FS;
        //public ApiServer( ICallService callService, IFSService fsService, IGatewayService gatewayService)
        public ApiServer()
        {
            //_fsService = fsService;
            //_callService = callService;
            //_gatewayService = gatewayService;
            _fsService = new FSService();
            _callService = new CallService();
            FS = _fsService.GetFSServer();
            fsInbound = new FSInbound(FS.Host, FS.Port, FS.Password, FS.OutAddress);
            _run = true;

        }
        public void Start()
        {
            try
            {
                fsInbound.connect();
                _run = true;
                fsInbound.serve_forever();
                Console.WriteLine("Connected");

            }
            catch (Exception e)
            {
                Console.WriteLine("Error connecting...");
            }
        }
        public void Stop()
        {
            _run = false;
            fsInbound.exit();
        }
        public Call Call(CallRequest request)
        {
            Request call_req;
            Call call;
            call_req = PrepareCallRequest(request,out call);
            call.AccountSid = request.AccountSid;

                    fsInbound.CallRequest[call.Sid] = call_req;
                    if (fsInbound.Originate(call.Sid))
                    {
                        //save call back to database
                        try
                        {
                            _callService.AddCallLog(call);
                        }
                        catch (Exception ex)
                        {
                        }
                    }
                    
                
            return call;
        }
        public CallPlayResponse CallPlay(CallPlayRequest request)
        {
            CallPlayResponse res = new CallPlayResponse();

            return res;
        }
        public bool StopCallPlay(string CallSid)
        {
            return true;
        }
        public CallSpeakResponse CallSpeak(CallSpeakRequest request)
        {
            CallSpeakResponse res = new CallSpeakResponse();

            return res;
        }
        public CallDigitResponse CallDigit(CallDigitRequest request)
        {
            CallDigitResponse res = new CallDigitResponse();

            return res;
        }
        public CallRecordResponse CallRecord(CallRecordRequest request)
        {
            CallRecordResponse res = new CallRecordResponse();

            return res;
        }
        public bool StopCallRecord(string CallSid)
        {
            return true;
        }
        public MuteParticipantResponse ConferenceMuteParticipant(MuteParticipantRequest request)
        {
            return new MuteParticipantResponse();
        }
        public bool ConferenceKickParticipant(string ConfereceSid, string CallSid)
        {
            return true;
        }
        public ConferencePlayResponse ConferencePlay(ConferencePlayRequest request)
        {
            return new ConferencePlayResponse();
        }
        public ConferenceRecordResponse ConferenceRecord(ConferenceRecordRequest request)
        {
            return new ConferenceRecordResponse();
        }
        public ConferenceSpeakResponse  ConferenceSpeak(ConferenceSpeakRequest request)
        {
            return  new ConferenceSpeakResponse();
        }
        public bool ConferenceStopRecord(string ConfereceSid)
        {
            return true;
        }
        public bool ConferenceStopCall(string ConfereceSid)
        {
            return true;
        }
        private Request PrepareCallRequest( CallRequest request, out Call call)
        {
            call = new Call(){AccountSid=request.AccountSid, CallTo=request.To, AnswerUrl = request.AnswerUrl, Direction="outbound-api", CallerId=request.From, Timeout=request.TimeLimit};
            Queue<DialString> dialStrings = new Queue<DialString>();
            System.Text.StringBuilder args_list = new System.Text.StringBuilder();
            string sched_hangup_id = string.Empty;
            // don't allow "|" and "," in 'to' (destination) to avoid call injection
            request.To = request.To.Split(',', '|')[0];
            //set hangup_on_ring
            int hangup = 0;
            if (!int.TryParse(request.HangupOnRing, out hangup))
                hangup = -1;
            if (hangup == 0)
                args_list.Append("execute_on_ring='hangup ORIGINATOR_CANCEL',");
            else if (hangup > 0)
                args_list.Append(string.Format("execute_on_ring='sched_hangup +{0} ORIGINATOR_CANCEL',", hangup));
            // set send_digits
            if (!string.IsNullOrEmpty(request.SendDigits))
                args_list.Append(string.Format("execute_on_answer='send_dtmf {0}',", request.SendDigits));
            //set time_limit
            int time = 0;
            if (!int.TryParse(request.TimeLimit, out time))
                time = -1;
            if (time > 0)
            {
                //create sched_hangup_id
                sched_hangup_id = System.Guid.NewGuid().ToString();
                args_list.Append(string.Format("api_on_answer='sched_api {0} +{1} hupall ALLOTTED_TIMEOUT agbara_callSid {2}',", sched_hangup_id, request.TimeLimit, call.Sid));
                args_list.Append(string.Format("agbara_sched_hangup_id={0}", sched_hangup_id));
            }

            if (!string.IsNullOrEmpty(request.StatusCallbackUrl))
            {
                args_list.Append(string.Format("agbara_statuscallback_url={0},", request.StatusCallbackUrl));
                args_list.Append(string.Format("agbara_statuscallbackmethod={0},", request.StatusCallbackMethod));
            }

            if (!string.IsNullOrEmpty(request.FallbackUrl))
            {
                args_list.Append(string.Format("agbara_fallbackrul={0},", request.FallbackUrl));
                args_list.Append(string.Format("agbara_fallbackmethod={0},", request.FallbackMethod));
            }

            if (!string.IsNullOrEmpty(request.ApplicationSid))
            {
                args_list.Append(string.Format("agbara_applicationsid={0},", request.ApplicationSid));
            }

            args_list.Append(string.Format("agbara_callsid={0},", call.Sid));
            args_list.Append(string.Format("agbara_accountsid={0},", call.AccountSid));
            args_list.Append(string.Format("agbara_answer_url={0},", request.AnswerUrl));
            
            args_list.Append(string.Format("origination_caller_id_number={0},", request.From));
            
            //session_heartbeat
            args_list.Append(string.Format("enable_heartbeat_events={0},", 60));
            //Set Direction
            args_list.Append(string.Format("agbara_call_direction={0}", CallDirection.outboundapi));

            
            foreach (Gateway gate in _fsService.GetGatewaysForNumber(call.CallTo,FS.Sid))
            {
                for (int i = 1; i < int.Parse(gate.GatewayRetry); i++)
                {
                    DialString dialString = new DialString(call.Sid, request.To, gate.GatewayString, gate.GatewayCodec, gate.GatewayTimeout.ToString(), args_list.ToString());
                    dialStrings.Enqueue(dialString);
                }
            }
            Request call_req = new Request(call.Sid, dialStrings, request.AnswerUrl, request.StatusCallbackUrl, request.StatusCallbackMethod);
            return call_req;
        }
    }
}
