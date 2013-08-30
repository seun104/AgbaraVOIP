using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using System.Configuration;
using System.IO;
namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Conference : Element
    {
        
        int DEFAULT_TIMELIMIT = 0;
        int DEFAULT_MAXMEMBERS = 10;
        const int MAX_LOOPS = 1000;
        public bool record = false;
        private string full_room = "";
        private string room = "";
        private string moh_sound = "";
        private bool muted = false;
        private bool beep = false;
        private bool StartOnEnter = false;
        private bool EndOnExit = false;
        private int max_members = 10;
        private string action = "";
        private string method = "";
        private string conf_id = "";
        private string member_id = "";
        public Participant participant;
        public bool hangup_on_star = false;
        private string enter_sound = "";
        private string exit_sound = "";

        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            room = Text;
            if (string.IsNullOrEmpty(room))
            {
                //raise exception because room name cannot be empty
                return;
            }
            full_room = room +"@agbara";
            moh_sound = ExtractAttributeValue("waitUrl");
            method = ExtractAttributeValue("waitMethod");
            bool.TryParse(ExtractAttributeValue("muted"), out muted);
            bool.TryParse(ExtractAttributeValue("beep"), out beep);
            bool.TryParse(ExtractAttributeValue("startConferenceOnEnter"), out StartOnEnter);
            bool.TryParse(ExtractAttributeValue("endConferenceOnExit"), out EndOnExit);
            int.TryParse(ExtractAttributeValue("maxMembers", DEFAULT_MAXMEMBERS.ToString()), out max_members);
            if (max_members <= 0 || max_members > DEFAULT_MAXMEMBERS)
                max_members = DEFAULT_MAXMEMBERS;

        }
        private List<string> _prepare_moh(FSOutbound outboundClient)
        {
            List<string> sound_files = new List<string>(); //[]
            if (!Util.IsUrlExist(moh_sound))
            {
                DirectoryInfo file = new DirectoryInfo(moh_sound);
                if (Util.IsFileExist(moh_sound))
                {
                    moh_sound = string.Format("file_string://{0}", moh_sound);
                    sound_files.Add(moh_sound);
                }
            }
            else
            {

                XElement doc = null;
                if (!string.IsNullOrEmpty(moh_sound))
                {
                    try
                    {
                        string response = outboundClient.SendToUrl(moh_sound, outboundClient.session_params, method);
                        doc = XElement.Parse(response);
                    }
                    catch (Exception ex)
                    {
                    }
                    if (doc.Name != "Response")
                        return sound_files;

                    // build play string from remote restxml
                    foreach (XElement element in doc.Elements())
                    {
                        //Play element
                        if (element.Name == "Play")
                        {
                            Play child = new Play();
                            child.ParseElement(element);
                            string sound_file = child.SoundFilePath;
                            if (!string.IsNullOrEmpty(sound_file))
                            {
                                int loop = child.LoopTimes;
                                if (loop == 0)
                                    loop = MAX_LOOPS;  //Add a high number to Play infinitely
                                //Play the file loop number of times
                                for (int i = 0; i < loop; i++)
                                {
                                    sound_files.Add(sound_file);
                                }
                                // Infinite Loop, so ignore other children
                                if (loop == MAX_LOOPS)
                                    break;
                            }
                        }
                        //Say element
                        else if (element.Name == "Say")
                        {
                            Say child = new Say();
                            child.ParseElement(element);
                           // child.Execute(outboundClient);
                            //sound_files.Add(pause_str);
                        }

                         //Redirect element
                        else if (element.Name == "Redirect")
                        {
                            Redirect child = new Redirect();
                            child.ParseElement(element);
                            child.Execute(outboundClient);
                        }
                    }
                }
            }
            return sound_files;
        }
        public override void Execute(FSOutbound outboundClient)
        {
            List<string> flags = new List<string>();
            //settings for conference room
            outboundClient.set("conference_controls=none");
            if (max_members > 0)
                outboundClient.set(string.Format("max-members={0}", max_members));
            else
                outboundClient.unset("max-members");


            //set moh sound
            List<string> mohs = _prepare_moh(outboundClient);
            if (mohs.Count > 0)
            {
                outboundClient.set("playback_delimiter=!");
                string play_str = string.Join("!", mohs);
                play_str = string.Format("file_string://silence_stream://1!{0}", play_str);
                outboundClient.set(string.Format("conference_moh_sound={0}", play_str));
            }
            else
                outboundClient.unset("conference_moh_sound");
            //set member flags
            if (muted)
                flags.Add("muted");
            if (StartOnEnter)
                flags.Add("moderator");
            else
                flags.Add("wait-mod");
            if (EndOnExit)
                flags.Add("endconf");
            string flags_opt = string.Join(",", flags);
            if (!string.IsNullOrEmpty(flags_opt))
                outboundClient.set(string.Format("conference_member_flags={0}", flags_opt));
            else
                outboundClient.unset("conference_member_flags");

            outboundClient.set(string.Format("agabara_confid={0}", 2333333));
            outboundClient.set(string.Format("agabara_memberid={0}", 45555));
            //really enter conference room
            var res = outboundClient.conference(full_room, outboundClient.get_channel_unique_id(), true);
            if (!res.IsReplyTextSuccess())
            {
                //error creating conference
                return;
            }
            //Console.WriteLine(""
            // get next event
            var evnt = outboundClient.ActionReturnedEvent();

            try
            {
                string digit_realm = "";
               
                //set hangup on star
                if (hangup_on_star)
                {
                    // create event template
                    string raw_event = string.Format("Event-Name=CUSTOM,Event-Subclass=conference::maintenance,Action=kick,Unique-ID={0},Member-ID={1},Conference-Name={2},Conference-Unique-ID={3}", outboundClient.get_channel_unique_id(), member_id, room, conf_id);
                    digit_realm = string.Format("agbara_bda_{0}", outboundClient.get_channel_unique_id());
                    string cmd = string.Format("{0},*,exec:event,'{1}'", digit_realm, raw_event);
                    outboundClient.bind_digit_action(cmd);
                }
                

                //play beep on enter if enabled
                if (!string.IsNullOrEmpty(member_id) && beep)
                {
                    if (enter_sound == "beep:1")
                        outboundClient.BgAPICommand(string.Format("conference {0} play tone_stream://%%(300,200,700) async", room));
                    else if (enter_sound == "beep:2")
                        outboundClient.BgAPICommand(string.Format("conference {0} play tone_stream://L=2;%%(300,200,700) async", room));
                }

                // wait conference ending for this member
                var ret = outboundClient.ActionReturnedEvent();

                //play beep on exit if enabled
                if (!string.IsNullOrEmpty(member_id))
                {
                    if (exit_sound == "beep:1")
                        outboundClient.APICommand(string.Format("conference {0} play tone_stream://%%(300,200,700) async", room));
                    else if (exit_sound == "beep:2")
                        outboundClient.APICommand(string.Format("conference {0} play tone_stream://L=2;%%(300,200,700) async", room));
                }
                //unset digit realm
                if (!string.IsNullOrEmpty(digit_realm))
                    outboundClient.clear_digit_action(digit_realm);
            }
            catch (Exception ex)
            {
            }

            finally
            {
                //notify channel has left room
               // _notify_exit_conf(outboundClient);
                //If action is set, redirect to this url, Otherwise, continue to next Element
                //if (!string.IsNullOrEmpty(action) && outboundClient.is_valid_url(action))
                //{
                //    Hashtable param = new Hashtable();
                //    param["ConferenceName"] = room;
                //    param["ConferenceUUID"] = conf_id;
                //    param["ConferenceMemberID"] = member_id;
                //    //if (!string.IsNullOrEmpty(record_file))
                //    //{
                //    //    param["RecordFile"] = record_file;
                //    //    FetchNextAgbaraRespone(action, param, method);
                //    //}
                //}
            }
        }
    }
}
