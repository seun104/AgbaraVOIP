using System;
using System.Collections.Generic;
using System.Linq;
using System.Web;
using Nancy.Authentication.Basic;
using Nancy;
using Nancy.Bootstrapper;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    public class AuthenticationBootstrapper : DefaultNancyBootstrapper
    {
        protected override void ApplicationStartup(TinyIoC.TinyIoCContainer container, IPipelines pipeline)
        {
            container.Register<IAccountService, AccountService>().AsSingleton();   
            base.ApplicationStartup(container, pipeline);
            pipeline.EnableBasicAuthentication(new BasicAuthenticationConfiguration(container.Resolve<IUserValidator>(),"AgbaraVOIP"));

        }

        

    }
}
