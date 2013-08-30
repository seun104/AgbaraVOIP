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
    public class AccountModule : BaseModule
    {
        private readonly IAccountService actService;
        public AccountModule()
        {
            this.actService = new AccountService();
            this.RequiresAuthentication();
            #region Get All SubAccounts
            Get["/Accounts"] = x =>
            {
                var AccountSid = Context.CurrentUser.UserName;
                IQueryable<Account> res = actService.GetAllAccounts(AccountSid);
                return Response.AsXml<IQueryable<Account>>(res);
            };
            Get["/Accounts.json"] = x =>
            {
                var AccountSid = Context.CurrentUser.UserName;
                IQueryable<Account> res = actService.GetAllAccounts(AccountSid);
                return Response.AsJson<IQueryable<Account>>(res);
            };
            #endregion
            #region Create New SubAccount
            
            Post["/Accounts"] = x => 
            {
                var request = this.Bind<CreateAccountRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                 Account res = actService.CreateAccount(request.FriendlyName, request.AccountSid);
                return Response.AsXml<Account >(res);
            };
            Post["/Accounts.json"] = x =>
            {
                var request = this.Bind<CreateAccountRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Account res = actService.CreateAccount(request.FriendlyName, request.AccountSid);
                return Response.AsJson<Account>(res);
            };

            #endregion
            #region Create New Master Account

            Post["/Accounts/Master"] = x =>
            {
                var request = this.Bind<CreateAccountRequest>();
                Account res = actService.CreateAccount(request.FriendlyName);
                return Response.AsXml<Account>(res);
            };
            Post["/Accounts/Master.json"] = x =>
            {
                var request = this.Bind<CreateAccountRequest>();
                Account res = actService.CreateAccount(request.FriendlyName);
                return Response.AsJson<Account>(res);
            };

            #endregion
            #region Retrive An Account
            Get["/Accounts/{AccountSid}"] = x =>

            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    Account res = actService.GetAccount(AccountSid);
                    return Response.AsXml<Account>(res);
                }
            };

            Get["/Accounts/{AccountSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    Account res = actService.GetAccount(AccountSid);
                    return Response.AsJson<Account>(res);
                }
            };
            #endregion
            #region  Modify Account

            Post["/Accounts/{AccountSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ChangeAccountStatusRequest>();
                    var AccountSid = Context.CurrentUser.UserName;
                    Account res = actService.ChangeAccountStatus(AccountSid, request.Status);
                    return Response.AsXml<Account>(res);
                }
            };
            Post["/Accounts/{AccountSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ChangeAccountStatusRequest>();
                    var AccountSid = Context.CurrentUser.UserName;
                    Account res = actService.ChangeAccountStatus(AccountSid, request.Status);
                    return Response.AsXml<Account>(res);
                }
            };
            #endregion
        }


    }
}
