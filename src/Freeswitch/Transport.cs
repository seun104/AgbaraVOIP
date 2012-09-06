//Copyright (c) 2011 agbara Team. See LICENSE for details.
//Freeswitch Transport classes
using System;
using System.Net;
using System.Net.Sockets;
using System.Text;
using System.IO;
using System.Threading;
using System.Diagnostics;
using Emmanuel.AgbaraVOIP.AgbaraCommon;

namespace Emmanuel.AgbaraVOIP.Freeswitch
{
    public interface Transport
    {
        void write(string data);
        string read(int offset);
        string read_line();
        void close();
        void connect();
        int get_connect_timeout();
    }
     public  class InboundTransport :Transport
     {
         private string host;
         private int port;
         private int connect_timeout;
         private Socket sockfd;
         private NetworkStream _stream;
         private char[] _buffer;
         private StreamReader strreader;
         private StreamWriter strwriter;
         public InboundTransport(string host, int port, int connect_timeout)
         {
             this.host = host;
             this.port = port;
             this.connect_timeout = connect_timeout;
             sockfd =null;
         }
         public void connect()
        {
             sockfd = new Socket(AddressFamily.InterNetwork, SocketType.Stream, ProtocolType.Tcp);
            sockfd.Connect(this.host, this.port);
            _stream = new NetworkStream(sockfd);
            strwriter = new StreamWriter(_stream);
            strreader = new StreamReader(_stream);
        }
         public void write(string data)
         {
             if (!sockfd.Connected)
                 throw new ConnectError("not connected");
             else
             {
                     strwriter.WriteLine(data);
                     strwriter.Flush();
             }
             
         }
         public string read(int offset)
         {
             _buffer = new char[offset];
             if (sockfd.Connected)
             {
                 try
                 {
                     strreader.Read(_buffer, 0, offset);
                 }
                 catch (IOException ex)
                 {
                     close();
                 }
             }
             string data = Encoding.UTF8.GetString(Encoding.UTF8.GetBytes(_buffer));
             return data;

         }
         public string read_line()
         {
             string line = string.Empty;
             if (sockfd.Connected)
             {
                 try
                 {                     
                    line= strreader.ReadLine();
                 }
                 catch (IOException ex)
                 {
                     close();
                 }
             }
             return line;
         }
         public void close()
         {
             
                 try
                 {
                     sockfd.Shutdown(SocketShutdown.Receive);
                     sockfd.Close();
                 }
                 catch (Exception ex)
                 {

                 }
             
         }
         public int get_connect_timeout()
         {
             return this.connect_timeout;
         }

      }
     public class OutboundTransport : Transport
     {
         private int connect_timeout;
         private Socket sockfd;
         private NetworkStream _stream;
         private  char[] _buffer;
         private StreamReader strreader;
         private StreamWriter strwriter;
         public OutboundTransport(Socket sock, int connect_timeout)
         {
             sockfd = sock;
             this.connect_timeout = connect_timeout;
             connect();
         }

         public void connect()
         {
             _stream = new NetworkStream(sockfd);
             strwriter = new StreamWriter(_stream);
             strreader = new StreamReader(_stream);
         }
         public void write(string data)
         {
             if (!sockfd.Connected)
                 throw new ConnectError("not connected");
             else
             {
                 strwriter.WriteLine(data);
                 strwriter.Flush();
             }

         }
         public string read(int offset)
         {
             _buffer = new char[offset];
             if (sockfd.Connected)
             {
                 try
                 {
                     strreader.Read(_buffer, 0, offset);
                 }
                 catch (IOException ex)
                 {
                     close();
                 }
             }
             string data = Encoding.UTF8.GetString(Encoding.UTF8.GetBytes(_buffer));
             return data;

         }
         public string read_line()
         {
             string line = string.Empty ; 
             if (sockfd.Connected)
             {
                 try
                 {
                   line = strreader.ReadLine();
                 }
                 catch (IOException ex)
                 {
                     close();
                 }
             }
             return line;
         }
         public void close()
         {
             try
             {
                 sockfd.Shutdown(SocketShutdown.Receive);
                 sockfd.Close();
             }
             catch
             {

             }
         }
         public int get_connect_timeout()
         {
             return this.connect_timeout;
         }

     }
    
}
