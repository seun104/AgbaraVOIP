using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraAPI;
using Emmanuel.AgbaraVOIP.AgbaraXML;
using System;
using log4net;

namespace Emmanuel.AgbaraService
{
    public interface IService
    {
        void Start();
        void Stop();
    }
    public class AgbaraVOIPService : IService
    {
        ILog log = LogManager.GetLogger(typeof(AgbaraVOIPService));
        public  void Start()
        {
            //Connect
            try
            {
                log.Info("Starting Agbara Rest API..");
                Task.Factory.StartNew(() => AgbaAPIWcfHostingServer.Start());
                log.Info("Starting AgbaraML Processor..");
                Task.Factory.StartNew(() => AgbaMLServer.Start());
            }
            catch (Exception ex)
            {
                log.Fatal("The services existed with the following error...{0}",ex.InnerException);
            }
        }
        public  void Stop()
        {
        }
    }
}
