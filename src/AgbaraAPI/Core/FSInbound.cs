using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.Freeswitch;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
using Emmanuel.AgbaraVOIP.AgbaraCommon;
using Emmanuel.AgbaraVOIP.Domain;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Core
{
    public class FSInbound : InboundSocket
    {
        const string EVENT_FILTER = "BACKGROUND_JOB CHANNEL_PROGRESS CHANNEL_PROGRESS_MEDIA CHANNEL_HANGUP CHANNEL_HANGUP_COMPLETE CHANNEL_ANSWER SESSION_HEARTBEAT HEARTBEAT";
        private string FSOutboundAddress;
        private string DefaultHTTPMethod;
        private string AuthId = "";
        private string AuthToken = "";
        private ICallService callSrvc;
        public Dictionary<string, string> BackgroundJobs;
        public Dictionary<string, Request> CallRequest;
        public Dictionary<string, int> CallElapsedTime;
        public FSInbound(string Host, int Port, string Password, string OutboundAddress = "", string DefaultHTTPMethod = "POST")
            : base(Host, Port, Password, EVENT_FILTER, 5, false)
        {
            //Call Services

            callSrvc = new CallService();
            FSOutboundAddress = OutboundAddress;
            //Mapping of Key: job-uuid - Value: CallSid
            BackgroundJobs = new Dictionary<string, string>();
            //Call Requests
            CallRequest = new Dictionary<string, Request>();
            //Elapsed Times on a call
            CallElapsedTime = new Dictionary<string, int>();

            this.DefaultHTTPMethod = DefaultHTTPMethod;
            base.OnBACKGROUND_JOB += new EventHandlers(ON_BACKGROUND_JOB);
            base.OnCHANNEL_PROGRESS += new EventHandlers(ON_CHANNEL_PROGRESS);
            base.OnCHANNEL_PROGRESS_MEDIA += new EventHandlers(ON_CHANNEL_PROGRESS_MEDIA);
            base.OnSESSION_HEARTBEAT += new EventHandlers(ON_SESSION_HEARTBEAT);
            base.OnCHANNEL_HANGUP_COMPLETE += new EventHandlers(ON_CHANNEL_HANGUP_COMPLETE);
            base.OnCHANNEL_ANSWER += new EventHandlers(ON_CHANNEL_ANSWER);
            base.OnCUSTOM += new EventHandlers(ON_CUSTOM);
            base.OnCHANNEL_BRIDGE += new EventHandlers(ON_CHANNEL_BRIDGE);
            base.OnCALL_UPDATE += new EventHandlers(ON_CALL_UPDATE);
        }
        public void ON_CUSTOM(Event ev)
        {

        }
        public void ON_CALL_UPDATE(Event ev)
        {
            //A Leg from API outbound call answered
            var agbaraFlag = ev.GetHeader("variable_agbara_app");
            //TODO
            //agbaraflag
            
                var disposition = ev.GetHeader("variable_endpoint_disposition");
                if (disposition == "ANSWER")
                {
                    var CallSid = ev.GetHeader("variable_agbara_callsid");
                    CallElapsedTime[CallSid] = 0;
                }
            

        }
        public void ON_CHANNEL_BRIDGE(Event ev)
        {
            var disposition = ev.GetHeader("variable_endpoint_disposition");
            if (disposition == "ANSWER")
            {
                var CallSid = ev.GetHeader("variable_agbara_callsid");
                CallElapsedTime[CallSid] = 0;
                Call call = new Call();
                //get call B Sid
                call.Sid = ev.GetHeader("variable_agbara_bleg_callsid");
                CallElapsedTime[call.Sid] = 0;
                call.Direction = CallDirection.outbounddial;
                call.Status = CallStatus.inprogress;
                try
                {
                    callSrvc.AddCallLog(call);
                }
                catch (Exception ex)
                {
                    throw ex;
                }
            }
        }
        public void ON_SESSION_HEARTBEAT(Event ev)
        {
            //TODO
            //store in db

            long answer_seconds_since_epoch = long.Parse(ev.GetHeader("Caller-Channel-Answered-Time"));
            long heartbeat_seconds_since_epoch = long.Parse(ev.GetHeader("Event-Date-Timestamp"));
            int ElapsedTime = EpochTimeConverter.GetEpochTimeDifferent(heartbeat_seconds_since_epoch, answer_seconds_since_epoch);
            var BCallSid = ev.GetHeader("variable_agbara_bleg_callsid");
            var ACallSid = ev.GetHeader("variable_agbara_callsid");
            if (!string.IsNullOrEmpty(BCallSid))
            {
                int current = CallElapsedTime[BCallSid];
                CallElapsedTime[BCallSid] = current + ElapsedTime;
            }
            else if (!string.IsNullOrEmpty(ACallSid))
            {
                int current = CallElapsedTime[ACallSid];
                CallElapsedTime[ACallSid] = current + ElapsedTime;
            }

            //TODO
            //bill user real time here

        }
        public void ON_CHANNEL_ANSWER(Event ev)
        {
            //Set up call elapsed
            var CallSid = ev.GetHeader("variable_agbara_callsid");
            CallElapsedTime[CallSid] = 0;
        }

        public void ON_BACKGROUND_JOB(Event ev)
        {
            Request call_req = new Request(null, null, null, null, null);

            string status = string.Empty;
            string reason = string.Empty;
            //Capture Job Event
            // Capture background job only for originate and ignore all other jobs
            string job_cmd = ev.GetHeader("Job-Command");
            string job_uuid = ev.GetHeader("Job-UUID");
            if (job_cmd == "originate" && string.IsNullOrEmpty(job_uuid))
            {
                try
                {
                    string data = ev.GetBody();
                    int pos = data.IndexOf(' ');
                    if (pos != -1)
                    {
                        status = data.Substring(0, pos).Trim();
                        reason = data.Substring(pos + 1).Trim();
                    }
                }
                catch (Exception ex)
                {
                    return;
                }
                string CallSid = BackgroundJobs[job_uuid];
                if (string.IsNullOrEmpty(CallSid))
                {
                    return;
                }
                try
                {
                    call_req = CallRequest[CallSid];
                }
                catch (Exception ex)
                {
                    return;
                }
                //Handle failure case of originate
                //This case does not raise a on_channel_hangup event.
                //All other failures will be captured by on_channel_hangup
                if (!status.Contains("OK"))
                    //In case ring/early state done, just warn releasing call request will be done in hangup event
                    if (call_req.state_flag == "Ringing" || call_req.state_flag == "EarlyMedia")
                    {
                        return;
                    }
                    //If no more gateways, release call request
                    else if (call_req.gateways.Count == 0)
                    {
                        //set an empty call_uuid
                        string call_uuid = "";
                        string hangup_url = call_req.hangup_url;
                        SetHangUpComplete(CallSid, call_uuid, reason, ev, hangup_url);
                        return;
                    }
                    //If there are gateways and call request state_flag is not set
                    //try again a call
                    else if (call_req.gateways.Count > 0)
                        Originate(CallSid);

            }
        }
        public void ON_CHANNEL_PROGRESS(Event ev)
        {
            Request call_req;
            string CallSid = ev.GetHeader("variable_agbara_CallSid");
            string direction = ev.GetHeader("Call-Direction");
            // Detect ringing state
            if (!string.IsNullOrEmpty(CallSid) && direction == "outbound")
            {
                try
                {
                    call_req = CallRequest[CallSid];
                }
                catch (Exception ex)
                {
                    return;
                }
                //only send if not already ringing/early state
                if (string.IsNullOrEmpty(call_req.state_flag))
                {
                    // set state flag to true
                    call_req.state_flag = "Ringing";
                    //clear gateways to avoid retry
                    call_req.gateways.Clear();
                    string called_num = ev.GetHeader("Caller-Destination-Number");
                    string caller_num = ev.GetHeader("Caller-Caller-ID-Number");
                    ///self.log.info("Call from %s to %s Ringing for RequestUUID %s" (caller_num, called_num, CallSid))
                    // send ring if ring_url found
                    string ring_url = call_req.ring_url;
                    if (!string.IsNullOrEmpty(ring_url))
                    {
                        SortedList param = new SortedList();
                        param.Add("RequestUUID", CallSid);
                        param.Add("Direction", direction);
                        param.Add("To", called_num);
                        param.Add("CallStatus", "ringing");
                        param.Add("From", caller_num);
                        string response = Task<string>.Factory.StartNew(() => SendToUrl(ring_url, param, string.Empty)).Result;
                    }
                }
            }

        }
        public void ON_CHANNEL_PROGRESS_MEDIA(Event ev)
        {
            Request call_req;
            string CallSid = ev.GetHeader("variable_agbara_CallSid");
            string direction = ev.GetHeader("Call-Direction");
            //Detect early media state
            if (!string.IsNullOrEmpty(CallSid) && direction == "outbound")
            {
                try
                {
                    call_req = CallRequest[CallSid];
                }
                catch (Exception ex)
                {
                    return;
                }
                // only send if not already ringing/early state
                if (string.IsNullOrEmpty(call_req.state_flag))
                    //# set state flag to true
                    call_req.state_flag = "EarlyMedia";
                // clear gateways to avoid retry
                call_req.gateways.Clear();
                string called_num = ev.GetHeader("Caller-Destination-Number");
                string caller_num = ev.GetHeader("Caller-Caller-ID-Number");
                // send ring if ring_url found
                string ring_url = call_req.ring_url;
                if (!string.IsNullOrEmpty(ring_url))
                {
                    SortedList param = new SortedList();
                    param.Add("RequestUUID", CallSid);
                    param.Add("Direction", direction);
                    param.Add("To", called_num);
                    param.Add("CallStatus", "ringing");
                    param.Add("From", caller_num);
                    string response = Task<string>.Factory.StartNew(() => SendToUrl(ring_url, param, string.Empty)).Result;
                }
            }
        }
        public void ON_CHANNEL_HANGUP_COMPLETE(Event ev)
        {
            Request call_req;
            string CallSid = ev.GetHeader("variable_agbara_callsid");
            string direction = ev.GetHeader("Call-Direction");
            if (!string.IsNullOrEmpty(CallSid) && direction != "outbound")
                return;
            string call_uuid = ev.GetHeader("Unique-ID");
            string reason = ev.GetHeader("Hangup-Cause");
            try
            {
                call_req = CallRequest[CallSid];
            }
            catch (Exception ex)
            {
                return;
            }
            // If there are gateways to try again, spawn originate
            if (call_req.gateways.Count > 0)
            {
                Originate(CallSid);
                return;
            }
            else // Else clean call request
            {
                string hangup_url = call_req.hangup_url;
                SetHangUpComplete(CallSid, call_uuid, reason, ev, hangup_url);
            }

        }
        private void SetHangUpComplete(string CallSid, string call_uuid, string reason, Event ev, string hangup_url)
        {
           

            long answer_seconds_since_epoch = long.Parse(ev.GetHeader("Caller-Channel-Answered-Time"));
            long hangup_seconds_since_epoch = long.Parse(ev.GetHeader("Caller-Channel-Hangup-Time"));
            string called_num = ev.GetHeader("Caller-Destination-Number");
                string caller_num = ev.GetHeader("Caller-Caller-ID-Number");
            string direction = ev.GetHeader("variable_agbara_call_direction");
            //get call details
            Call call = callSrvc.GetCallDetail(CallSid);
            call.Status = CallStatus.completed;
            call.CallerId = caller_num;
            call.CallTo = called_num;
            call.DateUpdated = DateTime.Now;
            call.StartTime = EpochTimeConverter.ConvertFromEpochTime(answer_seconds_since_epoch);
            call.EndTime = EpochTimeConverter.ConvertFromEpochTime(hangup_seconds_since_epoch);
            call.Duration = EpochTimeConverter.GetEpochTimeDifferent(hangup_seconds_since_epoch, answer_seconds_since_epoch);
            call.Direction = direction;

             try
            {
                CallRequest.Remove(CallSid);
                CallElapsedTime.Remove(CallSid);
            }
            catch (Exception ex)
            {
            }

             try
             {
                 callSrvc.UpdateCallLog(call);
             }
             catch (Exception ex)
             {
             }
        }
        private string SendToUrl(string url, SortedList param, string method)
        {
            string data = string.Empty;
            if (string.IsNullOrEmpty(method))
                method = DefaultHTTPMethod;

            if (string.IsNullOrEmpty(url))
            {
                return string.Empty;
            }
            HTTPRequest HttpObj = new HTTPRequest(AuthId, AuthToken);
            try
            {
                data = HttpObj.FetchResponse(url, method, param);
            }
            catch (Exception ex)
            {
                return "";
            }
            return data;
        }
        public bool Originate(string CallSid)
        {
            Request CallReq;
            bool ret = false;
            string outbound_str = string.Empty;
            DialString gw = new DialString(null, null, null, null, null, null);
            try
            {
                CallReq = CallRequest[CallSid];
            }
            catch (Exception ex)
            {
                return false;
            }

            try
            {
                gw = CallReq.gateways.Dequeue();
            }
            catch (Exception ex)
            {
                CallRequest.Remove(CallSid);
                return false;
            }


            StringBuilder _options = new StringBuilder();
            //Set agbara app flag
            _options.Append("agbara_app=true,");
            if (!string.IsNullOrEmpty(gw.timeout))
            {
                _options.Append(string.Format("originate_timeout={0},", gw.timeout));
            }
            _options.Append("ignore_early_media=true,");
            if (!string.IsNullOrEmpty(gw.codecs))
            {
                _options.Append(string.Format("absolute_codec_string={0}", gw.codecs));
            }

            if (!string.IsNullOrEmpty(FSOutboundAddress))
            {
                outbound_str = string.Format("'socket:{0} async full' inline", FSOutboundAddress);
            }
            else
            {
                outbound_str = "&park()";
            }

            string dial_str = string.Format("originate {0}{1},{2}{3}{4}/{5} {6}", "{", gw.extra_dial_string, _options, "}", gw.gw, gw.to, outbound_str);
            BgApiResponse ApiResponse = (BgApiResponse)Task<Event>.Factory.StartNew(() => BgAPICommand(dial_str)).Result;

            if (ApiResponse.IsSuccess())
            {
                string job_uuid = ApiResponse.GetJobUUID();
                if (!string.IsNullOrEmpty(job_uuid))
                {

                    BackgroundJobs[job_uuid] = CallSid;
                    ret = true;

                }
                else
                {
                    ret = false;
                }
            }
            return ret;
        }
        public bool Play(string CallSid, string PlayUrl, string Loop, string Legs)
        {

            //Todo
            return true;
        }
        public bool Speak(string CallSid, string Text)
        {
            //Todo
            return true;
        }
        public bool Record()
        {
            //Todo
            return true;
        }
        public bool SendDigit()
        {
            //Todo
            return true;
        }
        public bool BulkOriginate(List<string> CallSidList)
        {
            bool ret;

            if (CallSidList.Count > 0)
            {
                foreach (string reqId in CallSidList)
                {
                    string CallSid = reqId;
                    Task.Factory.StartNew(() => Originate(CallSid));
                }
                ret = true;
            }
            else
            {
                ret = false;
            }
            return ret;
        }
        public bool HangupCall(string CallUUId = "", string CallSid = "")
        {
            bool ret;
            string cmd;
            if (string.IsNullOrEmpty(CallUUId) && string.IsNullOrEmpty(CallSid))
            {
                ret = false;
            }

            if (!string.IsNullOrEmpty(CallUUId))
            {
                cmd = string.Format("uuid_kill{0} NORMAL_CLEARING", CallUUId);
            }
            else  //Use request uuid
            {
                try
                {
                    Request call_req = CallRequest[CallSid];
                }
                catch (Exception ex)
                {

                    ret = false;
                    return ret;
                }
                cmd = string.Format("hupall NORMAL_CLEARING agbara_CallSid {0}", CallSid);
            }
            APIResponse res = (APIResponse)APICommand(cmd);
            if (!res.IsSuccess())
            {
                return false;
            }
            return true;
        }
        public bool HangupAllCalls()
        {
            bool ret = false;
            BgApiResponse bg_api_response = (BgApiResponse)BgAPICommand("hupall NORMAL_CLEARING");
            string job_uuid = bg_api_response.GetJobUUID();
            if (string.IsNullOrEmpty(job_uuid))
            {
                //log.error("Hangup All Calls Failed -- JobUUID not received")
                ret = false;
            }
            else
            {
                ret = true;
            }
            //log.info("Executed Hangup for all calls")
            return ret;
        }
    }
}
