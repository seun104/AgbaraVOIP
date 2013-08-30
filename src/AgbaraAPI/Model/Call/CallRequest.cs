using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.ComponentModel.DataAnnotations;

namespace Emmanuel.AgbaraVOIP.AgbaraAPI.Model
{
    public class CallRequest
    {
        [Required(ErrorMessage="AccountSid cannot be empty")]
        public string AccountSid { get; set; }
        public string From { get; set; }
        [Required(ErrorMessage = "To Field cannot be empty"), DataType(DataType.PhoneNumber)]
        public string To { get; set; }
        public string ApplicationSid { get; set; }
        [Required(ErrorMessage = "Answer Url cannot be empty"), DataType(DataType.Url,ErrorMessage="Url Not Properly Formatted")]
        public string AnswerUrl { get; set; }
        public string Method { get; set; }
        [DataType(DataType.Url,ErrorMessage="Url Not Properly Formatted")]
        public string FallbackUrl { get; set; }
        public string FallbackMethod { get; set; }
        [DataType(DataType.Url, ErrorMessage = "Url Not Properly Formatted")]
        public string StatusCallbackUrl { get; set; }
        public string StatusCallbackMethod { get; set; }
        [DataType(DataType.Url, ErrorMessage = "Url Not Properly Formatted")]
        public string SendDigits { get; set; }
        public string TimeLimit { get; set; }
        public string HangupOnRing { get; set; }
    }

    public class CallResponse
    {
        public string Message { get; set; }
        public bool IsSuccessful { get; set; }
        public string CallSid { get; set; }
    }
}
