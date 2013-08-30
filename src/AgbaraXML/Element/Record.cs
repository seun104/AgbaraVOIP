using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using System.Configuration;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using Emmanuel.AgbaraVOIP.Freeswitch;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Record : Element
    {

        private int silence_threshold = 500;
        private int max_length = 0;
        private int timeout = 0;
        private string finish_on_key = "";
        private string play_beep = "";
        private string both_legs = "true";
        private string action = "";
        private string method = "";
        private string apiVersion;
        private string transcribe = "";
        private string transcribeCallback = "";
        private IRecordingService recoSvc;
        public Record()
            : base()
        {
            recoSvc = new RecordingService();
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            if (!int.TryParse(ExtractAttributeValue("maxLength"), out max_length))
                max_length = 30;
            if (!int.TryParse(ExtractAttributeValue("timeout", "15"), out timeout))
                timeout = 15;
            finish_on_key = ExtractAttributeValue("finishOnKey", "#");
            play_beep = ExtractAttributeValue("playBeep", "false");
            action = ExtractAttributeValue("action");
            method = ExtractAttributeValue("method", "POST");
            transcribe = ExtractAttributeValue("transcribe");
            transcribeCallback = ExtractAttributeValue("transcribeCallback");
            //TODO Validate finishonkey, maxLenght, timeout
        }

        public override void Execute(FSOutbound outboundClient)
        {

            string recordingPath = "";
            AppSettingsReader reader = new AppSettingsReader();
            recordingPath = (string)reader.GetValue("RecordingDirectory", recordingPath.GetType());
            Event evnt = null;

            if (play_beep == "true")
            {
                string beep = "tone_stream://%(300,200,700)";
                outboundClient.playback(beep);
                var retEvnt = outboundClient.ActionReturnedEvent();
            }
            string record_file = string.Format("{0}{1}.wav", recordingPath, outboundClient.CallSid);
            outboundClient.start_dtmf();
            outboundClient.record(record_file, max_length.ToString(), silence_threshold.ToString(), timeout.ToString(), finish_on_key);
            evnt = outboundClient.ActionReturnedEvent();
            outboundClient.stop_dtmf();

            //Otherwise, continue to next Element
            if (!string.IsNullOrEmpty(action) && Util.IsValidUrl(action))
            {
                outboundClient.session_params["RecordUrl"] = record_file;

                int record_ms;
                try
                {
                    var duration = evnt.GetHeader("variable_record_ms");
                    if (string.IsNullOrEmpty(duration))
                        record_ms = 0;
                    else
                        record_ms = int.Parse(duration);
                }
                catch (Exception ex)
                {
                    record_ms = 0;
                }
                outboundClient.session_params["RecordingDuration"] = record_ms;
                string record_digits = evnt.GetHeader("variable_playback_terminator_used");
                if (!string.IsNullOrEmpty(record_digits))
                    outboundClient.session_params["Digits"] = record_digits;
                else
                    outboundClient.session_params["Digits"] = "hangup";

                //save recording
                Recording record = new Recording();
                record.AccountSid = outboundClient.AccountSid;
                record.CallSid = outboundClient.CallSid;
                record.Duration = record_ms;
                record.RecordUrl = record_file;
                try
                {
                    recoSvc.CreateRecording(record);

                }
                catch (Exception ex)
                {
                    //log
                }

                // fetch xml
                FetchNextAgbaraRespone(action, outboundClient.session_params, method);

            }
        }
    }
}
