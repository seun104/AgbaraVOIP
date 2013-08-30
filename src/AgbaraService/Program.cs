using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Topshelf;
using log4net.Config;
using log4net;
using System.IO;

namespace Emmanuel.AgbaraService
{
    class Program
    {
       static ILog log = LogManager.GetLogger("Setup");
        static void Main(string[] args)
        {
            if (args.Length > 0 && (args[0] == "install" || args[0] == "uninstall"))
            {
               //Check if database is configured already...
                if (!IsDBOk())
                    SetupDB();
            }

            StartHosting();
        }
        private static void StartHosting()
        {
           
            XmlConfigurator.ConfigureAndWatch(new FileInfo(".\\log4net.config"));
            Host h = HostFactory.New(x =>
                {
                    x.Service<AgbaraVOIPService>(s =>
                        {
                            s.SetServiceName("AgbaraVOIPService");
                            s.ConstructUsing(name => new AgbaraVOIPService());
                            s.WhenStarted(tc => tc.Start());
                            s.WhenStopped(tc => tc.Stop());
                        });

                    x.RunAsLocalSystem();

                    x.SetDescription("AgbaraVOIP Window Service");
                    x.SetDisplayName("AgbaraVOIP Window Service");
                    x.SetServiceName("AgbaraVOIPService");
                });


            h.Run();
        }
        

        static void SetupDB()
        {
            log.Debug("Creating database object...");
        }
        static bool IsDBOk()
        {
            log.Info("Checking if database is installed...");
            return true;
        }
    }
}
