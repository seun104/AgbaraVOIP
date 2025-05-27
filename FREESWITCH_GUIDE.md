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
    *   When Agbara-Go originates a call using an `AnswerUrl` from a `CallRequest`, this URL often translates to an instruction for FreeSWITCH on how to handle the call once the called party answers.
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
    *   If you intend for Agbara-Go to originate calls to actual phone numbers (PSTN) or SIP URIs, your FreeSWITCH instance must have correctly configured SIP profiles and gateways. This is standard FreeSWITCH setup.

4.  **Firewall:**
    *   Ensure your firewall allows Agbara-Go to connect to the FreeSWITCH ESL port (e.g., 8021).

**Agbara-Go will connect to FreeSWITCH using the configured ESL host, port, and password.** These will be set via environment variables in the Agbara-Go application.

Refer to the main `README.md` for Agbara-Go specific configuration.

## Dynamic Call Control via HTTP (TwiML-like Responses)

For advanced call handling, Agbara-Go can respond with TwiML-like XML instructions to control the call flow in FreeSWITCH. This requires FreeSWITCH to make HTTP requests to specific endpoints in the Agbara-Go application when certain call events occur (e.g., when a call is answered, or after collecting digits).

Agbara-Go's `Application.VoiceUrl` will typically define the endpoint that FreeSWITCH should query.

There are several ways to configure FreeSWITCH to make these HTTP requests:

### 1. Using Lua with `htcache` or `socket.http` (Recommended for Flexibility)

This is a highly flexible method. You can create a Lua script that FreeSWITCH executes as part of the dialplan. This script then makes an HTTP request to Agbara-Go's `Application.VoiceUrl`, potentially passing call-specific variables.

**Example Lua Script (`agbara_http_call_control.lua` - place in FreeSWITCH scripts directory):**
```lua
-- agbara_http_call_control.lua
local url = argv[1]      -- URL passed from dialplan (Application.VoiceUrl)
local call_uuid = argv[2]  -- Channel UUID
local account_sid = argv[3] -- Account SID
-- Add any other relevant session variables you want to pass

if not url or not call_uuid then
  freeswitch.consoleLog("ERR", "Lua: Missing URL or CallUUID for HTTP call control.\n")
  session:hangup("LUASCRIPT_ERROR")
  return
end

-- Construct query parameters
local params = "CallSid=" .. call_uuid .. "&AccountSid=" .. account_sid
-- Add more params: From, To, Digits (if any), etc.
-- local from_num = session:getVariable("caller_id_number")
-- params = params .. "&From=" .. from_num

local full_url = url
if string.find(url, "?") then
  full_url = url .. "&" .. params
else
  full_url = url .. "?" .. params
end

freeswitch.consoleLog("INFO", "Lua: Requesting call control from Agbara-Go: " .. full_url .. "\n")

-- Use FreeSWITCH's htcache for HTTP GET (POST might need luasocket or external curl)
-- For POST or more complex requests, luasocket.http is an option if available.
-- Or, use os.execute("curl ...") if curl is installed and security allows.
local http_request = फ्रीस्विच. एफएसएपीआई("htcache", full_url) -- htcache is simpler for GET

if http_request and string.sub(http_request, 1, 3) ~= "-OK" then
  -- htcache returns "-OK <body/filepath>" on success, or error string
  -- If it returns the body directly and it's XML:
  -- freeswitch.consoleLog("INFO", "Lua: Received XML from Agbara-Go:\n" .. http_request .. "\n")
  -- session:execute("execute_xml_dialplan", http_request) -- If XML is directly executable dialplan
  
  -- A common pattern is htcache downloads to a file, then you parse/execute from file.
  -- For simplicity, let's assume Agbara-Go returns XML that can be directly executed or
  -- that your dialplan logic after this script handles the response.
  -- If Agbara-Go returns TwiML, you'd need a TwiML-to-FreeSWITCH dialplan converter or specific apps.
  
  -- For this example, we'll assume the response is directly executable XML dialplan snippet.
  -- This part is simplified. Real TwiML processing is more involved.
  if string.find(http_request, "<document") then -- Basic check for XML
    session:execute("execute_xml_dialplan", http_request)
  else
    freeswitch.consoleLog("ERR", "Lua: Failed to fetch or invalid XML from Agbara-Go: " .. http_request .. "\n")
    session:hangup("LUASCRIPT_HTTP_FETCH_FAILED")
  end
else
  freeswitch.consoleLog("ERR", "Lua: htcache command failed or no response for " .. full_url .. "\n")
  session:hangup("LUASCRIPT_HTCACHE_ERROR")
end
```

**Example Dialplan (e.g., in `conf/dialplan/default.xml` or a custom context):**
This dialplan extension would be targeted by the `originate` command from Agbara-Go.
```xml
<extension name="agbara_call_handler">
  <condition field="destination_number" expression="^(your_agbara_app_trigger_number)$">
    <action application="answer"/>
    <action application="set" data="agbara_account_sid=${account_sid}"/> <!-- Get from channel var if set by originate -->
    <action application="set" data="agbara_application_voice_url=${application_voice_url}"/> <!-- Set by originate -->
    <action application="lua" data="agbara_http_call_control.lua ${agbara_application_voice_url} ${uuid} ${agbara_account_sid}"/>
    <action application="hangup"/> <!-- Default hangup if Lua script doesn't take over fully -->
  </condition>
</extension>
```
When Agbara-Go originates a call, it would set channel variables like `agbara_application_voice_url` and `agbara_account_sid`, and the dial string would target this extension.

### 2. Using `mod_httapi`

`mod_httapi` allows FreeSWITCH to interact with an HTTP server using a predefined API structure. You define "bindings" that map FreeSWITCH applications/APIs to HTTP endpoints. FreeSWITCH makes requests to these endpoints, and your Agbara-Go app would need to conform to the expected request/response format of `mod_httapi`. This module is powerful but might require more specific request parsing and response generation in Agbara-Go. Refer to the `mod_httapi` documentation for details.

### 3. Using `mod_xml_curl` (Less Common for Dynamic Control)

`mod_xml_curl` is typically used for fetching entire XML configurations (dialplan, directory, etc.) from a web server when FreeSWITCH starts or reloads XML. While it can be used for dynamic routing by having FreeSWITCH fetch dialplan XML on a per-call basis, it's generally less suited for step-by-step TwiML-like interactive call control compared to Lua+HTTP or `mod_httapi`.

**Agbara-Go API Endpoints for FreeSWITCH:**

Your Agbara-Go application will need to expose HTTP endpoints that FreeSWITCH can call. These endpoints (defined by `Application.VoiceUrl`) will receive call parameters from FreeSWITCH and must respond with TwiML-like XML that FreeSWITCH can understand and execute. The `pkg/twiml` package in Agbara-Go will assist in generating this XML.

**Key Variables to Pass from FreeSWITCH to Agbara-Go:**
When FreeSWITCH makes an HTTP request to Agbara-Go, ensure it passes at least:
*   `Channel-Call-UUID` (or `uuid`): The unique ID of the call leg.
*   `AccountSid`: The Agbara Account SID associated with the call.
*   `Caller-Caller-ID-Number` / `Caller-Caller-ID-Name`
*   `Caller-Destination-Number`
*   `Digits` (if any input was gathered, e.g., via `<Gather>`)

These allow Agbara-Go to provide contextually relevant TwiML instructions.
