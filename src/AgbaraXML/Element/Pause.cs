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
    public class Pause : Element
    {
        
        public int length = 0;
        private string transfer = "";
        public Pause()
            : base()
        {
            length = 1;
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            if (!int.TryParse(ExtractAttributeValue("length"), out length))
                length = 1;
        }

        public override void Execute(FSOutbound outboundClient)
        {
            
            outboundClient.sleep((length * 1000).ToString(), outboundClient.get_channel_unique_id(), false);
            var evnt = outboundClient.ActionReturnedEvent();
        }
    }
}
