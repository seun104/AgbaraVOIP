using System;
using log4net;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;

namespace Emmanuel.AgbaraConsole
{
    class Program
    {
        static ILog log = LogManager.GetLogger("Console");
        static void Main(string[] args)
        {
           

                      Console.WriteLine("Starting1");
                    //Check if database is configured already...
                    if (!IsDBOk())
                        SetupDB();
                    try
                    {
                        AgbaraVOIPService.Start();
                        log.Info("AgbaraVOIP Successfully Started...");
                    }
                    catch (Exception ex)
                    {
                        log.Error("AgbaraVOIP Failed To Start...");
                    }


                    Console.ReadLine();
        }
        private static void SetupDB()
        {
            log.Debug("Creating database object...");
            OrmBase db = new OrmBase();
                if (db.SetUPDB())
                    log.Info("Database is created...");
                else
                    log.Info("Database failed to be created...");
            
        }
        private static bool IsDBOk()
        {
            log.Info("Checking if database is installed...");
            return false; ;
        }
    }
}
