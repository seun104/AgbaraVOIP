using System;
using System.IO;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraXML.Utils
{
    internal class LogWriter
    {
        public static string CurrentLogFile
        {
            get
            {
                return AppDomain.CurrentDomain.BaseDirectory + "logfile_" + ((int)DateTime.Now.DayOfWeek).ToString() + ".log";
            }
        }

        public static void WriteLog(string scope, string content)
        {
            try
            {
                StreamWriter streamWriter = new StreamWriter(LogWriter.CurrentLogFile, true, Encoding.UTF8);
                streamWriter.WriteLine(DateTime.Now.ToString() + "\r\n");
                streamWriter.WriteLine(scope + "\r\n");
                streamWriter.WriteLine(content + "\r\n\r\n");
                streamWriter.Close();
            }
            catch (Exception ex)
            {
            }
        }
    }
}
