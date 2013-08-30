using System;
using System.Linq;
using System.Collections.Generic;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using DreamSongs.MongoRepository;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class FSService :IFSService
    {
        private MongoRepository<FSServer> fsRepo = new MongoRepository<FSServer>();

        public FSServer GetFSServer()
        {
            //TODO
            //Read it from config files

            return new FSServer { Host= "127.0.0.1", OutAddress="127.0.0.1:8085", Password="ClueCon", Port=8021, SId="FS1001"};
        }
        public IQueryable<Gateway> GetGatewaysForNumber(string Number, string FSSid)
        {
            //TODO
            //Read it from config files

            List<Gateway> gws = new List<Gateway>();
            gws.Add(new Gateway() { DateCreated = DateTime.UtcNow, DateUpdated = DateTime.UtcNow, FSSid = "FS1001", GatewayCodec = "PCMA,PCMU", GatewayRetry = "2", GatewayString = "sofia/gateway/localphone", GatewayTimeout = "30", Sid = Guid.NewGuid().ToString("N") });
            gws.Add(new Gateway() { DateCreated = DateTime.UtcNow, DateUpdated = DateTime.UtcNow, FSSid = "FS1001", GatewayCodec = "PCMA,PCMU", GatewayRetry = "2", GatewayString = "sofia/gateway/agbaravoip", GatewayTimeout = "30", Sid = Guid.NewGuid().ToString("N") });
            return gws.Where(g=>g.FSSid == FSSid).AsQueryable();
        }
    }
}
