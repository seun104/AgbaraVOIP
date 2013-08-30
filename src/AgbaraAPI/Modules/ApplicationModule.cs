using System.Linq;
using Nancy;
using Nancy.Security;
using Nancy.ModelBinding;
using Nancy.Responses;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{
    public class ApplicationModule : BaseModule
    {
        private readonly IApplicationService appService;
        public ApplicationModule()
        {
            this.appService = new ApplicationService();
            this.RequiresAuthentication();

            #region Get All Application
            Get["/Accounts/{AccountSid}/Applications"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    IQueryable<Application> res = appService.GetAllApplications(AccountSid);
                    return Response.AsXml<IQueryable<Application>>(res);
                }
            };
            Get["/Accounts/{AccountSid}/Applications.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    IQueryable<Application> res = appService.GetAllApplications(AccountSid);
                    return Response.AsJson<IQueryable<Application>>(res);
                }
            };
            #endregion

            #region Create New Application

            Post["/Accounts/{AccountSid}/Applications"] = x => 
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<CreateApplicationRequest>();
                    Application res = appService.CreateApplication(request.FriendlyName, request.ApplicationSid);
                    return Response.AsXml<Application>(res);
                }
            };
            Post["/Accounts/{AccountSid}/Applications.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<CreateApplicationRequest>();
                    Application res = appService.CreateApplication(request.FriendlyName, request.ApplicationSid);
                    return Response.AsJson<Application>(res);
                }
            };

            #endregion

            #region Retrive An Application
            Get["/Accounts/{AccountSid}/Applications/{ApplicationSid}"] = x =>

            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var ApplicationSid = x.ApplicationSid;
                    var AccountSid = Context.CurrentUser.UserName;
                    Application res = appService.GetApplication(ApplicationSid);
                    return Response.AsXml<Application>(res);
                }
            };

            Get["/Accounts/{AccountSid}/Applications/{ApplicationSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    var ApplicationSid = x.ApplicationSid;
                    Application res = appService.GetApplication(ApplicationSid);
                    return Response.AsJson<Application>(res);
                }
            };
            #endregion

            #region  Modify Application

            Post["/Accounts/{AccountSid}/Applications/{ApplicationSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ModifyApplicationRequest>();
                    var AccountSid = Context.CurrentUser.UserName;
                    var ApplicationSid = x.ApplicationSid;
                    Application res = appService.ModifyApplication(ApplicationSid, request.Status);
                    return Response.AsXml<Application>(res);
                }
            };

            Post["/Accounts/{AccountSid}/Applications/{ApplicationSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ModifyApplicationRequest>();
                    var AccountSid = Context.CurrentUser.UserName;
                    var ApplicationSid = x.ApplicationSid;
                    Application res = appService.ModifyApplication(ApplicationSid, request.Status);
                    return Response.AsXml<Application>(res);
                }
            };
            #endregion
        }


    }
}

