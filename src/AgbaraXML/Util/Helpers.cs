using System;
using System.IO;
using System.Net;
using System.Web;
using System.Text;
using System.Collections;
using System.Collections.Generic;
using System.Threading.Tasks;
using Emmanuel.AgbaraVOIP.Freeswitch;
using Emmanuel.AgbaraVOIP.AgbaraCommon;
namespace Emmanuel.AgbaraVOIP.AgbaraXML.Utils
{
    //public class Gateway
    //{
    //    public string request_uuid { get; set; }
    //    public string to { get; set; }
    //    public string gw { get; set; }
    //    public string codecs { get; set; }
    //    public string timeout { get; set; }
    //    public string extra_dial_string { get; set; }

    //    public Gateway(string request_uuid, string to, string gw, string codec, string timeout, string extra_dial_string)
    //    {
    //        this.request_uuid = request_uuid;
    //        this.to = to;
    //        this.gw = gw;
    //        this.codecs = codec;
    //        this.timeout = timeout;
    //        this.extra_dial_string = extra_dial_string;
    //    }

    //}
    //public class HTTPRequest
    //{
    //    const string USER_AGENT = "agbara";
    //    private string id;
    //    private string token;
    //    public HTTPRequest(string id, string token)
    //    {
    //        this.id = id;
    //        this.token = token;
    //    }
    //    private string Download(string uri, Hashtable vars)
    //    {
    //        // 1. format query string
    //        if (vars != null)
    //        {
    //            string query = "";
    //            foreach (DictionaryEntry d in vars)
    //                query += "&" + d.Key.ToString() + "=" +
    //                    HttpUtility.UrlEncode(d.Value.ToString());
    //            if (query.Length > 0)
    //                uri = uri + "?" + query.Substring(1);
    //        }

    //        // 2. setup basic authenication
    //        string authstring = Convert.ToBase64String(
    //            Encoding.ASCII.GetBytes(String.Format("{0}:{1}",
    //            id, token)));

    //        // 3. perform GET using WebClient
    //        WebClient client = new WebClient();
    //        client.Headers.Add(string.Format("X_agbara_SIGNATURE, {0}", authstring));
    //        client.Headers.Add("Authorization", String.Format("Basic {0}", authstring));
    //        Uri ur = new Uri(uri);

    //        byte[] resp = AsyncCtpExtensions.DownloadDataTaskAsync(client, ur).Result;// client.DownloadData(ur);

    //        return Encoding.ASCII.GetString(resp);
    //    }
    //    private string Upload(string uri, string method, Hashtable vars)
    //    {
    //        // 1. format body data
    //        string data = "";
    //        if (vars != null)
    //        {
    //            foreach (DictionaryEntry d in vars)
    //            {
    //                data += d.Key.ToString() + "=" +
    //                    HttpUtility.UrlEncode(d.Value.ToString()) + "&";
    //            }

    //        }

    //        // 2. setup basic authenication
    //        string authstring = Convert.ToBase64String(Encoding.ASCII.GetBytes(String.Format("{0}:{1}",
    //                                                   id, token)));

    //        // 3. perform POST/PUT/DELETE using WebClient
    //        ServicePointManager.Expect100Continue = false;
    //        Byte[] postbytes = Encoding.ASCII.GetBytes(data);
    //        WebClient client = new WebClient();
    //        client.Headers.Add(string.Format("X_agbara_SIGNATURE, {0}", authstring));
    //        client.Headers.Add("Authorization", String.Format("Basic {0}", authstring));
    //        client.Headers.Add("Content-Type", "application/x-www-form-urlencoded");

    //        //byte[] resp = client.UploadData(uri, method, postbytes);
    //        byte[] resp = AsyncCtpExtensions.UploadDataTaskAsync(client, uri, method, postbytes).Result;
    //        return Encoding.ASCII.GetString(resp);
    //    }
    //    public string FetchResponse(string url, string method, Hashtable vars)
    //    {
    //        string response = null;
    //        if (url == null || url.Length <= 0)
    //            throw (new ArgumentException("Invalid path parameter"));

    //        method = method.ToUpper();
    //        if (method == null || (method != "GET" && method != "POST" &&
    //                               method != "PUT" && method != "DELETE"))
    //        {
    //            throw (new ArgumentException("Invalid method parameter"));
    //        }

    //        if (method != "GET" && vars.Count <= 0)
    //        {
    //            throw (new ArgumentException("No vars parameters"));
    //        }

    //        try
    //        {
    //            if (method == "GET")
    //            {
    //                response = Download(url, vars);
    //            }
    //            else
    //            {
    //                response = Upload(url, method, vars);
    //            }
    //        }
    //        catch (Exception e)
    //        {
    //            throw new ConnectError(e.Message);
    //        }

    //        return response;
    //    }
    //}
    public class Util
    {
        public static bool IsValidUrl(string url)
        {
            return true;
        }
        public static bool IsUrlExist(string url)
        {
            if (string.IsNullOrEmpty(url))
            {
                return false;
            }
            return ((url.Substring(0, 7).ToLower() == "http://") || (url.Substring(0, 7).ToLower() == "https://"));
        }
        public static bool IsFileExist(string url)
        {
            return true;
        }
        public static string NormaliseUrl(string Url)
        {
            return Url.Trim().Replace(" ", "+");
        }
    }
}
