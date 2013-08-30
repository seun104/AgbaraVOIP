using System.Linq;
using Nancy;
using Nancy.Security;
using Nancy.ModelBinding;
using Nancy.Responses;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{
    public class ConferenceModule : BaseModule
    {
        private readonly IConferenceService confService;
        private readonly IInboundDependency apiserver;
        public ConferenceModule(IInboundDependency apiServe)
        {
            
            this.RequiresAuthentication();
            this.confService = new ConferenceService();

            #region Get All Conferences
            Get["/Accounts/{AccountSid}/Conferences"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    IQueryable<Conference> res = confService.GetAllConferences(AccountSid);
                    return Response.AsXml<IQueryable<Conference>>(res);
                }
            };
            Get["/Accounts/{AccountSid}/Conferences.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    IQueryable<Conference> res = confService.GetAllConferences(AccountSid);
                    return Response.AsJson<IQueryable<Conference>>(res);
                }
            };
            #endregion

            #region Get Conference
            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    Conference res = confService.GetConference(x.ConferenceSid);
                    return Response.AsXml<Conference>(res);
                }
            };

            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    Conference res = confService.GetConference(x.ConferenceSid);
                    return Response.AsJson<Conference>(res);
                }
            };
            #endregion

            #region Get All Conference Participants
            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    var ConferenceSid = x.ConfenrenceSid;
                    IQueryable<Participant> res = confService.GetAllConferenceParticipants(AccountSid, ConferenceSid);
                    return Response.AsXml<IQueryable<Participant>>(res);
                }
            };
            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var AccountSid = Context.CurrentUser.UserName;
                    var ConferenceSid = x.ConfenrenceSid;
                    IQueryable<Participant> res = confService.GetAllConferenceParticipants(AccountSid, ConferenceSid);
                    return Response.AsJson<IQueryable<Participant>>(res);
                }
            };
            #endregion

            #region Get Participant
            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants/{CallSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    Participant res = confService.GetParticipant(x.ConferenceSid,x.CallSid);
                    return Response.AsXml<Participant>(res);
                }
            };

            Get["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants/{CallSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    Participant res = confService.GetParticipant(x.ConferenceSid, x.CallSid);
                    return Response.AsJson<Participant>(res);
                }
            };
            #endregion

            #region Mute/Unmute Participant
            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants/{CallSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<MuteParticipantRequest>();
                    //Todo
                    //Validation
                    MuteParticipantResponse res = apiserver.GetServer().ConferenceMuteParticipant(request);
                    return Response.AsXml<MuteParticipantResponse>(res);
                }
            };

            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants/{CallSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<MuteParticipantRequest>();
                    //Todo
                    //Validation
                    MuteParticipantResponse res = apiserver.GetServer().ConferenceMuteParticipant(request);
                    return Response.AsJson<MuteParticipantResponse>(res);
                }
            };
            #endregion

            #region Kick Participant
            Delete["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Participants/{CallSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    if (apiserver.GetServer().ConferenceKickParticipant(x.ConferenceSid, x.CallSid))
                    { return HttpStatusCode.NoContent; }
                    else { return HttpStatusCode.NoResponse; }
                }
            };
            #endregion

            #region Record Conference
            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Record"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ConferenceRecordRequest>();
                    //Todo
                    //Validation
                    ConferenceRecordResponse res = apiserver.GetServer().ConferenceRecord(request);
                    return Response.AsXml<ConferenceRecordResponse>(res);
                }
            };

            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Record.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ConferenceRecordRequest>();
                    //Todo
                    //Validation
                    ConferenceRecordResponse res = apiserver.GetServer().ConferenceRecord(request);
                    return Response.AsJson<ConferenceRecordResponse>(res);
                }
            };
            #endregion

            #region Stop Record Conference
            Delete["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Record"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    if (x.AccountSid != Context.CurrentUser.UserName)
                    { return HttpStatusCode.Unauthorized; }
                    else
                    {
                        if (apiserver.GetServer().ConferenceStopRecord(x.ConferenceSid))
                        { return HttpStatusCode.NoContent; }
                        else { return HttpStatusCode.NoResponse; }
                    }
                }
            };
            #endregion

            #region Conference Play
            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Play"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ConferencePlayRequest>();
                    //Todo
                    //Validation
                    ConferencePlayResponse res = apiserver.GetServer().ConferencePlay(request);
                    return Response.AsXml<ConferencePlayResponse>(res);
                }
            };

            Post["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Play.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    var request = this.Bind<ConferencePlayRequest>();
                    //Todo
                    //Validation
                    ConferencePlayResponse res = apiserver.GetServer().ConferencePlay(request);
                    return Response.AsJson<ConferencePlayResponse>(res);
                }
            };
            #endregion

            #region Conference Stop Play
            Delete["/Accounts/{AccountSid}/Conferences/{ConferenceSid}/Play"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                {
                    return HttpStatusCode.Unauthorized;
                }
                else
                {
                    if (x.AccountSid != Context.CurrentUser.UserName)
                    { return HttpStatusCode.Unauthorized; }
                    else
                    {
                        if (apiserver.GetServer().ConferenceStopCall(x.ConferenceSid))
                        { return HttpStatusCode.NoContent; }
                        else { return HttpStatusCode.NoResponse; }
                    }
                }
            };
            #endregion
        }


    }
}

