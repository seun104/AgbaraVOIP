using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface IRecordingService
    {
        Recording CreateRecording(Recording recording);
        IQueryable<Recording> GetAllRecordings(string AccountSid);
        IQueryable<Recording> GetCallAllRecordings(string CallSid);
        Recording GetRecordingMP3(string RecordingSid);
        Recording GetRecordingWAV(string RecordingSid);
        Recording GetRecording(string RecordingSid);
    }
}


