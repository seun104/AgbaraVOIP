using System;
using System.Collections.Generic;
using Nancy;
using Nancy.Security;
using Nancy.ModelBinding;
using Nancy.Responses;
using Nancy.Validation;
using TinyIoC;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{
    public class TestModule : NancyModule
    {
        private readonly IInboundDependency apiserver;
        public TestModule(IInboundDependency apiServer)
            : base("/test")
        {
            this.apiserver = apiServer;
            this.RequiresAuthentication();

            Post["/Call"] = x =>
            {
                
                    var request = this.Bind<CallRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    Call res = apiserver.GetServer().Call(request);
                    return Response.AsXml<Call>(res);
            };
            Post["/Call.json"] = x =>
            {

                var request = this.Bind<CallRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsJson<Call>(res);
            };

            Get["/Call"] = x =>
            {

                var request = this.Bind<CallRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsXml<Call>(res);
            };
        }
    }
}
