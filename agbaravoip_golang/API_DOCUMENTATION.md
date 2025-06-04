# AgbaraVOIP GoLang API Documentation

## Introduction

Welcome to the AgbaraVOIP GoLang API. This API allows you to manage your voice and messaging resources, create applications, and control calls.

### Authentication

Most API endpoints require authentication. The API uses HTTP Basic Authentication. You will need to provide your Account SID as the username and your Auth Token as the password.

Example: `Authorization: Basic <base64_encoded_credentials>`
Where `<base64_encoded_credentials>` is `AccountSID:AuthToken` encoded in Base64.

Endpoints that are publicly accessible (e.g., creating a master account) will be explicitly noted.

### API Versioning

The current API version is `v1`. All API paths are prefixed with `/api/v1`.

### Base URL

The base URL for all API requests is: `https://<your_agbaravoip_domain>/api/v1`

*(Note: Replace `<your_agbaravoip_domain>` with the actual domain of your AgbaraVOIP deployment.)*

## 1. Accounts

The Account resource represents a user or entity that can own applications, numbers, and make/receive calls and messages.

### Create a Master Account

*   **POST** `/accounts`
*   **Description:** Creates a new master AgbaraVOIP account. This is typically the first step for a new user.
*   **Authentication:** None (Publicly accessible for signup)
*   **Request Body:** `application/json`
    ```json
    {
        "friendly_name": "My Company Name",
        "auth_token": "a_very_strong_password123!"
    }
    ```
    *   `friendly_name` (string, optional): A human-readable name for the account.
    *   `auth_token` (string, required): The authentication token (password) for this account. This will be used for API authentication.
*   **Responses:**
    *   `201 Created`:
        ```json
        {
            "sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "parent_sid": null,
            "friendly_name": "My Company Name",
            "phone_number": "",
            "type": "full",
            "status": "active",
            "created_at": "2023-10-27T10:00:00Z",
            "updated_at": "2023-10-27T10:00:00Z"
        }
        ```
    *   `400 Bad Request` (Validation Error):
        ```json
        {
            "error": "Validation failed",
            "details": "auth_token is required"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not create account",
            "details": "A database error occurred or another internal issue."
        }
        ```

### Get Account Details

*   **GET** `/accounts/{account_sid}`
*   **Description:** Retrieves the details of a specific account.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
    *   Note: You can only retrieve details for the account associated with the provided credentials.
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account to retrieve (must match authenticated account).
*   **Responses:**
    *   `200 OK`:
        ```json
        {
            "sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "parent_sid": null,
            "friendly_name": "My Company Name",
            "phone_number": "",
            "type": "full",
            "status": "active",
            "created_at": "2023-10-27T10:00:00Z",
            "updated_at": "2023-10-27T10:00:00Z"
        }
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `403 Forbidden`:
        ```json
        {
            "error": "Forbidden",
            "details": "You can only retrieve your own account details."
        }
        ```
    *   `404 Not Found`:
        ```json
        {
            "error": "Account not found"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve account",
            "details": "An internal error occurred."
        }
        ```

### Create a Subaccount

*   **POST** `/accounts/{account_sid}/subaccounts`
*   **Description:** Creates a new subaccount under the authenticated master account.
*   **Authentication:** Basic Auth (Master Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the master account under which the subaccount will be created. Must match the authenticated account.
*   **Request Body:** `application/json` (using `CreateAccountRequest` DTO)
    ```json
    {
        "friendly_name": "Subaccount Team A",
        "auth_token": "subaccount_strong_password"
    }
    ```
    *   `friendly_name` (string, optional): A human-readable name for the subaccount.
    *   `auth_token` (string, required): The authentication token (password) for this new subaccount. This will be used for API authentication for the subaccount itself.
*   **Responses:**
    *   `201 Created`: (using `AccountResponse` DTO for the new subaccount)
        ```json
        {
            "sid": "SAyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
            "parent_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "friendly_name": "Subaccount Team A",
            "phone_number": "",
            "type": "full",
            "status": "active",
            "created_at": "2023-10-27T10:30:00Z",
            "updated_at": "2023-10-27T10:30:00Z"
        }
        ```
        *(Note: The `sid` for a subaccount might have a different prefix, e.g., `SA`, than a master account's `AC` prefix. The `type` and `status` will be set according to system defaults for new subaccounts.)*
    *   `400 Bad Request` (Validation Error):
        ```json
        {
            "error": "Validation failed",
            "details": "auth_token is required"
        }
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `403 Forbidden`:
        ```json
        {
            "error": "Forbidden",
            "details": "Authenticated account cannot create subaccounts or is not a master account."
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not create subaccount",
            "details": "An internal error occurred."
        }
        ```

### List Subaccounts

*   **GET** `/accounts/{account_sid}/subaccounts`
*   **Description:** Retrieves a list of all subaccounts belonging to the authenticated master account.
*   **Authentication:** Basic Auth (Master Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the master account whose subaccounts are to be listed. Must match the authenticated account.
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "SAyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
                "parent_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "friendly_name": "Subaccount Team A",
                "phone_number": "",
                "type": "full",
                "status": "active",
                "created_at": "2023-10-27T10:30:00Z",
                "updated_at": "2023-10-27T10:30:00Z"
            },
            {
                "sid": "SAzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
                "parent_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "friendly_name": "Subaccount Team B",
                "phone_number": "",
                "type": "trial",
                "status": "suspended",
                "created_at": "2023-10-27T10:35:00Z",
                "updated_at": "2023-10-27T10:35:00Z"
            }
        ]
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `403 Forbidden`:
        ```json
        {
            "error": "Forbidden",
            "details": "Authenticated account cannot list subaccounts or is not a master account."
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve subaccounts",
            "details": "An internal error occurred."
        }
        ```

---

## 2. Applications

Applications define how AgbaraVOIP handles incoming calls or messages for your numbers, by specifying URLs that return AgbaraXML instructions.

### Create an Application

*   **POST** `/accounts/{account_sid}/applications`
*   **Description:** Creates a new voice/SMS application under the authenticated account.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account that will own this application. Must match the authenticated account.
*   **Request Body:** `application/json`
    ```json
    {
        "friendly_name": "My Call Handling App",
        "voice_url": "https://myapp.com/handle_voice",
        "voice_method": "POST",
        "voice_fallback_url": "https://myapp.com/handle_voice_fallback",
        "voice_fallback_method": "POST",
        "sms_url": "https://myapp.com/handle_sms",
        "sms_method": "POST",
        "sms_fallback_url": "https://myapp.com/handle_sms_fallback",
        "sms_fallback_method": "POST",
        "status_callback_url": "https://myapp.com/status_updates",
        "status_callback_method": "POST"
    }
    ```
    *   `friendly_name` (string, required): A human-readable name for the application.
    *   `voice_url` (string, optional): URL AgbaraVOIP will request when a call is received.
    *   `voice_method` (string, optional): HTTP method (e.g., `GET`, `POST`) for `voice_url`. Defaults to `POST`.
    *   `voice_fallback_url` (string, optional): URL if `voice_url` fails.
    *   `voice_fallback_method` (string, optional): HTTP method for `voice_fallback_url`. Defaults to `POST`.
    *   `sms_url` (string, optional): URL for incoming SMS messages.
    *   `sms_method` (string, optional): HTTP method for `sms_url`. Defaults to `POST`.
    *   `sms_fallback_url` (string, optional): URL if `sms_url` fails.
    *   `sms_fallback_method` (string, optional): HTTP method for `sms_fallback_url`. Defaults to `POST`.
    *   `status_callback_url` (string, optional): URL for call status updates.
    *   `status_callback_method` (string, optional): HTTP method for `status_callback_url`. Defaults to `POST`.
*   **Responses:**
    *   `201 Created`:
        ```json
        {
            "sid": "APyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
            "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "friendly_name": "My Call Handling App",
            "voice_url": "https://myapp.com/handle_voice",
            "voice_method": "POST",
            "voice_fallback_url": "https://myapp.com/handle_voice_fallback",
            "voice_fallback_method": "POST",
            "sms_url": "https://myapp.com/handle_sms",
            "sms_method": "POST",
            "sms_fallback_url": "https://myapp.com/handle_sms_fallback",
            "sms_fallback_method": "POST",
            "status_callback_url": "https://myapp.com/status_updates",
            "status_callback_method": "POST",
            "created_at": "2023-10-27T11:00:00Z",
            "updated_at": "2023-10-27T11:00:00Z"
        }
        ```
    *   `400 Bad Request` (Validation Error):
        ```json
        {
            "error": "Validation failed",
            "details": "friendly_name is required"
        }
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not create application",
            "details": "An internal error occurred."
        }
        ```

### Get Application Details

*   **GET** `/accounts/{account_sid}/applications/{app_sid}`
*   **Description:** Retrieves details for a specific application owned by the authenticated account.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
    *   `app_sid` (string, required): The SID of the application to retrieve.
*   **Responses:**
    *   `200 OK`: (Structure similar to `201 Created` response for Create Application)
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `404 Not Found`:
        ```json
        {
            "error": "Application not found"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve application",
            "details": "An internal error occurred."
        }
        ```

### List Applications

*   **GET** `/accounts/{account_sid}/applications`
*   **Description:** Retrieves all applications owned by the authenticated account.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "APyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "friendly_name": "My Call Handling App",
                "voice_url": "https://myapp.com/handle_voice",
                "voice_method": "POST",
                // ... other fields ...
                "created_at": "2023-10-27T11:00:00Z",
                "updated_at": "2023-10-27T11:00:00Z"
            },
            {
                "sid": "APzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "friendly_name": "Another App",
                // ... other fields ...
                "created_at": "2023-10-28T12:00:00Z",
                "updated_at": "2023-10-28T12:00:00Z"
            }
        ]
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve applications",
            "details": "An internal error occurred."
        }
        ```

### Update an Application

*   **PUT** `/accounts/{account_sid}/applications/{app_sid}`
*   **Description:** Updates details for a specific application. Allows partial updates.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
    *   `app_sid` (string, required): The SID of the application to update.
*   **Request Body:** `application/json` (Provide only fields to be updated)
    ```json
    {
        "friendly_name": "Updated App Name",
        "voice_url": "https://new.myapp.com/voice"
    }
    ```
    *   All fields from `CreateApplicationRequest` (except `account_sid`) can be updated.
*   **Responses:**
    *   `200 OK`: (Structure similar to `201 Created` response for Create Application, showing updated values)
    *   `400 Bad Request` (Validation Error or No fields to update):
        ```json
        {
            "error": "Validation failed", // or "No fields to update"
            "details": "Specific error message."
        }
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `404 Not Found`:
        ```json
        {
            "error": "Application not found"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not update application",
            "details": "An internal error occurred."
        }
        ```

### Delete an Application

*   **DELETE** `/accounts/{account_sid}/applications/{app_sid}`
*   **Description:** Deletes a specific application.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
    *   `app_sid` (string, required): The SID of the application to delete.
*   **Responses:**
    *   `204 No Content`: Successfully deleted.
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `404 Not Found`:
        ```json
        {
            "error": "Application not found"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not delete application",
            "details": "An internal error occurred."
        }
        ```

---

## 3. Calls

The Call resource represents a voice call connection.

### Create a Call (Originate an Outbound Call)

*   **POST** `/accounts/{account_sid}/calls`
*   **Description:** Initiates an outbound call from a number associated with your account to a destination number. Call control logic is typically fetched from an `answer_url` or an `application_sid`.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account making the call.
*   **Request Body:** `application/json`
    ```json
    {
        "from": "+12345678901",
        "to": "+19876543210",
        "answer_url": "https://myapp.com/outbound_call_logic.xml",
        "application_sid": "APzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
        "timeout_seconds": 60
    }
    ```
    *   `from` (string, required): The phone number (in E.164 format) to make the call from. This must be a number associated with your AgbaraVOIP account.
    *   `to` (string, required): The phone number (in E.164 format) or SIP URI to call.
    *   `answer_url` (string, optional): A URL that returns AgbaraXML instructions for controlling the call. If `application_sid` is provided and this is empty, the `voice_url` from the application will be used. One of `answer_url` or a valid `application_sid` (which has a `voice_url`) is required.
    *   `application_sid` (string, optional): The SID of an application to use for this call. Its `voice_url` will be used if `answer_url` is not provided. If both are provided, `answer_url` takes precedence for the initial webhook. The `application_sid` is still associated for call logging.
    *   `timeout_seconds` (integer, optional): The number of seconds to wait for the call to be answered. Default is system-dependent.
*   **Responses:**
    *   `201 Created`:
        ```json
        {
            "sid": "CAwwwwwwwwwwwwwwwwwwwwwwwwwwwwww",
            "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "application_sid": "APzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
            "from": "+12345678901",
            "to": "+19876543210",
            "answer_url": "https://myapp.com/outbound_call_logic.xml",
            "status": "queued", // Or "ringing", "in-progress" depending on system speed
            "direction": "outbound-api",
            "duration_seconds": 0,
            "price": "0.00000",
            "answered_by": null,
            "timeout_seconds": 60,
            "hangup_cause": null,
            "forwarded_from": null,
            "start_time": "", // May be empty if not yet started
            "answer_time": "", // May be empty if not yet answered
            "end_time": "", // May be empty if not yet ended
            "created_at": "2023-10-27T12:00:00Z",
            "updated_at": "2023-10-27T12:00:00Z"
        }
        ```
    *   `400 Bad Request` (Validation Error):
        ```json
        {
            "error": "Validation failed",
            "details": "from and to fields are required"
        }
        ```
        ```json
        {
            "error": "Validation failed",
            "details": "answer_url or valid application_sid is required"
        }
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not initiate call", // Or "Call origination failed at telephony level"
            "details": "An internal error occurred or Freeswitch error."
        }
        ```

### Get Call Details

*   **GET** `/accounts/{account_sid}/calls/{call_sid}`
*   **Description:** Retrieves details for a specific call.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
    *   `call_sid` (string, required): The SID of the call to retrieve.
*   **Responses:**
    *   `200 OK`: (Structure similar to `201 Created` response for Create Call, reflecting current call state)
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `404 Not Found`:
        ```json
        {
            "error": "Call not found"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve call",
            "details": "An internal error occurred."
        }
        ```

### List Calls

*   **GET** `/accounts/{account_sid}/calls`
*   **Description:** Retrieves a list of calls associated with the account. Supports filtering.
*   **Authentication:** Basic Auth (Account SID and Auth Token)
*   **Path Parameters:**
    *   `account_sid` (string, required): The SID of the account.
*   **Query Parameters (Optional):**
    *   `status` (string): Filter by call status (e.g., `completed`, `failed`, `in-progress`).
    *   `from` (string): Filter by the `from` phone number.
    *   `to` (string): Filter by the `to` phone number.
    *   *(Other filters like date ranges might be available based on service capabilities)*
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "CAwwwwwwwwwwwwwwwwwwwwwwwwwwwwww",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                // ... other call fields ...
                "status": "completed",
                "created_at": "2023-10-27T12:00:00Z",
                "updated_at": "2023-10-27T12:05:00Z"
            },
            {
                "sid": "CAqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                // ... other call fields ...
                "status": "failed",
                "created_at": "2023-10-27T13:00:00Z",
                "updated_at": "2023-10-27T13:00:10Z"
            }
        ]
        ```
    *   `401 Unauthorized`:
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Could not retrieve calls",
            "details": "An internal error occurred."
        }
        ```

---
### Note on In-Call Control, Conference/Recording Management, and SMS

The `GOLANG_POSTGRES_IMPLEMENTATION_PLAN.md` outlines several advanced API functionalities:

*   **In-Call Control:** Endpoints like `POST .../calls/{call_sid}/play`, `.../say`, `.../dtmf`, `.../record/start`, `.../record/stop`.
*   **Conference Management:** Endpoints for creating conferences, managing participants, and conference-level actions.
*   **Recording Management:** Endpoints for listing and retrieving recordings.
*   **SMS Management:** Endpoints for sending SMS messages (`POST .../sms/messages`) and listing/retrieving SMS details.

**Current Implementation Status:**

The currently available GoLang API handlers (`account_handlers.go`, `application_handlers.go`, `call_handlers.go`) primarily focus on Account, Application, and basic Call (origination, retrieval, listing) management.

**The advanced in-call control, comprehensive conference management, recording management, and SMS sending/management APIs are not directly implemented as discrete REST endpoints in this version of the GoLang service.**

Instead, these functionalities are expected to be primarily handled via **AgbaraXML** returned by your application URLs (`voice_url`, `answer_url`). For example:
*   To play audio in a call, your AgbaraXML would use the `<Play>` verb.
*   To record a call, your AgbaraXML would use the `<Record>` verb.
*   To initiate a conference, your AgbaraXML would use the `<Dial><Conference>...</Conference></Dial>` verbs.
*   Sending SMS messages is not covered by the current API handlers. This functionality, as per the original system design, might be handled through different mechanisms or is pending full reimplementation in the GoLang service's REST API.

The GoLang service processes this AgbaraXML and translates the verbs into commands for the Freeswitch media server. Future versions of the API may expose more of these features directly via REST endpoints. Refer to the AgbaraXML documentation for details on available verbs and their usage.

---

## 4. SMS (Short Message Service)

As noted in the section above, the `GOLANG_POSTGRES_IMPLEMENTATION_PLAN.md` includes plans for SMS message management API endpoints:
*   `POST /v1/accounts/{account_sid}/sms/messages` (Send SMS)
*   `GET /v1/accounts/{account_sid}/sms/messages/{sms_sid}` (Get SMS details)
*   `GET /v1/accounts/{account_sid}/sms/messages` (List SMS messages)

**Current Implementation Status:**
These specific REST API endpoints for sending and managing SMS messages are **not yet implemented** in the current GoLang service handlers.

SMS functionality (particularly receiving SMS and potentially sending them via application logic) would typically be defined by the `sms_url` and related settings in your **Application** resource, which would then process incoming messages using AgbaraXML or custom logic on your application server. Direct API-based SMS sending is not available in this version.

---

## Common Error Responses

Besides the specific error codes listed per endpoint, you may encounter standard HTTP error codes:

*   `400 Bad Request`: The request was malformed, contained invalid JSON, or failed validation. Check the `details` field for more information.
*   `401 Unauthorized`: Authentication failed (e.g., missing or incorrect credentials).
*   `403 Forbidden`: Authenticated user does not have permission to access the requested resource.
*   `404 Not Found`: The requested resource (e.g., account, application, call) does not exist.
*   `429 Too Many Requests`: You have exceeded the API rate limits.
*   `500 Internal Server Error`: An unexpected error occurred on the server. This could be due to a bug, a database issue, or problems with underlying services like Freeswitch.
*   `503 Service Unavailable`: The service is temporarily unavailable, possibly due to maintenance or overload.

Error responses generally follow this format:
```json
{
    "error": "Short error description",
    "details": "More detailed explanation of the error, if available."
}
```
