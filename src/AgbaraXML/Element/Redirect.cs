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
    public class Redirect : Element
    {
        
        private string method = "";
        private string url = "";
        public Redirect()
            : base()
        {
            method = "";
            url = "";
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            method = ExtractAttributeValue("method","POST");
            url = base.Text.Trim();
            if (!Util.IsValidUrl(url))
                url = "";
        }
        public override void Execute(FSOutbound outboundClient)
        {
            FetchNextAgbaraRespone(url, outboundClient.session_params, method);
        }
    }
}
