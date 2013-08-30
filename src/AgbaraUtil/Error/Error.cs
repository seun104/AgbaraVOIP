using System;
using System.Diagnostics;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class LimitExceededError : Exception
    {
        public LimitExceededError(string message)
            : base(message)
        {
        }


    }

    public class ConnectError : Exception
    {
        public ConnectError(string message) : base(message) { }

    }
}
