using System.Collections.Generic;
using Nancy;
using Nancy.Security;
using Nancy.ModelBinding;
using Nancy.Responses;
using Nancy.Validation;
using TinyIoC;
using Emmanuel.AgbaraVOIP.Domain.Objects;
using Emmanuel.AgbaraVOIP.Domain.Services;
using Emmanuel.AgbaraVOIP.Domain.Services.SQL;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Bootstrapper;
using Emmanuel.AgbaraVOIP.AgbaraAPI.Model;
namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Module
{

    public class CallModule : BaseModule
    {
        private readonly IInboundDependency apiserver;
        private readonly ICallService callService;

        public CallModule(IInboundDependency apiServer) 
        {

            this.apiserver = apiServer;
            this.callService = new CallService();
            this.RequiresAuthentication();

            #region Get All Calls
            Get["/Accounts/{AccountSid}/Calls"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                return Response.AsXml<IEnumerable<Call>>(callService.GetAllCalls(Context.CurrentUser.UserName));
            };
            #endregion

            #region Make Call
            Post["/Accounts/{AccountSid}/Calls/Call.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                var request = this.Bind<CallRequest>();
                //Todo
                //Validation
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsJson<Call>(res);
            };
            Post["/Accounts/{AccountSid}/Calls/Call"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                var request = this.Bind<CallRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsXml<Call>(res);
            };
            #endregion

            #region Modify Call
            Post["/Accounts/{AccountSid}/Calls/{CallSid}.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }

                var request = this.Bind<CallRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsJson<Call>(res);
            };
            Post["/Accounts/{AccountSid}/Calls/{CallSid}"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                var request = this.Bind<CallRequest>();
                request.AccountSid = Context.CurrentUser.UserName;
                Call res = apiserver.GetServer().Call(request);
                return Response.AsXml<Call>(res);
            };
            #endregion

            #region Play to Call
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Play.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallPlayRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    request.CallSid = x.CallSid;
                    CallPlayResponse res = apiserver.GetServer().CallPlay(request);
                    return Response.AsJson<CallPlayResponse>(res);
                }
            };
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Play"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallPlayRequest>();
                    request.CallSid = x.CallSid;
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallPlayResponse res = apiserver.GetServer().CallPlay(request);
                    return Response.AsXml<CallPlayResponse>(res);
                }
            };
            #endregion

            #region Stop Call Play
            Delete["/Accounts/{AccountSid}/Calls/{CallSid}/Play"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    if (apiserver.GetServer().StopCallPlay(x.CallSid))
                    { return HttpStatusCode.NoContent; }
                    else { return HttpStatusCode.NoResponse; }
                    //else to do
                }
            };
            #endregion

            #region Record Call
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Record.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallRecordRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallRecordResponse res = apiserver.GetServer().CallRecord(request);
                    return Response.AsJson<CallRecordResponse>(res);
                }
            };
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Record"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallRecordRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallRecordResponse res = apiserver.GetServer().CallRecord(request);
                    return Response.AsXml<CallRecordResponse>(res);
                }
            };
            #endregion

            #region Stop Call Record
            Delete["/Accounts/{AccountSid}/Calls/{CallSid}/Record"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    if (apiserver.GetServer().StopCallRecord(x.CallSid))
                    { return HttpStatusCode.NoContent; }
                    else { return HttpStatusCode.NoResponse; }
                }
            };
            #endregion

            #region Speak
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Speak.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallSpeakRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallSpeakResponse res = apiserver.GetServer().CallSpeak(request);
                    return Response.AsJson<CallSpeakResponse>(res);
                }
            };
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Speak"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallSpeakRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallSpeakResponse res = apiserver.GetServer().CallSpeak(request);
                    return Response.AsXml<CallSpeakResponse>(res);
                }
            };
            #endregion

            #region Digits
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Digit.json"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallDigitRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallDigitResponse res = apiserver.GetServer().CallDigit(request);
                    return Response.AsJson<CallDigitResponse>(res);
                }
            };
            Post["/Accounts/{AccountSid}/Calls/{CallSid}/Digit"] = x =>
            {
                if (x.AccountSid != Context.CurrentUser.UserName)
                { return HttpStatusCode.Unauthorized; }
                else
                {
                    var request = this.Bind<CallDigitRequest>();
                    request.AccountSid = Context.CurrentUser.UserName;
                    CallDigitResponse res = apiserver.GetServer().CallDigit(request);
                    return Response.AsXml<CallDigitResponse>(res);
                }
            };
            #endregion


        }


    }
}
