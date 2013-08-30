using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using System.Configuration;
using Emmanuel.AgbaraVOIP.AgbaraCommon;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Dial : Element
    {
        const int DEFAULT_TIMEOUT = 30;
        const int DEFAULT_TIMELIMIT = 14400;
        string method;
        string action;
        bool record;
        bool hangupOnStar = false;
        string caller_id;
        int time_limit = 0;
        int timeout = 0;
        string dial_str;
        private IFSService fs;
        public Dial()
            : base()
        {
            Nestables.Add("Number");
            Nestables.Add("Conference");
            this.method = "";
            this.action = "";
            this.record = false;
            this.hangupOnStar = false;
            this.caller_id = "";
            this.time_limit = DEFAULT_TIMELIMIT;
            this.timeout = DEFAULT_TIMEOUT;
            this.dial_str = "";
            fs = new FSService();
        }
        public override void ParseElement(XElement elements, string uri = "")
        {
            base.ParseElement(elements, uri);
            this.method = ExtractAttributeValue("method");
            action = ExtractAttributeValue("action");
            caller_id = ExtractAttributeValue("callerId");
            int.TryParse(ExtractAttributeValue("timeout", "30"), out timeout);
            if (timeout <= 0)
                timeout = DEFAULT_TIMEOUT;
            int.TryParse(ExtractAttributeValue("timeLimit"), out time_limit);
            if (time_limit <= 0)
                time_limit = DEFAULT_TIMELIMIT;
            bool.TryParse(ExtractAttributeValue("hangupOnStar"), out hangupOnStar);
            bool.TryParse(ExtractAttributeValue("record"), out record);

        }
        private List<string> _prepare_play_string(FSOutbound outboundClient, string remote_url)
        {
            List<string> sound_files = new List<string>();
            if (!string.IsNullOrEmpty(remote_url))
                return sound_files;
            try
            {
                string response = outboundClient.SendToUrl(remote_url, outboundClient.session_params, this.method);
                XElement doc = XElement.Load(response);
                if (doc.Name != "Response")
                    return sound_files;

                // build play string from remote restxml
                foreach (XElement ele in doc.Nodes())
                {
                    //Play element
                    if (ele.Name == "Play")
                    {
                        var child = new Play();
                        child.ParseElement(ele);
                        string sound_file = child.SoundFilePath;
                        if (!string.IsNullOrEmpty(sound_file))
                        {
                            var loop = child.LoopTimes;
                            if (loop == 0)
                                loop = 1000;  // Add a high number to Play infinitely
                            // Play the file loop number of times
                            for (int i = 1; i < loop; i++)
                                sound_files.Add(sound_file);
                            //Infinite Loop, so ignore other children
                            if (loop == 1000)
                                break;
                        }
                    }
                    //Say element
                    else if (ele.Name == "Say")
                    {
                        var child = new Say();
                        child.ParseElement(ele);
                        var text = child.Text;
                        // escape simple quote
                        text = text.Replace("'", "\\'");
                        var loop = child.loop_times;
                        var engine = child.engine;
                        var voice = child.voice;
                        var say_str = string.Format("say:{0}:{1}:'{2}'", engine, voice, text);
                        for (int i = 1; i < loop; i++)
                            sound_files.Add(say_str);
                    }
                    //Pause element
                    else if (ele.Name == "Pause")
                    {
                        var child = new Pause();
                        child.ParseElement(ele);
                        var pause_secs = child.length;
                        string pause_str = string.Format("file_string://silence_stream://{0}", (pause_secs * 1000));
                        sound_files.Add(pause_str);
                    }
                }
            }
            catch (Exception e)
            {
            }
            return sound_files;

        }
        private string PrepareNumber(Number Num, FSOutbound outboundClient)
        {
            List<string> num_gw = new List<string>();
            string option_send_digits = string.Empty;
            if (string.IsNullOrEmpty(Num.number))
            {
                return "";
            }

            if (!string.IsNullOrEmpty(Num.SendDigit))
                option_send_digits = string.Format("api_on_answer='uuid_recv_dtmf ${uuid} {0}'", Num.SendDigit);
            else
                option_send_digits = "";
            foreach (Gateway gw in fs.GetGatewaysForNumber(Num.number, ""))
            {
                List<string> num_options = new List<string>();

                if (!string.IsNullOrEmpty(option_send_digits))
                    num_options.Add(option_send_digits);

                num_options.Add(string.Format("absolute_codec_string={0}", gw.GatewayCodec));
                num_options.Add(string.Format("leg_timeout={0}", gw.GatewayTimeout));
                int gw_retries = 1;
                try
                {
                    gw_retries = int.Parse(gw.GatewayRetry);
                    if (gw_retries <= 0)
                        gw_retries = 1;
                }
                catch (Exception e)
                {
                    gw_retries = 1;
                }
                string options = "";
                if (!string.IsNullOrEmpty(num_options.ToString()))
                    options = string.Format("[{0}]", string.Join(",", num_options));
                else
                    options = "";
                string num_str = string.Format("{0}{1}/{2}", options, gw.GatewayString, Num.number);
                string dial_num = "";
                for (int i = 1; i <= gw_retries; i++)
                {
                    dial_num = string.Join("|", num_str);
                }
                num_gw.Add(dial_num);
            }
            string result = string.Join("|", num_gw);
            return result;
        }
        public override void Execute(FSOutbound outboundClient)
        {
            List<string> numbers = new List<string>();
            string sched_hangup_id = string.Empty;
            string dialNumber = string.Empty;

            if (!string.IsNullOrEmpty(base.Text.Trim()))
            {
                dialNumber = base.Text.Trim();
                Number num = new Number();
                num.number = dialNumber;
                numbers.Add(PrepareNumber(num, outboundClient));
            }
            else
            {
                //Set numbers to dial from Number nouns
                foreach (Element child in Children)
                {
                    string dial_num = "";
                    Number num = null;
                    Conference conf = null;
                    if (child.GetType() == typeof(Number))
                    {
                        num = (Number)child;
                        dial_num = PrepareNumber(num, outboundClient);
                        if (string.IsNullOrEmpty(dial_num))
                            continue;
                        numbers.Add(dial_num);
                    }
                    else if (child.GetType() == typeof(Conference))
                    {
                        //if conference is needed
                        conf = (Conference)child;
                        //set record for conference
                        if (record) { conf.record = true; }
                        //set hangupOnStar for member
                        if (hangupOnStar) { conf.hangup_on_star = true; }
                        //Create Partcipant
                        Participant participant = new Participant();
                        participant.CallSid = outboundClient.CallSid;
                        participant.AccountSid = outboundClient.AccountSid;
                        conf.participant = participant;
                        conf.Execute(outboundClient);
                    }
                }
            }
            if (numbers == null)
            {
                return;
            }
            else
            {
                string duration_ms = string.Empty;
                Call bleg = new Call();
                //Set timeout
                outboundClient.set(string.Format("call_timeout={0}", timeout));
                outboundClient.set(string.Format("answer_timeout={0}", timeout));
                //Set callerid or unset if not provided
                if (!string.IsNullOrEmpty(caller_id))
                    outboundClient.set(string.Format("effective_caller_id_number={0}", caller_id));
                else
                    outboundClient.unset("effective_caller_id_number");
                //Set continue on fail
                outboundClient.set("continue_on_fail=true");
                //Set ring flag if dial will ring.
                //But first set agbara_dial_rang to false to be sure we don't get it from an old Dial
                outboundClient.set("agbara_dial_rang=false");
                outboundClient.set("execute_on_ring=set::agbara_dial_rang=true");

                //Create dialstring
                dial_str = string.Join(":_:", numbers);

                // Don't hangup after bridge !
                outboundClient.set("hangup_after_bridge=false");
                if (hangupOnStar)
                    outboundClient.set("bridge_terminate_key=*");
                else
                    outboundClient.unset("bridge_terminate_key");
                outboundClient.set("bridge_early_media=true");
                outboundClient.unset("instant_ringback");
                outboundClient.unset("ringback");

                //set bleg call sid
                outboundClient.set(string.Format("agbara_bleg_callsid={0}", bleg.Sid));
                //enable session heartbeat
                outboundClient.set("enable_heartbeat_events=60");

                // set call direction
                outboundClient.set(string.Format( "agbara_call_direction={0}", CallDirection.outbounddial));
                string dial_rang = "";
                string hangup_cause = "NORMAL_CLEARING";

                //string recordingPath = "";
                //AppSettingsReader reader = new AppSettingsReader();
                //recordingPath = (string)reader.GetValue("RecordingDirectory", recordingPath.GetType());
                //var CallSid = outboundClient.CallSid;
                //string filename = string.Format("{0}_{1}", DateTime.UtcNow.ToShortDateString(), outboundClient.get_channel_unique_id());
                //string record_file = string.Format("{0}{1}.wav", recordingPath, CallSid);
                //if (record)
                //{
                //    outboundClient.set("RECORD_STEREO=true");
                //    outboundClient.APICommand(string.Format("uuid_record {0} start {1}", outboundClient.get_channel_unique_id(), record_file));
                //}


                try
                {
                    //execute bridge
                    outboundClient.bridge(dial_str, outboundClient.get_channel_unique_id(), true);
                    //waiting event
                    var evnt = outboundClient.ActionReturnedEvent();
                    //parse received events
                    if (evnt.GetHeader("Event-Name") == "CHANNEL_UNBRIDGE")
                    {
                        evnt = outboundClient.ActionReturnedEvent();
                    }
                    string reason = "";
                    string originate_disposition = evnt.GetHeader("variable_originate_disposition");

                    long answer_seconds_since_epoch = long.Parse(evnt.GetHeader("Caller-Channel-Answered-Time"));
                    long end_seconds_since_epoch = long.Parse(evnt.GetHeader("Event-Date-Timestamp"));

                    hangup_cause = originate_disposition;
                    if (hangup_cause == "ORIGINATOR_CANCEL")
                        reason = string.Format("{0} (A leg)", hangup_cause);
                    else
                        reason = string.Format("{0} (B leg)", hangup_cause);
                    if (string.IsNullOrEmpty(hangup_cause) || hangup_cause == "SUCCESS")
                    {
                        hangup_cause = outboundClient.GetHangupCause();
                        reason = string.Format("{0} (A leg)", hangup_cause);
                        if (string.IsNullOrEmpty(hangup_cause))
                        {
                            hangup_cause = outboundClient.GetVar("bridge_hangup_cause", outboundClient.get_channel_unique_id());
                            reason = string.Format("{0} (B leg)", hangup_cause);
                            if (string.IsNullOrEmpty(hangup_cause))
                            {
                                hangup_cause = outboundClient.GetVar("hangup_cause", outboundClient.get_channel_unique_id());
                                reason = string.Format("{0} (A leg)", hangup_cause);
                                if (string.IsNullOrEmpty(hangup_cause))
                                {
                                    hangup_cause = "NORMAL_CLEARING";
                                    reason = string.Format("{0} (A leg)", hangup_cause);
                                }
                            }
                        }
                    }

                    //Get ring status
                    dial_rang = outboundClient.GetVar("agbara_dial_rang", outboundClient.get_channel_unique_id());

                    //get duration
                    duration_ms = EpochTimeConverter.GetEpochTimeDifferent(answer_seconds_since_epoch, end_seconds_since_epoch).ToString();
                }

                catch (Exception e)
                {
                }
                finally
                {
                    outboundClient.session_params.Add("DialCallSid", bleg.Sid);
                    outboundClient.session_params.Add("DialCallDuration", duration_ms);
                    if (dial_rang == "true")
                    {

                        outboundClient.session_params.Add("DialCallStatus", CallStatus.completed);
                    }
                    else
                    {
                        outboundClient.session_params.Add("DialCallStatus", CallStatus.failed);
                    }
                }
            }

            //If record is specified
            if (record) { outboundClient.session_params.Add("RecordingUrl", ""); }

            if (!string.IsNullOrEmpty(action) && Util.IsValidUrl(action) && !string.IsNullOrEmpty(method))
            {
                Task.Factory.StartNew(() => FetchNextAgbaraRespone(action, outboundClient.session_params, method));
            }

        }
    }

}


