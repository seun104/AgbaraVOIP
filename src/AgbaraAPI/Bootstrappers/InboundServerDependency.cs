using System;
using System.Net;
using TinyIoC;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Core;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    /// <summary>
    /// A module dependency that will have an application lifetime scope.
    /// </summary>
    public class InboundDependency : IInboundDependency
    {
        private readonly ApiServer Api;
        /// <summary>
        /// Initializes a new instance of the RequestDependencyClass class.
        /// </summary>
        public InboundDependency()
        {
            this.Api = new ApiServer();
            try
            {
                this.Api.Start();
            }
            catch (Exception ex)
            {

            }
        }
        public ApiServer GetServer()
        {
            return this.Api;
        }
    }
}
