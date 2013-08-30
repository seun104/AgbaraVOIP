using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.IO;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using System.Configuration;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Play : Element
    {
        const int MAX_LOOPS = 1000;
        public int LoopTimes;
        public string SoundFilePath = "";
        public string TempSoundFilePath = "";
        public Play()
            : base()
        {
            LoopTimes = 1;
        }

        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            int Loop = 0;
            int.TryParse(ExtractAttributeValue("loop", "1"), out Loop);

            if (Loop == 0 || Loop > MAX_LOOPS)
                LoopTimes = MAX_LOOPS;
            else
                LoopTimes = Loop;
            //Pull out the text within the element
            string audio_path = base.Text.Trim();
            if (string.IsNullOrEmpty(audio_path))
            {
                //return restexception for empty file
            }
            if (!Util.IsUrlExist(audio_path))
            {
                DirectoryInfo file = new DirectoryInfo(audio_path);
                if (Util.IsFileExist(audio_path))
                    SoundFilePath = string.Format("file_string://{0}", audio_path);
            }
            else
            {
                //set to temp path for prepare to process audio caching async
                TempSoundFilePath = audio_path;
            }
        }
        public void Prepare(FSOutbound outboundClient)
        {
            //if not self.sound_file_path:
            //url = normalize_url_space(self.temp_audio_path)
            //self.sound_file_path = get_resource(outbound_socket, url)
        }
        public override void Execute(FSOutbound outboundClient)
        {
            string play_str = "";
            if (!string.IsNullOrEmpty(SoundFilePath))
            {
                if (LoopTimes == 1)
                    play_str = SoundFilePath;
                else
                    outboundClient.set("playback_delimiter=!");
                play_str = "file_string://silence_stream://1!";
                List<string> path = new List<string>();
                for (int i = 1; i <= LoopTimes; i++)
                {
                    path.Add(SoundFilePath);
                }
                play_str += string.Join("!", path);
                var res = outboundClient.playback(play_str);
                if (res.IsReplyTextSuccess())
                {
                    var evnt = outboundClient.ActionReturnedEvent();
                    if (evnt.GetHeaders().Count < 1)
                    {
                        return;
                    }
                }
                else
                {
                }
                return;
            }

        }
    }
}
