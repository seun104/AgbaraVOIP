using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface IAccountService
    {
        bool Validate(string SId, string AuthToken);
        Account CreateAccount(string FriendlyName);
        Account CreateAccount(string FriendlyName, string AccountSid);
        IQueryable<Account> GetAllAccounts(string AccountSid);
        Account GetAccount(string AccountSid);
        Account ChangeAccountStatus(string AccountSid, string Status );
        Account ChangeAccountType(string AccountSid, string Type);
    }
}
