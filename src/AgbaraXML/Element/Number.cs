using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Elements
{
    public class Number : Element
    {
        public string number;
        public string SendDigit;
        public string Url;


        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);

            // don't allow "|" and "," in a number noun to avoid call injection
            number = Text.Split(',', '|')[0];
            SendDigit = ExtractAttributeValue("sendDigits");
            Url = ExtractAttributeValue("url");
        }
    }
}
