using System;
using System.Threading;
using System.Threading.Tasks;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{
    public class InboundSocket : EventSocket
    {
        private string host;
        private int port;
        private string password;
        private string _filter;
        private int connect_timeout;
        private bool _eventjsons;
        public InboundSocket(string host, int port, string password, string filter = "ALL", int connect_timeout = 5, bool eventjson = true)
            : base(filter, eventjson)
        {
            this.host = host;
            this.port = port;
            this.password = password;
            this._filter = filter;
            this.connect_timeout = connect_timeout;
            this._eventjsons = eventjson;
            transport = new InboundTransport(host, port, connect_timeout);
         }
        public override void  connect()
    {
     /*
        Connects to mod_eventsocket, authenticates and sets event filter.
        Returns True on success or raises ConnectError exception on failure.
       */
        base.connect();
        Event filter_response;
        //# Connects transport, if connection fails, raise ConnectError
        try
        {
            transport.connect();
        }
        catch (Exception ex)
        {
               throw new ConnectError("Cannot to the server");
        }
        start_event_handler();

        //Send password to the socket
        Event evnt =  auth(password);
       
        if (!evnt.IsReplyTextSuccess() && evnt == null)
        {
            throw new ConnectError("Authentication Failed...");
        }
       
        // Sets event filter or raises ConnectError
        if (!string.IsNullOrEmpty(_filter))
        {
            if (_is_eventjson)
                filter_response = EventJson(_filter);
            else
                filter_response = EventPlain(_filter);
            if (!filter_response.IsReplyTextSuccess())
            {
                disconnect();
                throw new ConnectError("Event filter failure");
            }
        }

        // Sets connected flag to True
        connected = true;
    }
        public void exit()
        {
            base.Exit();
            disconnect();
        }
        public void serve_forever()
        {
            /*
               Starts waiting for events in endless loop.
               */
            while (is_connected())
            {

                Thread.SpinWait(10);
            }
        }
    }
}