using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;

namespace Emmanuel.AgbaraVOIP.Domain.Services.Mongo
{
    public class ConferenceService: IConferenceService
    {
        public Objects.Conference CreateConference(string FriendlyName, string AccountSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Participant CreateConferenceParticipant(string CallSid, string ConferenceSid, string AccountSid, bool Muted, bool StartOnEnter, bool EndOnExit)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.Conference> GetAllConferences(string AccountSid)
        {
            throw new NotImplementedException();
        }

        public IQueryable<Objects.Participant> GetAllConferenceParticipants(string AccountSid, string ConferenceSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Participant GetParticipant(string ConferenceSid, string CallSid)
        {
            throw new NotImplementedException();
        }

        public Objects.Conference GetConference(string ConferenceSid)
        {
            throw new NotImplementedException();
        }



        public Objects.Participant MuteParticipant(string ConferenceSid, string CallSid, bool Status)
        {
            throw new NotImplementedException();
        }
    }
}
