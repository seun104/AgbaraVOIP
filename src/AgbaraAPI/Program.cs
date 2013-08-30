namespace Emmanuel.AgbaraVOIP.AgbaraAPI
{
    using System;
    using System.Diagnostics;

    using Nancy.Hosting.Self;

    class Program
    {
        static void Main()
        {
            WebServer server = new WebServer();
            try
            {
                server.Start();
                Console.WriteLine("API Server Started running at {0}...");
                Process.Start("http://127.0.0.1:8082");
            }
            catch (Exception ex)
            {
                Console.WriteLine("Api Server Failed to start");
            }
            Console.ReadLine();
            server.Stop();
            Console.WriteLine("Server Stopped. Good bye!");
        }
    }
}