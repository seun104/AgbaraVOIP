using System;
using System.Collections.Generic;
using Nancy.Security;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    public class UserIdentity : IUserIdentity
    {
        public string UserName { get; set; }

        public IEnumerable<string> Claims { get; set; }
    }
}
