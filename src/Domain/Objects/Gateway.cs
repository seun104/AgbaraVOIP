using System;
namespace Emmanuel.AgbaraVOIP.Domain.Objects
{
    public class Gateway 
    {
        public string Sid { get; set; }
        public  string FSSid { get; set; }
        public  string GatewayString { get; set; }
        public  string GatewayCodec { get; set; }
        public  string GatewayRetry { get; set; }
        public  string GatewayTimeout { get; set; }
        public  string Routes { get; set; }
        public  DateTime DateUpdated { get; set; }
        public  DateTime DateCreated { get; set; }
        public Gateway()
        {
            DateCreated = DateTime.Now;
            DateUpdated = DateTime.Now;
            Sid = "GW" + Guid.NewGuid().ToString("N");
        }
    }
}
