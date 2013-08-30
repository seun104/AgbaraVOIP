using Emmanuel.AgbaraVOIP.AgbaraAPI.Core;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper
{
    public interface IInboundDependency
    {
        ApiServer GetServer();
    }
}
