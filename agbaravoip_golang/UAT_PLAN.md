# AgbaraVOIP GoLang Service: User Acceptance Testing (UAT) Plan

## 1. Introduction and Objectives

This document outlines the plan for User Acceptance Testing (UAT) of the AgbaraVOIP GoLang service. The primary objectives of this UAT are:

*   Verify that the core functionalities of the reimplemented AgbaraVOIP service (up to Phase 6 completion) meet user requirements and business needs.
*   Ensure the system is usable and performs as expected from a user's perspective.
*   Identify any critical issues, bugs, or deviations from the expected behavior before considering production readiness.
*   Build user confidence in the new system.

## 2. Scope of UAT

The scope of this UAT includes the following core features and functionalities based on the `GOLANG_POSTGRES_IMPLEMENTATION_PLAN.md` up to Phase 6:

*   **Account Management:**
    *   Master Account creation.
    *   Sub-account creation under a master account.
    *   Retrieval of account details (self-service via API).
*   **Application Management (via API):**
    *   Creating voice/SMS applications.
    *   Listing applications.
    *   Updating application details.
    *   Deleting applications.
*   **Call Origination (via API):**
    *   Initiating outbound calls.
*   **AgbaraXML Call Control (Core Verbs):**
    *   `<Say>`: Text-to-speech.
    *   `<Play>`: Playing audio files.
    *   `<Gather>`: Collecting DTMF digits.
    *   `<Record>`: Call recording (initiation and basic metadata handling).
    *   `<Dial>` (primitive for single number and for `<Conference>`).
    *   `<Conference>`: Basic multi-party conferencing.
    *   `<Hangup>`: Terminating calls.
    *   `<Pause>`: Pausing call flow.
    *   `<Redirect>`: Transferring call control to a new AgbaraXML document.
*   **SMS Functionality:**
    *   Sending SMS via the `<Sms>` AgbaraXML verb.
    *   Receiving inbound SMS via the `/api/v1/sms/inbound` webhook and subsequent AgbaraXML processing via the application's `sms_url`.

**Out of Scope for this UAT (unless Phase 7 items were implicitly completed):**

*   Advanced security features (e.g., detailed RBAC beyond account ownership).
*   Full performance and scalability testing under high load.
*   Specific admin APIs for Freeswitch Server & Gateway CRUD (unless simple versions exist).
*   JWT authentication (UAT will primarily use Basic Auth as per current API documentation, unless JWT is explicitly provided as the method for testers).
*   UI-based interactions (UAT will focus on API interactions and the resulting telephony behavior).

## 3. UAT Environment Requirements

*   **Deployed AgbaraVOIP GoLang Service:** A stable, deployed instance of the GoLang application, including all its components (API server, ESL listener).
*   **PostgreSQL Database:** Accessible and populated with necessary initial data if any.
*   **Freeswitch Instance:** Configured to work with the deployed AgbaraVOIP GoLang service (e.g., outbound ESL connections pointing to the Go service).
*   **Test Application Server(s):** One or more web servers capable of:
    *   Serving static AgbaraXML files from configured URLs (for `voice_url`, `sms_url`, `action` URLs in `<Gather>`, etc.).
    *   Logging HTTP requests received from AgbaraVOIP (e.g., to verify `<Gather>` actions or status callbacks).
*   **API Client:** A tool to make HTTP requests to the AgbaraVOIP API (e.g., Postman, curl, or a custom script).
*   **Telephony Endpoints:**
    *   At least two phone numbers or SIP clients for making and receiving calls, and participating in conferences.
    *   A way to send/receive actual SMS messages if testing the `<Sms>` verb with a live gateway, or a simulator.
*   **Accessible Audio File:** A short, publicly accessible audio URL for testing the `<Play>` verb.

## 4. Roles and Responsibilities (Conceptual)

*   **UAT Coordinator:** Oversees the UAT process, provides support to testers, and manages feedback.
*   **Testers:** Typically end-users, product owners, QA personnel, or stakeholders who will execute the test cases.
*   **Development Team:** On standby to address critical issues found during UAT and to help diagnose problems.

## 5. High-Level Test Scenarios & Cases

Testers should aim to cover the following scenarios. Each scenario may have multiple test cases.

**5.1. Account Management**
    *   **TC_AM_01:** Successfully create a new master account via API.
        *   *Expected:* Account created, SID returned, can be used for subsequent API calls.
    *   **TC_AM_02:** Attempt to create a master account with missing required fields (e.g., no auth_token).
        *   *Expected:* API returns a validation error.
    *   **TC_AM_03:** Successfully create a sub-account under an existing master account via API.
        *   *Expected:* Sub-account created with correct parent SID.
    *   **TC_AM_04:** List sub-accounts for a master account.
        *   *Expected:* Correct list of sub-accounts is returned.
    *   **TC_AM_05:** Retrieve details of the master account itself.
        *   *Expected:* Correct account details are returned.

**5.2. Application Management**
    *   **TC_APP_01:** Create a new application with valid `voice_url` and `sms_url` via API.
        *   *Expected:* Application created, SID returned.
    *   **TC_APP_02:** List applications for an account.
        *   *Expected:* Newly created application is present in the list.
    *   **TC_APP_03:** Update the friendly name and a URL (e.g., `voice_url`) of an existing application.
        *   *Expected:* Application details are updated.
    *   **TC_APP_04:** Delete an application.
        *   *Expected:* Application is deleted and no longer appears in the list.

**5.3. Call Origination & AgbaraXML Processing**
    *   **Setup:** Prepare simple AgbaraXML files hosted on an accessible web server.
    *   **TC_CALL_01 (Say & Hangup):** Originate a call to an application that returns `<Response><Say>UAT test successful.</Say><Hangup/></Response>`.
        *   *Expected:* Call connects, message is heard, call disconnects.
    *   **TC_CALL_02 (Play & Hangup):** Originate a call to an application that returns `<Response><Play>http://<public_audio_url>/test.mp3</Play><Hangup/></Response>`.
        *   *Expected:* Call connects, audio file is played, call disconnects.
    *   **TC_CALL_03 (Pause):** Originate a call with `<Response><Say>Pausing for 3 seconds.</Say><Pause length="3"/><Say>Resumed.</Say><Hangup/></Response>`.
        *   *Expected:* First message, noticeable pause, second message, hangup.
    *   **TC_CALL_04 (Gather):** Originate a call to an application with `<Gather action="http://<your_server>/log_digits" numDigits="3"><Say>Please enter 3 digits.</Say></Gather><Say>Thank you.</Say><Hangup/>`.
        *   *Expected:* Prompt is played, user enters 3 digits, "Thank you" is heard, call hangs up. The `log_digits` endpoint on `<your_server>` should receive a request containing the entered digits.
    *   **TC_CALL_05 (Record):** Originate a call with `<Record action="http://<your_server>/log_recording" maxLength="5" playBeep="true"/><Say>Recording finished.</Say><Hangup/>`.
        *   *Expected:* Beep is heard, a short recording is made, "Recording finished" is heard. The `log_recording` endpoint should receive metadata about the recording (URL, duration). (Actual recording file retrieval might be a separate verification step by admins).
    *   **TC_CALL_06 (Redirect):** Call an app with `<Response><Say>Redirecting.</Say><Redirect>http://<your_server>/redirected_logic.xml</Redirect></Response>`. The `redirected_logic.xml` should contain `<Response><Say>Redirect successful.</Say><Hangup/></Response>`.
        *   *Expected:* "Redirecting" is heard, then "Redirect successful" is heard, then hangup.

**5.4. Conference Calls (via AgbaraXML)**
    *   **Setup:** AgbaraXML: `<Response><Dial><Conference>UAT_Room_123</Conference></Dial></Response>`
    *   **TC_CONF_01 (Two Participants):**
        1.  User A originates a call to an application using the conference AgbaraXML above.
        2.  User B originates a call to the same application.
        *   *Expected:* Both User A and User B are connected to "UAT_Room_123" and can talk to each other.
    *   **TC_CONF_02 (Conference End):** One user hangs up.
        *   *Expected:* The other user might remain or be disconnected based on Freeswitch conference profile settings (test current behavior).

**5.5. SMS Functionality**
    *   **TC_SMS_01 (Send SMS via XML):**
        *   **Setup:** AgbaraXML: `<Response><Sms to="+1XXXXXXXXXX" from="+1YYYYYYYYYY" action="http://<your_server>/sms_status">Test SMS from Agbara UAT.</Sms><Hangup/></Response>` (Replace numbers with valid test numbers).
        *   Originate a call to an application that returns this XML.
        *   *Expected:* The destination number receives the SMS. The `sms_status` endpoint receives a callback indicating success/failure. (Requires a configured SMS gateway or simulator).
    *   **TC_SMS_02 (Receive Inbound SMS):**
        *   **Setup:** An application is configured with an `sms_url` pointing to `http://<your_server>/receive_sms.xml`. This XML should be simple, e.g., `<Response><Say>SMS received handler.</Say></Response>` (though `<Say>` is for voice, for testing the XML fetch it's okay, or use `<Sms>reply</Sms>`).
        *   Manually POST a simulated inbound SMS payload (matching gateway format if known, or a generic one) to the `/api/v1/sms/inbound` AgbaraVOIP endpoint.
        *   *Expected:* The AgbaraVOIP service requests `http://<your_server>/receive_sms.xml`. The request to `receive_sms.xml` should contain parameters from the inbound SMS (From, To, Body).

## 6. Test Execution Process

1.  **Preparation:** Testers familiarize themselves with the UAT plan, test cases, and API client tools. The UAT environment is confirmed to be ready.
2.  **Execution:** Testers execute the defined test scenarios and any exploratory tests they deem necessary.
3.  **Logging:** All test results, issues, bugs, and observations are logged with detailed steps to reproduce, expected vs. actual results, and any relevant logs or screenshots.
4.  **Reporting:** Regular updates on UAT progress and critical issues are provided to the UAT Coordinator.

## 7. Success Criteria

UAT will be considered successful if:

*   All critical and major bugs identified during UAT are resolved and retested successfully.
*   Core functionalities (as defined in the Scope) are working as expected.
*   Users are confident in using the system for its intended purpose.
*   A formal sign-off is provided by the Product Owner or key stakeholders.

## 8. Feedback and Issue Management

*   A designated tool or document (e.g., JIRA, spreadsheet) will be used for logging UAT feedback and tracking issues.
*   Each issue will be assigned a priority (Critical, Major, Minor).
*   The development team will review and address issues based on priority.
*   Resolved issues will be retested by the UAT team.

---
This UAT plan provides a baseline. Testers are encouraged to perform exploratory testing around the defined scenarios.
