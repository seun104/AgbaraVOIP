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
    public class Gather : Element
    {

        const int DEFAULT_MAX_DIGITS = 99;
        const int DEFAULT_TIMEOUT = 5;
        private int num_digits = 0;
        private int timeout = 0;
        private int retries = 1;
        private string finish_on_key = "";
        private string action = "";
        private string method = "";
        private bool play_beep = false;
        private string valid_digits = "0123456789*#";

        public Gather()
            : base()
        {
            base.Nestables.Add("Play");
            base.Nestables.Add("Say");
            base.Nestables.Add("Pause");
        }
        public override void ParseElement(XElement element, string uri = "")
        {
            base.ParseElement(element, uri);
            int num = 0;
            if (!int.TryParse(ExtractAttributeValue("numDigits", DEFAULT_MAX_DIGITS.ToString()), out num))
            {
                num_digits = DEFAULT_MAX_DIGITS;
            }

            if (num > DEFAULT_MAX_DIGITS)
                num_digits = DEFAULT_MAX_DIGITS;
            else
                num_digits = num;
            int time = 0;
            if (!int.TryParse(ExtractAttributeValue("timeout", DEFAULT_MAX_DIGITS.ToString()), out time))
            {
                timeout = DEFAULT_TIMEOUT * 1000;
            }
            else
                timeout = time;

            finish_on_key = ExtractAttributeValue("finishOnKey", "#");
            action = ExtractAttributeValue("action");
            method = ExtractAttributeValue("method", "POST");

            if (string.IsNullOrEmpty(action) && !Util.IsValidUrl(action))
                action = uri;
            timeout = timeout * 1000;

        }

        public void Prepare()
        {
            foreach (Element child in Children)
            {

                //if hasattr(child, "prepare"):
                //    TODO Prepare Element concurrently
                //    child.Prepare()
            }
        }
        public override void Execute(FSOutbound outboundClient)
        {
            foreach (Element child in Children)
                if (child.GetType() == typeof(Play))
                {
                    var play = (Play)child;
                    play.Execute(outboundClient);
                }
                else if (child.GetType() == typeof(Pause))
                {
                    var pause = (Pause)child;
                    pause.Execute(outboundClient);
                }
                else if (child.GetType() == typeof(Say))
                {
                    var say = (Say)child;
                    say.Execute(outboundClient);
                }
            outboundClient.play_and_get_digits(1, num_digits, retries, timeout, finish_on_key, null, "", "pagd_input", valid_digits, "", play_beep);

            var evnt = outboundClient.ActionReturnedEvent();
            string digits = evnt.GetHeader("variable_pagd_input");
            if (!string.IsNullOrEmpty(digits) && !string.IsNullOrEmpty(action))
            {
                outboundClient.session_params["Digits"] = digits;
                FetchNextAgbaraRespone(action, outboundClient.session_params, method);
            }
        }
    }
}
