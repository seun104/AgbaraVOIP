using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CreateAccountRequest
    {
        public string AccountSid { get; set; }
        public string FriendlyName { get; set; }
    }
}
