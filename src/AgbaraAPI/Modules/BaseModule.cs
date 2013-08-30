using System;
using Nancy;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{
    public class BaseModule : NancyModule
    {
        public BaseModule(): base("/v1")
        {
        }

       

    }
}
