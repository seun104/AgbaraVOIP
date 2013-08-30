using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.AgbaraAPI;
using Emmanuel.AgbaraVOIP.AgbaraXML;
using System;
using log4net;

namespace Emmanuel.AgbaraConsole
{
   
    public class AgbaraVOIPService 
    {
       static ILog log = LogManager.GetLogger(typeof(AgbaraVOIPService));
        public static void Start()
        {
            //Connect
            try
            {
                log.Info("Starting Agbara Rest API..");
                 AgbaAPISelfHostingServer.Start();
                log.Info("Starting AgbaraML Processor..");
                AgbaMLServer.Start();
            }
            catch (Exception ex)
            {
                log.Fatal("The services existed with the following error...{0}",ex.InnerException);
            }
        }
        public static void Stop()
        {
            AgbaAPIWcfHostingServer.Stop();
        }
    }
}
