using System;
using System.Collections.Generic;
using System.Linq;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using DreamSongs.MongoRepository;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class AccountService :IAccountService
    {
        MongoRepository<Account> accountRepo = new MongoRepository<Account>();
        public bool Validate(string SId, string AuthToken)
        {
            return accountRepo.GetSingle(a => a.Sid == SId && a.AuthToken == AuthToken) != null ? true : false;
        }

        public Account CreateAccount(string FriendlyName)
        {
            Account acc = new Account() { FriendlyName = FriendlyName};
            try
            {
                accountRepo.Add(acc);
            }
            catch (Exception ex)
            {
                acc = new Account();
            }

            return acc;
        }

        public Objects.Account CreateAccount(string FriendlyName, string AccountSid)
        {
            Account acc = new Account() {FriendlyName=FriendlyName, ParentSid=AccountSid };
            try
            {
                accountRepo.Add(acc);
            }
            catch (Exception ex)
            {
                acc = new Account();
            }

            return acc;

        }

        public IQueryable<Objects.Account> GetAllAccounts(string AccountSid)
        {
            return accountRepo.All(a => a.ParentSid == AccountSid);
        }

        public Objects.Account GetAccount(string AccountSid)
        {
            return accountRepo.GetSingle(a => a.Sid == AccountSid);
        }

        public Objects.Account ChangeAccountStatus(string AccountSid, string Status)
        {
            
            var acc = GetAccount(AccountSid);
            acc.Status = Status;

            try
            {
                accountRepo.Update(acc);
            }
            catch (Exception ex)
            {
                acc = new Account();
            }
            return acc;
        }

        public Objects.Account ChangeAccountType(string AccountSid, string Type)
        {
            var acc = GetAccount(AccountSid);

            acc.Type = Type; ;

            try
            {
                accountRepo.Update(acc);
            }
            catch (Exception ex)
            {
                acc = new Account();
            }
            return acc;
        }
    }
}
