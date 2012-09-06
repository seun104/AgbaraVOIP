using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Net;
using System.Threading;
using System.Threading.Tasks;
using System.Net.Sockets;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{
    public class OutboundClient : EventSocket
    {
        //FreeSWITCH Outbound Event Socket. A new instance of this class is created for every call/ session from FreeSWITCH.
        private string _uuid;
        private CommandResponse _channel;
        private CommandResponse connect_response;
        public bool _iseventjson;
        public OutboundClient(Socket socket, string filter = "ALL", bool iseventjson = false, int connecttimeout = 60): base(filter,  iseventjson)
        {
            base.transport = new OutboundTransport(socket, connecttimeout);
            _uuid = string.Empty;
            
        }
        public override void connect()
        {
            base.connect();
            //Starts event handler for this client/session.
            start_event_handler();
            try
            {
                connect_response = (CommandResponse)ProtocolSend("connect");
                if (!connect_response.IsSuccess())
                {
                    throw new ConnectError("Error Connecting");
                }
            }
            catch (Exception ex)
            {
                throw new ConnectError("Timeout Connecting");
            }
            // Sets channel and channel unique id from this event
            _channel = connect_response;
            _uuid = connect_response.GetHeader("Unique-ID");

            //Set connected flag to True
            connected = true;

            // Sets event filter or raises ConnectError
            CommandResponse filter_response;
            if (_is_eventjson)
            {
                filter_response = (CommandResponse)EventJson(_filter);
            }
            else
            {

                filter_response = (CommandResponse)EventPlain(_filter);
            }
            if (!filter_response.IsSuccess())
            {
                throw new ConnectError("Event filter failure");
            }
        }
        public CommandResponse get_channel()
        {
            return _channel;
        }
        public string get_channel_unique_id()
        {
            return _uuid;
        }
        public virtual void Run()
        {
            //   This method must be implemented by subclass.
            //   This is the entry point for outbound application.
            while (true)
            {
                start_event_handler();
                Thread.Sleep(10);
            }
        }
    }

    public class OutboundServer 
    {
        internal string _filter;
        private TcpListener listener;
        internal Type _requestClass;
        internal bool isRunning = true;

        public OutboundServer(IPEndPoint address, string filter = "ALL")
            
        {
            _filter = filter;
            listener = new TcpListener(address);
        }
        public void serve_forever()
        {
            start();
            while (isRunning)
            {
                 Socket sckClient = listener.AcceptSocket();
                 Task.Factory.StartNew(() => do_handle(sckClient));
                 Thread.Sleep(10);
            }
        }
        private void do_handle( Socket sock)
        {
            try
            {
                handle_request(sock);
            }
            catch (Exception ex)
            {
                throw new ConnectError(ex.Message);
            }
            finally
            {
               finish_request(sock);
            }
        }
        public virtual void start()
        {
            listener.Start();
        }
        public virtual void handle_request( Socket socket)
        {
            //Must be declared by class implenting this
            object[] param = new object[]{socket,_filter};
            var ty = (OutboundClient)Activator.CreateInstance(_requestClass, param);
        }
        private void finish_request(Socket sock)
        {
            try
            {
                sock.Shutdown(SocketShutdown.Both);
            }
            catch (Exception ex)
            {
            }

            try
            {
                sock.Close();
            }
            catch (Exception ex)
            {

            }
        }
    }
}
