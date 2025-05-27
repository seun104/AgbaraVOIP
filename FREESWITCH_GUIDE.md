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
