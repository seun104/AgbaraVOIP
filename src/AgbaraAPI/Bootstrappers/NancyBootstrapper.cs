using System;
using Nancy;
using TinyIoC;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    public class NancyBootstrapper : DefaultNancyBootstrapper
    {
        protected override void ConfigureApplicationContainer(TinyIoCContainer container)
        {
            // Register our app dependency as a normal singleton
            
            container.Register<IInboundDependency, InboundDependency>().AsSingleton();
         
            base.ConfigureApplicationContainer(container);
        }
        protected override void ConfigureRequestContainer(TinyIoCContainer container, NancyContext context)
        {
            base.ConfigureRequestContainer(container, context);
            
        }
        protected override void RequestStartup(TinyIoCContainer container, Nancy.Bootstrapper.IPipelines pipelines, NancyContext context)
        {
            base.RequestStartup(container, pipelines, context);
        }

    }
}
