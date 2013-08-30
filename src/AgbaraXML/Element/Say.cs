using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using System.Configuration;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Say : Element
    {

        const int MAX_LOOPS = 1000;
        public int loop_times = 1;
        string sound_file_path = "";
        public string engine = "";
        public string voice = "";
        public string language = "";
        public Say()
            : base()
        {

        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            int loop = 0;
            if (!int.TryParse(ExtractAttributeValue("loop", "1"), out loop))
                loop = 1;

            if (loop == 0 || loop > MAX_LOOPS)
                loop_times = MAX_LOOPS;
            else
                loop_times = loop;
            engine = ExtractAttributeValue("engine");
            voice = ExtractAttributeValue("voice");
            language = ExtractAttributeValue("language", "en");
        }
        public override void Execute(FSOutbound client)
        {
            if (voice == "male")
            {
                voice = "kal";
            }
            else
            {
                voice = "slt";
            }
            string say_args = string.Format("{0}|{1}|{2}", engine, voice, base.Text);
            var res = client.speak(say_args, "", true, loop_times);
            for (int i = 1; i <= loop_times; i++)
            {
                var evnt = client.ActionReturnedEvent();
            }
            return;

        }
    }
}
