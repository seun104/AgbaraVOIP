using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class RecordingService: IRecordingService
    {
        public Objects.Recording CreateRecording(Objects.Recording Record)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.Recording> GetAllRecordings(string AccountSid)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.Recording> GetCallAllRecordings(string CallSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Recording GetRecordingMP3(string RecordingSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Recording GetRecordingWAV(string RecordingSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Recording GetRecording(string RecordingSid)
        {
            throw new NotImplementedException();
        }
    }
}
