# Agbara-Go & FreeSWITCH Integration Guide

## FreeSWITCH Server Prerequisite

The Agbara-Go application, particularly for its Call Management features (like originating calls) and advanced Conference controls (planned for future integration), requires a running and properly configured FreeSWITCH instance.

**Setting up a FreeSWITCH server is a system administration task that falls outside the scope of this Agbara-Go application guide.** You are responsible for installing, configuring, and maintaining your FreeSWITCH server. Please refer to the [official FreeSWITCH documentation](https://freeswitch.org/docs/) for installation and detailed configuration instructions.

### Key FreeSWITCH Configurations for Agbara-Go

For Agbara-Go to connect and interact with your FreeSWITCH server, ensure the following are configured:

1.  **Event Socket Layer (ESL) Enabled:**
    *   The `mod_event_socket` module must be loaded and configured in your FreeSWITCH instance.
    *   Edit `event_socket.conf.xml` (usually found in `/usr/local/freeswitch/conf/autoload_configs/` or similar).
    *   Ensure it's listening on an IP address and port that Agbara-Go can reach. For development, `127.0.0.1` (localhost) is common.
        ```xml
        <configuration name="event_socket.conf" description="Socket Client">
          <settings>
            <param name="listen-ip" value="127.0.0.1"/>
            <param name="listen-port" value="8021"/>
            <param name="password" value="YourESLPassword"/> <!-- Set a strong password -->
            <!-- <param name="apply-inbound-acl" value="lan"/> -->
            <!-- <param name="stop-on-bind-error" value="true"/> -->
          </settings>
        </configuration>
        ```
    *   The `listen-ip`, `listen-port`, and `password` will be needed for Agbara-Go's configuration.

2.  **Dialplan Considerations (for Call Origination via `AnswerUrl`):**
    *   When Agbara-Go originates a call using an `AnswerUrl` from a `CallRequest` that is *not* an HTTP URL (i.e., it's a direct FreeSWITCH application string or dialplan target), this URL translates to an instruction for FreeSWITCH on how to handle the call once the called party answers.
    *   For initial testing and basic operation, you might need a dialplan context that can:
        *   Route calls to external gateways or SIP users.
        *   Execute simple applications like `echo()`, `playback(...)`, or `hangup()`.
    *   Example: If `AnswerUrl` is `my_context/public/12345`, you'd need `my_context` defined in your dialplan.
    *   A very simple setup for testing might involve a public context with an extension that just plays an echo:
        ```xml
        <!-- In conf/dialplan/public.xml or similar -->
        <extension name="public_echo_test">
          <condition field="destination_number" expression="^echo_test$">
            <action application="answer"/>
            <action application="echo"/>
          </condition>
        </extension>
        ```
        In this case, `AnswerUrl` in Agbara-Go could be something like `echo_test` (if your originate dialstring targets the public context implicitly or explicitly).

3.  **Outbound Call Configuration (SIP Profiles/Gateways):**
    *   If you intend for Agbara-Go to originate calls to actual phone numbers (PSTN) or SIP URIs, your FreeSWITCH instance must have correctly configured SIP profiles and gateways. This is standard FreeSWITCH setup. See "Gateway Configuration for Outbound Calls" section below for more.

4.  **Firewall:**
    *   Ensure your firewall allows Agbara-Go to connect to the FreeSWITCH ESL port (e.g., 8021).

**Agbara-Go will connect to FreeSWITCH using the configured ESL host, port, and password.** These will be set via environment variables in the Agbara-Go application.

Refer to the main `README.md` for Agbara-Go specific configuration.

## Dynamic Call Control via HTTP (TwiML-like Responses)

For advanced call handling, Agbara-Go can respond with TwiML-like XML instructions to control the call flow in FreeSWITCH. This requires FreeSWITCH to make HTTP requests to specific endpoints in the Agbara-Go application when certain call events occur (e.g., when a call is answered, or after collecting digits).

Agbara-Go's `Application.VoiceUrl` (when it's an HTTP URL) typically defines the endpoint that FreeSWITCH should query. The primary way Agbara-Go expects FreeSWITCH to invoke these TwiML endpoints is via a Lua script.

### 1. Using Lua with `htcache` or `socket.http` (Recommended for Flexibility)

This is a highly flexible method. You can create a Lua script (e.g., `handle_agbara_voiceurl.lua`) that FreeSWITCH executes as part of the dialplan. This script then makes an HTTP request to Agbara-Go's `Application.VoiceUrl`, passing call-specific variables.

**Example Lua Script (`handle_agbara_voiceurl.lua` - place in FreeSWITCH scripts directory):**
```lua
-- handle_agbara_voiceurl.lua
-- Expected arguments from FreeSWITCH dialplan (set by Agbara-Go's originate command):
local app_voice_url = argv[1]   -- The Application.VoiceUrl from Agbara-Go
local call_uuid = argv[2]     -- FreeSWITCH Channel UUID (session.uuid)
local agbara_account_sid = argv[3] -- Agbara Account SID
local agbara_app_sid = argv[4]   -- Agbara Application SID

if not app_voice_url or not call_uuid or not agbara_account_sid then
  freeswitch.consoleLog("ERR", "Lua (handle_agbara_voiceurl): Missing required arguments: app_voice_url, call_uuid, or agbara_account_sid.\n")
  session:hangup("LUASCRIPT_ARG_ERROR")
  return
end

-- Construct query parameters to send to Agbara-Go
local params = "CallSid=" .. call_uuid .. "&AccountSid=" .. agbara_account_sid
if agbara_app_sid then
  params = params .. "&ApplicationSid=" .. agbara_app_sid
end
-- You can add more session variables if your Agbara-Go endpoint needs them:
-- local from_num = session:getVariable("caller_id_number")
-- params = params .. "&From=" .. from_num
-- local to_num = session:getVariable("destination_number")
-- params = params .. "&To=" .. to_num

local full_url = app_voice_url
if string.find(app_voice_url, "?") then
  full_url = app_voice_url .. "&" .. params
else
  full_url = app_voice_url .. "?" .. params
end

freeswitch.consoleLog("INFO", "Lua (handle_agbara_voiceurl): Requesting call control from Agbara-Go: " .. full_url .. "\n")

-- Using htcache for GET. For POST, consider luasocket.http or os.execute("curl ...")
local http_response_body = फ्रीस्विच. एफएसएपीआई("htcache", full_url)

if http_response_body and not string.find(http_response_body, "^-ERR") and string.find(http_response_body, "<Response") then
  freeswitch.consoleLog("INFO", "Lua (handle_agbara_voiceurl): Received TwiML from Agbara-Go. Executing.\n" .. http_response_body .. "\n")
  session:execute("execute_xml_dialplan", http_response_body)
else
  freeswitch.consoleLog("ERR", "Lua (handle_agbara_voiceurl): Failed to fetch or invalid TwiML/XML from Agbara-Go. Response: " .. (http_response_body or "NO_RESPONSE") .. "\n")
  session:hangup("LUASCRIPT_HTTP_FETCH_FAILED")
end
```

**Invocation from Dialplan (via Agbara-Go's `CallService`):**
When Agbara-Go originates a call for an Application that has an HTTP `VoiceUrl` (e.g., `http://<agbara-go-host>:<port>/api/v1/voice/control`), its `CallService` constructs a FreeSWITCH `originate` command that directly uses a Lua script like `handle_agbara_voiceurl.lua`.

The dial string generated by `CallService` might look like:
`{agbara_call_sid=CA...,agbara_account_sid=AC...,agbara_application_sid=AP...,origination_caller_id_name='...',origination_caller_id_number='...'}sofia/gateway/your_gateway/destination_number lua(handle_agbara_voiceurl.lua http://your_agbara_app_voice_url ${uuid} ${agbara_account_sid} ${agbara_application_sid})`

The `handle_agbara_voiceurl.lua` script (as exemplified above, and which should be placed in FreeSWITCH's scripts directory) would then make an HTTP request to the provided `VoiceUrl` (first argument to lua script), appending `CallSid` (second arg), `AccountSid` (third arg), and `ApplicationSid` (fourth arg) as query parameters. Agbara-Go's `/api/v1/voice/control` endpoint (or whatever `VoiceUrl` is) will receive this request and respond with TwiML.

### 2. Using `mod_httapi`

`mod_httapi` allows FreeSWITCH to interact with an HTTP server using a predefined API structure. You define "bindings" that map FreeSWITCH applications/APIs to HTTP endpoints. FreeSWITCH makes requests to these endpoints, and your Agbara-Go app would need to conform to the expected request/response format of `mod_httapi`. This module is powerful but might require more specific request parsing and response generation in Agbara-Go. Refer to the `mod_httapi` documentation for details.

### 3. Using `mod_xml_curl` (Less Common for Dynamic Control)

`mod_xml_curl` is typically used for fetching entire XML configurations (dialplan, directory, etc.) from a web server when FreeSWITCH starts or reloads XML. While it can be used for dynamic routing by having FreeSWITCH fetch dialplan XML on a per-call basis, it's generally less suited for step-by-step TwiML-like interactive call control compared to Lua+HTTP or `mod_httapi`.

**Agbara-Go API Endpoints for FreeSWITCH:**

Your Agbara-Go application will need to expose HTTP endpoints that FreeSWITCH can call. These endpoints (defined by `Application.VoiceUrl`) will receive call parameters from FreeSWITCH and must respond with TwiML-like XML that FreeSWITCH can understand and execute. The `pkg/twiml` package in Agbara-Go will assist in generating this XML.

**Key Variables to Pass from FreeSWITCH to Agbara-Go (via Lua):**
When the Lua script makes an HTTP request to Agbara-Go, ensure it passes at least:
*   `CallSid` (from `session.uuid`): The unique ID of the call leg.
*   `AccountSid`: The Agbara Account SID associated with the call (passed as arg to Lua).
*   `ApplicationSid`: The Agbara Application SID (passed as arg to Lua, if available).
*   Optionally: `From` (Caller ID), `To` (Destination), `Digits` (if any input was gathered).

These allow Agbara-Go to provide contextually relevant TwiML instructions.

### Gateway Configuration for Outbound Calls

Agbara-Go can instruct FreeSWITCH to use specific gateways for outbound calls based on Account Settings (`DefaultOutboundGateway` or `GatewaySelectionScript`).

*   **Define Gateways in FreeSWITCH:** You must configure these gateways within your FreeSWITCH's SIP Profiles (e.g., in `conf/sip_profiles/external/my_gateway.xml`). Each gateway will have its own registration details, codecs, and dialling rules.
    Example gateway definition snippet:
    ```xml
    <gateway name="my_carrier_gateway_1">
      <param name="username" value="user"/>
      <param name="password" value="pass"/>
      <param name="proxy" value="sip.carrier.com"/>
      <param name="register" value="true"/>
      <!-- other params -->
    </gateway>
    ```
*   **Agbara-Go Reference:** When an Agbara-Go Account has `DefaultOutboundGateway` set to `my_carrier_gateway_1`, the `CallService` will attempt to format the FreeSWITCH `originate` dial string like:
    `...{channel_vars}sofia/gateway/my_carrier_gateway_1/destination_number ...`
*   **Lua for Dynamic Gateways:** If using `GatewaySelectionScript` on an Account, that Lua script (which you create and place in FreeSWITCH's scripts directory) will be invoked by `originate`. The `CallService` will format the dial string like:
    `...{channel_vars}lua(your_gateway_script.lua destination_number) ...`
    The Lua script (`your_gateway_script.lua`) receives the destination number as an argument and must return the dial string part for the selected gateway (e.g., `sofia/gateway/selected_gateway/`). It's crucial this script is well-tested and secure.

## Real-time Event Handling

To provide real-time updates for call and conference statuses, Agbara-Go now maintains a persistent Event Socket Layer (ESL) connection to FreeSWITCH dedicated to listening for events.

### Event Subscription

Upon startup, Agbara-Go subscribes to a set of FreeSWITCH events. The default list of subscribed events is:
`CHANNEL_CREATE CHANNEL_ANSWER CHANNEL_HANGUP_COMPLETE CHANNEL_PROGRESS_MEDIA CUSTOM conference::maintenance RECORD_STOP`

This list can be customized via the `FS_EVENT_SUBSCRIPTIONS` environment variable in Agbara-Go.

### Event Processing

Received events are parsed and dispatched internally to relevant services:
*   **CallService**: Processes events like `CHANNEL_CREATE`, `CHANNEL_ANSWER`, `CHANNEL_HANGUP_COMPLETE`, `CHANNEL_PROGRESS_MEDIA` to update the status, timestamps (start/end), duration, and FreeSWITCH call ID of call records in the Agbara-Go database.
*   **ConferenceService**: Processes `CUSTOM conference::maintenance` events (for participant actions like join, leave, mute/unmute, talking states) and `CHANNEL_HANGUP_COMPLETE` (for participants leaving a conference) to update conference and participant records.

This ensures that the call and conference information retrieved via the Agbara-Go API reflects the most current state known from FreeSWITCH.

### Important Considerations for Event Handling:

1.  **Event Format:** The current implementation primarily parses event headers assuming `plain` event format from FreeSWITCH. If your FreeSWITCH ESL is configured to send events in JSON or XML format, the Agbara-Go event parsing logic (`pkg/freeswitch_events/dispatcher.go`) would need to be adapted.
2.  **Channel Variables for Correlation:**
    *   For Call events, Agbara-Go attempts to correlate events to its internal call records using the FreeSWITCH Channel UUID (`Unique-ID` header) matched against the `freeswitch_call_id` field, or by using a channel variable `variable_agbara_call_sid` if present in the event. Ensure your dialplan or originate commands set `agbara_call_sid=${agbara_call_sid}` (where `${agbara_call_sid}` is the Agbara Call SID generated by `CallService`) on channels if you need to rely on Agbara's internal Call SID for event correlation.
    *   For Conference participant events (like hangup), Agbara-Go looks for `variable_conference_name` (or `variable_conference_uuid`) on the channel to identify which conference the participant belonged to. Ensure your FreeSWITCH dialplan (e.g., the part that dials into a conference) sets this variable on participant channels:
        ```xml
        <action application="set" data="conference_name=${agbara_conference_sid}"/> <!-- where ${agbara_conference_sid} is the SID of the Agbara Conference -->
        <action application="conference" data="${agbara_conference_sid}@default"/>
        ```
3.  **Event Reliability:** While the ESL connection includes reconnection logic, ensure your FreeSWITCH event socket configuration is stable.
4.  **`conference::maintenance` Events:** For detailed conference participant updates, ensure that the FreeSWITCH conference profile is configured to fire these `CUSTOM` events (this is usually default behavior). The `conference::maintenance` events are subscribed to as `CUSTOM` type, and then filtered by `Event-Subclass` in the handler.
```
