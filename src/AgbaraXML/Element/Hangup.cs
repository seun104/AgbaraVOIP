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
    public class Hangup : Element
    {


        public Hangup()
            : base()
        {

        }

        public override void Execute(FSOutbound outboundClient)
        {
            outboundClient.hangup("NORMAL_CLEARING");
            return; //reason;
        }
    }
}
