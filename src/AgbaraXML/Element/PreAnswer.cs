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
    public class PreAnswer : Element
    {
        public PreAnswer()
            : base()
        {
            base.Nestables.Add("Play");
            base.Nestables.Add("Say");
            base.Nestables.Add("Gather");
            base.Nestables.Add("Pause");
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
        }

        public void Prepare()
        {
            //Todo
        }
        public override void Execute(FSOutbound outboundClient)
        {
            outboundClient.preanswer();
            foreach (Element child in Children)
            {
                child.Run(outboundClient);
            }
        }
    }
}
