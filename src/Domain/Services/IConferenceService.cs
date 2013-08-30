using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using Emmanuel.AgbaraVOIP.Domain.Objects;

namespace Emmanuel.AgbaraVOIP.Domain.Services
{
    public interface IConferenceService
    {
        Conference CreateConference(string FriendlyName, string AccountSid);
        Participant CreateConferenceParticipant(string CallSid, string ConferenceSid, string AccountSid, bool Muted, bool StartOnEnter, bool EndOnExit);
        IQueryable<Conference> GetAllConferences(string AccountSid);
        IQueryable<Participant> GetAllConferenceParticipants(string AccountSid, string ConferenceSid);
        Participant GetParticipant(string ConferenceSid, string CallSid);
        Conference GetConference(string ConferenceSid);
        Participant MuteParticipant(string ConferenceSid,string CallSid, bool Status);
    }
}

