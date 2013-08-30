using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class FSServer 
    {
        public string Sid { get; set; }
        public  string Host { get; set; }
        public  int Port { get; set; }
        public  string Password { get; set; }
        public  string OutAddress { get; set; }
        public FSServer()
        {
            Sid = "FS" + Guid.NewGuid().ToString("N");
        }
    }
}
