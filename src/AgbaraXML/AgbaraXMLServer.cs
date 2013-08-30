using System.Net;
using System.Net.Sockets;
using Emmanuel.AgbaraVOIP.Freeswitch;
namespace Emmanuel.AgbaraVOIP.AgbaraXML
{
    public class AgbaMLServer
    {
        public static void Start(int Port=8085)
        {
            IPEndPoint address = new IPEndPoint(IPAddress.Parse("127.0.0.1"), Port);
            AgbaraXMLServer server = new AgbaraXMLServer(address);
            server.serve_forever();
        }
    }
    internal class AgbaraXMLServer :OutboundServer
    {
        public AgbaraXMLServer(IPEndPoint address)
            : base(address)
        {
            System.Console.WriteLine("AgbaML Server Started ...");
        }
        public override void handle_request(Socket socket)
        {
            FSOutbound client = new FSOutbound(socket, "", "", "", "");
            
        }
    }
}

