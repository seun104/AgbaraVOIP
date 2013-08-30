using System;
using System.Collections.Generic;
using System.Collections;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class RedirectException : Exception
    {
        private string _url;
        private string _method;
        private SortedList _param;
        public RedirectException(SortedList Param, string Url = "", string Method = "POST")
        {
            _url = Url;
            _method = Method;
            _param = Param;
        }

        public string GetUrl()
        {
            return _url;
        }

        public string GetMethod()
        {
            return _method;
        }
        public SortedList GetParam()
        {
            return _param;
        }
    }
}
