using System;
using Nancy;
using Nancy.Security;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{
    public class DefaultModule : NancyModule
    {

        public DefaultModule()
        {
            Get["/"] = x =>
            {
                return "Welcome to AgbaraVOIP.org";
            };
        }
    }
}
