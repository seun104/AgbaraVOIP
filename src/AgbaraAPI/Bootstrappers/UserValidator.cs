using Nancy.Authentication.Basic;
using Nancy;
using Nancy.Security;
using System.Configuration;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    public class UserValidator : IUserValidator
    {
        public IUserIdentity Validate(string AccountSid, string AuthToken)
        {
            AccountService acc = new AccountService();
            if (acc.Validate(AccountSid, AuthToken))
            {
                return new UserIdentity { UserName = AccountSid };
            }
            else
            {
                return null;
            }
        }
    }
}
