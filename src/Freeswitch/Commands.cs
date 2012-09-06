using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{


    public abstract class Commands
    {
        public virtual Event ProtocolSend(string commands, string args = "")
        {

            return new Event(string.Empty);
        }
        public virtual Event ProtocolSendMsg(string name, string args, string uuid = "", bool Lock = false, int loop = 1, bool async = false)
        {

            return new Event(string.Empty);
        }
        public Event APICommand(string args)
        {
            //"Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#api"
            return ProtocolSend("api", args);

        }
        public Event BgAPICommand(string args)
        {
            //   "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#bgapi"
            return ProtocolSend("bgapi", args);
        }

        public Event Exit()
        {
            // "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#exit"
            return ProtocolSend("exit", "");
        }

        public Event Resume()
        {
            /* """Socket resume for Outbound connection only.

             If enabled, the dialplan will resume execution with the next action

             after the call to the socket application.

             If there is a bridge active when the disconnect happens, it is killed.*/
            return ProtocolSend("resume", "");
        }
        public Event EventPlain(string args)
        {
            //"Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#event"
            return ProtocolSend("event plain", args);
        }
        public Event EventJson(string args)
        {
            //"Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#event"
            return ProtocolSend("event json", args);
        }
        public Event @Event(string args)
        {
            //"Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#event"
            return ProtocolSend("event", args);
        }
        public Event Execute(string command, string args = "", string uuid = "")
        {
            return ProtocolSendMsg(command, args, uuid, true);
        }
        public virtual string GetVar(string var, string uuid = "")
        {
            /*
            Please refer to http;//wiki.freeswitch.org/wiki/Mod_commands#uuid_getvar
            For Inbound and Outbound connection, uuid argument is mandatory.
            */
            
            string result;

            Event api_response = APICommand(String.Format("uuid_getvar {0} {1}", var, uuid));
            result = api_response.GetBody().Trim();
            if (result == "_unpublic Event_")
                result = "";
            return result;
        }
        public virtual string SetVar(string var, string value, string uuid = "")
        {
            /*
            Please refer to http;//wiki.freeswitch.org/wiki/Mod_commands#uuid_setvar

            For Inbound connection, uuid argument is mandatory.
            """ */
            string result;
            if (string.IsNullOrEmpty(value))
                value = "";
            if (string.IsNullOrEmpty(uuid))
            {
                //uuid = get_channel_unique_id();
            }

            Event api_response = APICommand(String.Format("uuid_setvar {0} {1} {2}", uuid, var, value));
            result = api_response.GetBody().Trim();
            if (string.IsNullOrEmpty(result))
                result = "";
            return result;
        }
        public Event DigitActionSetRealm(string args, string uuid = "", bool islock = true)
        {
            /*Please refer to http://wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_digit_action_set_realm
            >>> digit_action_set_realm("test1")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("digit_action_set_realm", args, uuid, islock);
        }
        public Event clear_digit_action(string args, string uuid = "", bool islock = true)
        {
            /*>>> clear_digit_action("test1")

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("clear_digit_action", args, uuid, islock);
        }
        public Event filter(string args)
        {
            /*   "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#filter

               The user might pass any number of values to filter an event for. But, from the point
               filter() is used, just the filtered events will come to the app - this is where this
               function differs from event().

               >>> filter('Event-Name MYEVENT')
               >>> filter('Unique-ID 4f37c5eb-1937-45c6-b808-6fba2ffadb63')
               """*/
            return ProtocolSend("filter", args);
        }
        public Event filter_delete(string args)
        {
            /*  "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#filter_delete

              >>> filter_delete('Event-Name MYEVENT')
              """ */
            return ProtocolSend("filter delete", args);
        }
        public Event divert_events(string flag)
        {
            /*   "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#divert_events

               >>> divert_events("off")
               >>> divert_events("on")
               """*/
            return ProtocolSend("divert_events", flag);
        }
        public Event sendevent(string args)
        {
            /*   "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#sendevent

               >>> sendevent("CUSTOM\nEvent-Name; CUSTOM\nEvent-Subclass; myevent;;test\n")

               This example will send event ;
                 Event-Subclass; myevent%3A%3Atest
                 Command; sendevent%20CUSTOM
                 Event-Name; CUSTOM
               """ */
            return ProtocolSend("sendevent", args);
        }
        public Event auth(string args)
        {
            /* "Please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#auth

             This method is only used for Inbound connections. */
            return ProtocolSend("auth", args);
        }
        public Event myevents(string uuid = "")
        {
            /*   """For Inbound connection, please refer to http;//wiki.freeswitch.org/wiki/Event_Socket#Special_Case_-_.27myevents.27

               For Outbound connection, please refer to http;//wiki.freeswitch.org/wiki/Event_Socket_Outbound#Events

               >>> myevents()

               For Inbound connection, uuid argument is mandatory.
               """ */
            return ProtocolSend("myevents", uuid);
        }
        public Event linger()
        {
            /*   """Tell Freeswitch to wait for the last channel event before ending the connection

               Can only be used with Outbound connection.

               >>> linger()

               """ */
            return ProtocolSend("linger");
        }
        public Event verbose_events(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_verbose_events

            >>> verbose_events()

            For Inbound connection, uuid argument is mandatory.
            """ */
            return ProtocolSendMsg("verbose_events", "", uuid, islock);
        }
        public Event answer(string uuid = "", bool islock = true)
        {
            /*   Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_answer

               >>> answer()

               For Inbound connection, uuid argument is mandatory.
               """ */
            return ProtocolSendMsg("answer", "", uuid, islock);
        }
        public Event bridge(string args, string uuid = "", bool islock = true)
        {
            /*
            Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_bridge

            >>> bridge("{ignore_early_media=true}sofia/gateway/myGW/177808")

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("bridge", args, uuid, islock);
        }
        public Event hangup(string cause = "", string uuid = "", bool islock = true)
        {

            /* """Hangup call.

             Hangup `cause` list ; http;//wiki.freeswitch.org/wiki/Hangup_Causes (Enumeration column)

             >>> hangup()

             For Inbound connection, uuid argument is mandatory.
             """ */
            return ProtocolSendMsg("hangup", cause, uuid, islock);
        }
        public Event ring_ready(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_ring_ready

            >>> ring_ready()

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("ring_ready", "", uuid);
        }
        public Event record_session(string filename, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_record_session

            >>> record_session("/tmp/dump.gsm")

            For Inbound connection, uuid argument is mandatory.
            """ */
            return ProtocolSendMsg("record_session", filename, uuid, islock);
        }
        public Event bind_meta_app(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_bind_meta_app

            >>> bind_meta_app("2 ab s record_session;;/tmp/dump.gsm")

            For Inbound connection, uuid argument is mandatory.
            """ */
            return ProtocolSendMsg("bind_meta_app", args, uuid, islock);
        }
        public Event bind_digit_action(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_bind_digit_action

             >>> bind_digit_action("test1,456,exec;playback,ivr/ivr-welcome_to_freeswitch.wav")

             For Inbound connection, uuid argument is mandatory.
             """ */
            return ProtocolSendMsg("bind_digit_action", args, uuid, islock);
        }
        public Event wait_for_silence(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_wait_for_silence

            >>> wait_for_silence("200 15 10 5000")

            For Inbound connection, uuid argument is mandatory.
            """ */
            return ProtocolSendMsg("wait_for_silence", args, uuid, islock);
        }
        public Event sleep(string milliseconds, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_sleep

            >>> sleep(5000)
            >>> sleep("5000")

            For Inbound connection, uuid argument is mandatory.
            """*/
            return ProtocolSendMsg("sleep", milliseconds, uuid, islock);
        }
        public Event vmd(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Mod_vmd
            >>> vmd("start")
            >>> vmd("stop")

            For Inbound connection, uuid argument is mandatory.
            """ */
            return ProtocolSendMsg("vmd", args, uuid, islock);
        }
        public Event set(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_set
            >>> set("ringback=${us-ring}")

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("set", args, uuid, islock);
        }
        public Event set_global(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_set_global
            >>> set_global("global_var=value")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("set_global", args, uuid, islock);
        }
        public Event unset(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_unset
            >>> unset("ringback")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("unset", args, uuid, islock);
        }
        public Event start_dtmf(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_start_dtmf
            >>> start_dtmf()
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("start_dtmf", "", uuid, islock);
        }
        public Event stop_dtmf(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_stop_dtmf
            >>> stop_dtmf()
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("stop_dtmf", "", uuid, islock);
        }
        public Event start_dtmf_generate(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_start_dtmf_generate
            >>> start_dtmf_generate()
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("start_dtmf_generate", "true", uuid, islock);
        }
        public Event stop_dtmf_generate(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_stop_dtmf_generate
            >>> stop_dtmf_generate()
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("stop_dtmf_generate", "", uuid, islock);
        }
        public Event queue_dtmf(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_queue_dtmf
            Enqueue each received dtmf, that'll be sent once the call is bridged.
            >>> queue_dtmf("0123456789")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("queue_dtmf", args, uuid, islock);
        }
        public Event flush_dtmf(string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_flush_dtmf
              >>> flush_dtmf()
              For Inbound connection, uuid argument is mandatory.
              */
            return ProtocolSendMsg("flush_dtmf", "", uuid, islock);
        }
        public Event play_fsv(string filename, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Mod_fsv
            >>> play_fsv("/tmp/video.fsv")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("play_fsv", filename, uuid, islock);
        }
        public Event record_fsv(string filename, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Mod_fsv
            >>> record_fsv("/tmp/video.fsv")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("record_fsv", filename, uuid, islock);
        }
        public Event playback(string filename, string terminators = "", string uuid = "", bool islock = true, int loops = 1)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Mod_playback
            The optional argument `terminators` may contain a string withthe characters that will terminate the playback.
            >>> playback("/tmp/dump.gsm", terminators="#8")
            In this case, the audio playback is automatically terminated
            by pressing either '#' or '8'.
            For Inbound connection, uuid argument is mandatory.
            */
            if (string.IsNullOrEmpty(terminators))
            {
                terminators = "none";
            }
            set(string.Format("playback_terminators={0}", terminators), uuid);
            return ProtocolSendMsg("playback", filename, uuid, islock, loops);
        }
        public Event transfer(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_transfer
            >>> transfer("3222 XML public Eventault")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("transfer", args, uuid, islock);
        }
        public Event att_xfer(string url, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_att_xfer
            >>> att_xfer("user/1001")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("att_xfer", url, uuid, islock);
        }
        public Event endless_playback(string filename, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_endless_playback
            >>> endless_playback("/tmp/dump.gsm")
            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("endless_playback", filename, uuid, islock);
        }
        

        public Event record(string filename, string time_limit_secs = "", string silence_thresh = "", string silence_hits = "", string terminators = "", string uuid = "", bool islock = true, int loops = 1)
        {
            /*   Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_record

               */
            if (!string.IsNullOrEmpty(terminators))
            {
                set(string.Format("playback_terminators={0}", terminators),uuid);
            }
            string args = string.Format("{0} {1} {2} {3}", filename, time_limit_secs, silence_thresh, silence_hits);
          return  ProtocolSendMsg("record", args, uuid, islock = true, loops);
        }
        public void play_and_get_digits(int min_digits = 1, int max_digits = 1, int max_tries = 1, int timeout = 5000,
                               string terminators = "#", List<string> sound_files =null, string invalid_file = "", string var_name = "pagd_input",
                                string valid_digits = "0123456789*#", string digit_timeout = "", bool play_beep = true, string uuid="")
        {
            /*   Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_play_and_get_digits
               */
            string play_str;
            string beep = "tone_stream://%(300,200,700)";
            if (sound_files == null)
                if (play_beep)
                    play_str = "tone_stream://%(300,200,700)";
                else
                    play_str = "silence_stream://10";
            else
            {
                set("playback_delimiter=!");
                play_str = "file_string://silence_stream://1";
                foreach (string sound_file in sound_files)
                {
                    play_str += string.Format("{0}!{1}", play_str, sound_file);
                }
                if (play_beep)
                    play_str += string.Format("{0}!{1}", play_str, beep);
            }
            if (string.IsNullOrEmpty(invalid_file))
                invalid_file = "silence_stream://150";
            if (string.IsNullOrEmpty(digit_timeout))
                digit_timeout = timeout.ToString();
            List<string> reg = new List<string>();
            foreach (char d in valid_digits.ToCharArray())
            {
                string i;
                if (d == '*')
                {
                    i = @"\*";
                    reg.Add(i);
                }
                else
                {
                    reg.Add(d.ToString());
                }
            }
            string regexp;
            regexp = string.Join("|", reg);
            regexp = string.Format("({0}+", regexp);


            string args = string.Format("{0} {1} {2} {3} '{4}' {5} {6} {7} {8} {9}", min_digits, max_digits, max_tries, timeout, terminators, play_str, invalid_file, var_name, @"\d+", digit_timeout);
            Execute("play_and_get_digits", args,uuid);
        }
        public void preanswer()
        {
            /*  Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_pre_answer

              Can only be used for outbound connection
              */
            Execute("pre_answer");
        }
        public Event conference(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Mod_conference

              >>> conference(args)

              For Inbound connection, uuid argument is mandatory.
              */
            return ProtocolSendMsg("conference", args, uuid, islock);
        }
        public Event speak(string text, string uuid = "", bool islock = true, int loop=1)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/TTS

             >>> "set" data="tts_engine=flite"
             >>> "set" data="tts_voice=kal"
             >>> speak(text)

             For Inbound connection, uuid argument is mandatory.
             */
            return ProtocolSendMsg("speak", text, uuid, islock,loop);
        }
        public Event hupall(string args)
        {
            //"Please refer to http;//wiki.freeswitch.org/wiki/Mod_commands#hupall"
            return ProtocolSendMsg("hupall", args, "", true);
        }
        public Event say(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_say

            >>> say(en number pronounced 12345)

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("say", args, uuid, islock);
        }
        public Event ScheduleHangup(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_sched_hangup

            >>> sched_hangup("+60 ALLOTTED_TIMEOUT")

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("sched_hangup", args, uuid, islock);
        }
        public Event sched_transfer(string args, string uuid = "", bool islock = true)
        {
            /*"Please refer to http;//wiki.freeswitch.org/wiki/Misc._Dialplan_Tools_sched_transfer

            >>> sched_transfer("+60 9999 XML public Eventault")

            For Inbound connection, uuid argument is mandatory.
            */
            return ProtocolSendMsg("sched_transfer", args, uuid, islock);
        }
    }
}
