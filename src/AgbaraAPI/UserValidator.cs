using System;
using System.Collections.Generic;
using System.Linq;
using System.Web;
using Nancy.Authentication.Basic;
using Nancy.Security;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI
{
	public class UserValidator : IUserValidator
	{
		public IUserIdentity Validate(string user, string password)
		{
            return new DemoUserIdentity { UserName = user };
		}
	}
}