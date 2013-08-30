using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Collections;
using System.Web;
using System.Net;
using System.Security.Cryptography;
namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class HTTPRequest
    {
        const string USER_AGENT = "agbara";
        private string id;
        private string token;
        string authdata = "";
        public HTTPRequest(string id, string token)
        {
            this.id = id;
            this.token = token;
        }

        private string GenerateKey(string Key, string StringValue)
        {

            return "";
        }
        private string Download(string uri, SortedList vars)
        {
           // 1. format query string
            if (vars != null)
            {
                string query = "";
                foreach (DictionaryEntry d in vars)
                {
                    authdata += d.Key.ToString() + HttpUtility.UrlEncode(d.Value.ToString()) + "&";
                    query += "&" + d.Key.ToString() + "=" +  HttpUtility.UrlEncode(d.Value.ToString());
                }
                if (query.Length > 0)
                    uri = uri + "?" + query.Substring(1);
            }

            // 2. setup basic authenication
            string authstring = Convert.ToBase64String(
                Encoding.ASCII.GetBytes(String.Format("{0}:{1}",
                id, token)));
            
            //


            // 3. perform GET using WebClient
            WebClient client = new WebClient();
            client.Headers["X_agbara_SIGNATURE"] = authstring;
            client.Headers["Authorization"] = String.Format("Basic {0}", authstring);
            //Uri ur = new Uri(uri);
            byte[] resp = AsyncCtpExtensions.DownloadDataTaskAsync(client, uri).Result;// client.DownloadData(ur);

            return Encoding.ASCII.GetString(resp);
        }
        private string Upload(string uri, string method, SortedList vars)
        {
            SortedList l = new SortedList();
            // 1. format body data

            string data = "";
            
            if (vars != null)
            {
                foreach (DictionaryEntry d in vars)
                {
                    authdata += d.Key.ToString()  + HttpUtility.UrlEncode(d.Value.ToString()) + "&";
                    data += d.Key.ToString() + "=" +  HttpUtility.UrlEncode(d.Value.ToString()) + "&";
                }

            }

            // 2. setup basic authenication
            string authstring = Convert.ToBase64String(Encoding.ASCII.GetBytes(String.Format("{0}:{1}",
                                                       id, token)));

            // 3. perform POST/PUT/DELETE using WebClient
            ServicePointManager.Expect100Continue = false;
            Byte[] postbytes = Encoding.ASCII.GetBytes(data);
            WebClient client = new WebClient();
            client.Headers["X_agbara_SIGNATURE"] = authstring;
            client.Headers.Add("Authorization", String.Format("Basic {0}", authstring));
            client.Headers.Add("Content-Type", "application/x-www-form-urlencoded");

            //byte[] resp = client.UploadData(uri, method, postbytes);
            byte[] resp = AsyncCtpExtensions.UploadDataTaskAsync(client, uri, method, postbytes).Result;
            return Encoding.ASCII.GetString(resp);
        }
        public string FetchResponse(string url, string method, SortedList vars)
        {
            //simple hack if uri is running on localhost
            url = HttpUtility.UrlDecode(url);
            string response = null;
            if (url == null || url.Length <= 0)
                throw (new ArgumentException("Invalid path parameter"));

            method = method.ToUpper();
            if (method == null || (method != "GET" && method != "POST" &&
                                   method != "PUT" && method != "DELETE"))
            {
                throw (new ArgumentException("Invalid method parameter"));
            }

            if (method != "GET" && vars.Count <= 0)
            {
                throw (new ArgumentException("No vars parameters"));
            }

            try
            {
                if (method == "GET")
                {
                    response = Download(url, vars);
                }
                else
                {
                    response = Upload(url, method, vars);
                }
            }
            catch (Exception e)
            {
                throw new ConnectError(e.Message);
            }

            return response;
        }
    }
}
