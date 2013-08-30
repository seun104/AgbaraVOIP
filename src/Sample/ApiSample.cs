using System;
using System.IO;
using System.Net;
using System.Web;
using System.Text;
using System.Collections;
using System.Configuration;

namespace Sample.AgbaraAPI
{
    public class ApiSample
    {
        private static string RestSample()
        {

            const string ACCOUNT_SID = "seun104";
            const string ACCOUNT_TOKEN = "agbara";
            string response = string.Empty;
            var account = new AgbaraRESTAPIClient(ACCOUNT_SID, ACCOUNT_TOKEN);
            Hashtable h = new Hashtable();
            h.Add("AuthId", ACCOUNT_SID);
            h.Add("AuthToken", ACCOUNT_TOKEN);
            h.Add("from", "123456789");
            h.Add("to", ConfigurationManager.AppSettings["ToNumber"]);
            h.Add("AnswerUrl", ConfigurationManager.AppSettings["AnswerlUrl"]);
            try
            {
                response = account.request("/test/call", "POST", h);
            }
            catch
            {
                response = "Api Server Error Occured";
            }


            return response;
        }

        public static void Sample1()
        {
            string res = RestSample();
            Console.WriteLine(res);
        }
    }

}
