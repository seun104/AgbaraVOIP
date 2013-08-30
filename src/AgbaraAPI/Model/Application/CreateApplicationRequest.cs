using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CreateApplicationRequest
    {
        public string ApplicationSid { get; set; }
        public string FriendlyName { get; set; }
    }

    public class CreateApplicationResponse
    {
        public string ApplicationSid { get; set; }
        public string FriendlyName { get; set; }
    }
}
