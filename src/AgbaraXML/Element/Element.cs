using System;
using System.IO;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using System.Text;
using System.Collections;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraXML.Utils;
using System.Configuration;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.AgbaraXML
{
    public abstract class Element
    {
        public List<Element> Children = new List<Element>();
        public List<string> Nestables = new List<string>();
        public string Text = string.Empty;
        public Dictionary<string, string> Attributes = new Dictionary<string, string>();
        public string Name;

        public Element()
        {
            Name = this.GetType().Name;
        }

        public virtual void ParseElement(XElement element, string uri = "")
        {
            PrepareAttributes(element);
            PrepareText(element);
        }

        public void Run(FSOutbound client)
        {

            try
            {
                client.CurrentElement = this.GetType().Name;
                Execute(client);
                client.CurrentElement = "";
            }
            catch (Exception ex)
            {
            }

        }
        public virtual void Execute(FSOutbound OutboundSocket)
        {
        }
        public virtual string ExtractAttributeValue(string item, string initial = "")
        {
            try
            {
                item = Attributes[item];
            }
            catch (Exception ex)
            {
                item = initial;
            }

            return item;
        }

        public virtual void PrepareAttributes(XElement element)
        {
            
            foreach (XAttribute attribute in element.Attributes())
            {
                Attributes.Add(attribute.Name.ToString(), attribute.Value);
            }
        }

        public virtual void PrepareText(XElement element)
        {

            string text = element.Value;
            if (!element.HasElements)
            {
                if (string.IsNullOrEmpty(text))
                {
                    Text = "";
                }
                else
                {
                    Text = text.Trim();
                }
            }
        }

        public void FetchNextAgbaraRespone(string url, SortedList param, string method = "POST")
        {
            throw new RedirectException(param, url, method);
        }
    }
}