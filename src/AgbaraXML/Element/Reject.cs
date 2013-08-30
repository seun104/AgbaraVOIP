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
    public class Reject : Element
    {

        private string reason = "";
        private int schedule;
        public Reject()
            : base()
        {
            reason = "";
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            if (!int.TryParse(ExtractAttributeValue("schedule", "0"), out schedule))
                schedule = 0;
            reason = ExtractAttributeValue("reason");
            if (reason == "rejected")
                reason = "CALL_REJECTED";
            else if (reason == "busy")
                reason = "USER_BUSY";
            else
                reason = "";
        }
        public override void Execute(FSOutbound outboundClient)
        {

            if (!string.IsNullOrEmpty(reason))
                reason = "NORMAL_CLEARING";

            outboundClient.hangup(reason);
            return; //reason;
        }
    }
}
