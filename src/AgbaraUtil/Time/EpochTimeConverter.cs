using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.AgbaraCommon
{
    public class EpochTimeConverter
    {
        public static DateTime ConvertFromEpochTime(long timestamp)
        {
            DateTime origin = new DateTime(1970, 1, 1, 0, 0, 0, 0);
            return origin.AddSeconds(timestamp / 1000000);
        }
        public static double ConvertToEpochTime(DateTime datetime)
        {
            DateTime epoch = new DateTime(1970, 1, 1, 0, 0, 0, 0);
            return (datetime - epoch).TotalSeconds;
        }
        public static int GetEpochTimeDifferent(long timestampFrom, long timestampTo)
        {
            return EpochTimeConverter.ConvertFromEpochTime((timestampFrom - timestampTo)).Second ;
        }
        
    }
}
