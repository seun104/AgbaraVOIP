using System;
using System.ServiceModel;
using System.ServiceModel.Web;
using Nancy.Hosting.Self;
using Nancy.Hosting.Wcf;
using log4net;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI
{
    public class AgbaAPISelfHostingServer
    {
        private static SelfHostingWebServer server = new SelfHostingWebServer();
        public static void Start(string restAddress = "http://127.0.0.1:8082/")
        {
            server._serverAddress = restAddress;
            server.Start();
        }
        public static void Stop()
        {
            server.Stop();
        }
    }
    public class AgbaAPIWcfHostingServer
    {
        static ILog log = LogManager.GetLogger(typeof(AgbaAPIWcfHostingServer));
           private static WebServiceHost _nancyHost;

        public static string BaseUri { get; private set; }
        public AgbaAPIWcfHostingServer(string baseUri ="http://127.0.0.1:8082/")
        {
            BaseUri = baseUri;
        }

        public static void Start()
        {
            var service = new NancyWcfGenericService();
            _nancyHost = new WebServiceHost(service, new Uri(BaseUri));
            _nancyHost.AddServiceEndpoint(typeof(NancyWcfGenericService), new WebHttpBinding(), "");

            // this can fail if there isn't a urlacl for the desired port
            // if it fails, open a cmd with "run as administrator" and run:
            // netsh http add urlacl url=http://+:8082/ user=\Everyone
            _nancyHost.Open();
            log.Info(string.Format("Agbara API WCF Hosting Listening to request on {0}", BaseUri));
        }

        public static void Stop()
        {
            log.Info("Agbara API WCF Hosting Stopping");
            _nancyHost.Close();
        }
    }
    public class SelfHostingWebServer
    {
        static ILog log = LogManager.GetLogger(typeof(AgbaAPIWcfHostingServer));
        private  NancyHost nancyHost;
        public  string _serverAddress;
      
        public  void Start()
        {
            nancyHost = new NancyHost(new Uri(_serverAddress));
            try
            {
                nancyHost.Start();
                log.Info(string.Format("Agbara API WCF Hosting Listening to request on {0}", _serverAddress));
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
            }
        }
        public  void Stop()
        {
            nancyHost.Stop();
        }
    }
}
