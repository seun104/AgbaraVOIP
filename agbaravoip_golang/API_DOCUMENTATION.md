# AgbaraVOIP GoLang API Documentation

## Introduction

Welcome to the AgbaraVOIP GoLang API. This API allows you to manage your voice and messaging resources, create applications, and control calls.

### Authentication

Most API endpoints require authentication using a JSON Web Token (JWT). To obtain a JWT, you must first authenticate against the `/api/v1/auth/token` endpoint using HTTP Basic Authentication (your Account SID as the username and your Auth Token as the password). This will return a JWT Bearer token.

Once you have the JWT, include it in the `Authorization` header for all subsequent protected API requests:

`Authorization: Bearer <your_jwt_token>`

Endpoints that are publicly accessible (e.g., creating a master account via `POST /api/v1/accounts` or the token generation endpoint itself) or use different authentication schemes (like webhooks) will be explicitly noted.

#### Token Expiration and Management
*   **Expiration:** JWTs have a limited lifetime (e.g., as specified by the `jwt_token_duration` configuration, typically 15 minutes to a few hours). If your token expires, the API will return a `401 Unauthorized` error. You will need to re-authenticate at the `/api/v1/auth/token` endpoint to obtain a new token.
*   **Refresh Tokens:** Refresh tokens are not currently implemented. You must re-authenticate with your credentials when your access token expires.
*   **Token Revocation:** Active server-side revocation of tokens (before their natural expiry) is not currently supported. For security, ensure tokens are kept confidential and token expiry times are reasonably short.

### Obtain an API Token

*   **POST** `/auth/token`
*   **Description:** Authenticates your account credentials (via HTTP Basic Auth) and returns a JWT Bearer token for use with other API endpoints.
*   **Authentication:** HTTP Basic Authentication (Account SID and Auth Token).
*   **Request Body:** None.
*   **Responses:**
    *   `200 OK`:
        ```json
        {
            "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
            "expires_at": "2023-10-28T12:15:00Z",
            "token_type": "Bearer"
        }
        ```
    *   `400 Bad Request` (e.g., missing Authorization header):
        ```json
        {
            "error": "Authorization header required"
        }
        ```
    *   `401 Unauthorized` (Invalid credentials):
        ```json
        {
            "error": "Invalid credentials"
        }
        ```
    *   `500 Internal Server Error`:
        ```json
        {
            "error": "Token generation failed",
            "details": "An internal error occurred."
        }
        ```

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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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
*   **Authentication:** JWT Bearer Token
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

### Live Call Control

These endpoints allow for real-time manipulation of active calls.

#### Play Audio on a Live Call

*   **POST** `/api/v1/accounts/{account_sid}/calls/{call_sid}/play`
*   **Description:** Initiates playback of an audio file on the specified call leg (defaults to 'aleg').
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `call_sid` (string, required): The SID of the live call.
*   **Request Body:** `application/json` (`CallPlayRequest`)
    ```json
    {
        "url": "http://example.com/audio/prompt.wav",
        "loop": 1,
        "legs": "aleg"
    }
    ```
    *   `url` (string, required): The URL of the audio file to play. Must be accessible by the AgbaraVOIP server.
    *   `loop` (integer, optional): Number of times to loop the playback. `0` or `1` means play once. Service default is `1`. (Note: True multi-looping for `>1` might depend on specific Freeswitch application used by the service and may not be fully supported by simple commands).
    *   `legs` (string, optional): The call leg(s) to play audio to. Can be `aleg` (default), `bleg`, or `both`.
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "call_sid": "CAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Audio playback initiated.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   `400 Bad Request`: Invalid request payload (e.g., missing URL, invalid legs value).
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `403 Forbidden`: Call does not belong to the account.
    *   `404 Not Found`: Call SID not found or call is not in a state that allows playback (e.g., already completed).
    *   `503 Service Unavailable`: Media server (ESL) command failed or client unavailable.

#### Speak Text on a Live Call

*   **POST** `/api/v1/accounts/{account_sid}/calls/{call_sid}/say`
*   **Description:** Initiates text-to-speech on the specified call leg (defaults to 'aleg').
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `call_sid` (string, required): The SID of the live call.
*   **Request Body:** `application/json` (`CallSayRequest`)
    ```json
    {
        "text": "Hello, this is a test message.",
        "language": "en-US",
        "voice": "slt",
        "legs": "aleg"
    }
    ```
    *   `text` (string, required): The text to speak.
    *   `language` (string, optional): Language code (e.g., "en-US", "es-ES"). Defaults to system/engine default.
    *   `voice` (string, optional): Specific voice to use (e.g., "slt", "kal"). Defaults to system/engine default.
    *   `legs` (string, optional): The call leg(s) to speak text to. `aleg` (default). (Note: `bleg` or `both` might require B-leg UUID for specific targeting with `uuid_speak` and may default to `aleg` in current service implementation).
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "call_sid": "CAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Text-to-speech initiated.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   `400 Bad Request`: Invalid request payload (e.g., missing text).
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `403 Forbidden`: Call does not belong to the account.
    *   `404 Not Found`: Call SID not found or call is not in a speakable state.
    *   `503 Service Unavailable`: Media server (ESL) command failed or client unavailable.

#### Send DTMF Tones on a Live Call

*   **POST** `/api/v1/accounts/{account_sid}/calls/{call_sid}/dtmf`
*   **Description:** Sends a sequence of DTMF tones on the specified call leg (defaults to 'aleg').
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `call_sid` (string, required): The SID of the live call.
*   **Request Body:** `application/json` (`CallDTMFRequest`)
    ```json
    {
        "digits": "1234#",
        "duration_ms": 250,
        "legs": "aleg"
    }
    ```
    *   `digits` (string, required): The DTMF digits to send (e.g., "1234#").
    *   `duration_ms` (integer, optional): Duration for each digit in milliseconds (e.g., 100 to 2000). Actual support may depend on channel variable `dtmf_duration` or Freeswitch configuration.
    *   `legs` (string, optional): The call leg(s) to send DTMF to. `aleg` (default). (Note: `bleg` or `both` might require B-leg UUID for specific targeting and may default to `aleg` in current service implementation).
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "call_sid": "CAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "DTMF send initiated.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   `400 Bad Request`: Invalid request payload (e.g., missing digits).
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `403 Forbidden`: Call does not belong to the account.
    *   `404 Not Found`: Call SID not found or not in a state for DTMF.
    *   `503 Service Unavailable`: Media server (ESL) command failed or client unavailable.

#### Manage Call Recording (Start/Stop)

*   **POST** `/api/v1/accounts/{account_sid}/calls/{call_sid}/record`
*   **Description:** Starts or stops recording on the live call.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `call_sid` (string, required): The SID of the live call.
*   **Request Body:** `application/json` (`CallRecordRequest`)
    *   To **start** recording:
        ```json
        {
            "action": "start",
            "file_name": "my_custom_recording_name",
            "max_duration_seconds": 3600,
            "format": "mp3",
            "play_beep": true
        }
        ```
    *   To **stop** recording:
        ```json
        {
            "action": "stop",
            "file_name": "my_custom_recording_name.mp3"
        }
        ```
        (Or the specific `recordingName` returned by the start action, or a general stop command if supported by service by omitting `file_name`).
    *   `action` (string, required): Must be `start` or `stop`.
    *   `file_name` (string, optional): For `start`, desired base name for the recording file (system may add prefixes/suffixes/timestamps). For `stop`, the specific recording file name/path to stop (as returned by start or known from events). If omitted on `stop`, might attempt to stop based on `call_sid`.
    *   `max_duration_seconds` (integer, optional, for `start`): Maximum recording duration.
    *   `format` (string, optional, for `start`): Recording format (e.g., "wav", "mp3"). Defaults to "wav".
    *   `play_beep` (boolean, optional, for `start`): If true, play a beep sound before starting recording. (Actual beep playback depends on service implementation).
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "call_sid": "CAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Call recording started: CAxxxx_timestamp.wav", // Example message for start
            // "message": "Call recording stop initiated for: my_custom_recording_name.mp3", // Example for stop
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
            // "recording_name": "CAxxxx_timestamp.wav" // If CallActionResponse is updated
        }
        ```
    *   `400 Bad Request`: Invalid request payload (e.g., missing action, invalid action).
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `403 Forbidden`: Call does not belong to the account.
    *   `404 Not Found`: Call SID not found or not in a valid state for the recording action.
    *   `503 Service Unavailable`: Media server (ESL) command failed or client unavailable.

#### Hang Up a Live Call

*   **POST** `/api/v1/accounts/{account_sid}/calls/{call_sid}/hangup`
*   **Description:** Terminates (hangs up) an active call.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `call_sid` (string, required): The SID of the live call to hang up.
*   **Request Body:** None.
*   **Query Parameters:**
    *   `cause` (string, optional): Hangup cause to send to Freeswitch (e.g., `NORMAL_CLEARING`). Defaults to `NORMAL_CLEARING`.
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "call_sid": "CAxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Hangup command accepted.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `403 Forbidden`: Call does not belong to the account.
    *   `404 Not Found`: Call SID not found.
    *   `503 Service Unavailable`: Media server (ESL) command failed or client unavailable.

---
### Note on AgbaraXML vs API Control

While many telephony actions can be controlled via direct API calls (as documented in the Live Call Control, Conference Management, and SMS Management sections), the system also robustly supports call control via **AgbaraXML** documents fetched from your application URLs (`voice_url`, `answer_url`, `sms_url`).

*   **API Control:** Offers direct, imperative manipulation of resources like live calls, conferences, and sending SMS messages. Useful for applications that manage state and logic externally.
*   **AgbaraXML Control:** Provides a declarative way to define call/SMS flow. The AgbaraVOIP server fetches and processes these XML documents, executing verbs like `<Say>`, `<Play>`, `<Dial>`, `<Record>`, `<Sms>`, `<Conference>`, etc. This is suitable for applications where the call/messaging logic is primarily defined by these XML documents.

Both methods can be used. For example, a call might be initiated via the API, with its `answer_url` pointing to an AgbaraXML document that defines the initial call flow. Later, the same call could be modified using the Live Call Control APIs.

---

## 4. SMS Management

Manage sending and retrieving SMS messages associated with your account.
Inbound SMS messages are typically received via a webhook configured on your Application's `sms_url`, which should return AgbaraXML for processing.

### Send an SMS Message

*   **POST** `/api/v1/accounts/{account_sid}/sms/messages`
*   **Description:** Sends a new outbound SMS message from your account.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
*   **Request Body:** `application/json` (`SendSMSRequest`)
    ```json
    {
        "from": "+15005550006",
        "to": "+15005550007",
        "body": "Hello from AgbaraVOIP API!",
        "status_callback_url": "https://yourapp.com/sms_status_updates"
    }
    ```
    *   `from` (string, required): The sender ID or phone number. Must be a number associated with your account or a valid alphanumeric sender ID (if supported by the carrier and your account).
    *   `to` (string, required): The recipient's phone number in E.164 format.
    *   `body` (string, required): The text content of the SMS message.
    *   `status_callback_url` (string, optional): A URL to which AgbaraVOIP will send status updates for this SMS message (e.g., "sent", "failed", "delivered").
*   **Responses:**
    *   `201 Created`:
        ```json
        {
            "sid": "SMxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "to": "+15005550007",
            "from": "+15005550006",
            "body": "Hello from AgbaraVOIP API!",
            "status": "queued",
            "direction": "outbound-api",
            "price": null,
            "price_unit": null,
            "error_code": null,
            "error_message": null,
            "gateway_message_sid": null,
            "sent_at": null,
            "delivered_at": null,
            "created_at": "2023-10-28T10:00:00Z",
            "updated_at": "2023-10-28T10:00:00Z"
        }
        ```
    *   `400 Bad Request`: Invalid request payload (e.g., missing required fields).
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `500 Internal Server Error`: Error during SMS processing or gateway interaction.

### List SMS Messages

*   **GET** `/api/v1/accounts/{account_sid}/sms/messages`
*   **Description:** Retrieves a list of SMS messages associated with your account. Supports filtering.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
*   **Query Parameters (Optional):**
    *   `to` (string): Filter by recipient phone number.
    *   `from` (string): Filter by sender phone number/ID.
    *   `status` (string): Filter by SMS status (e.g., `sent`, `failed`, `delivered`, `received`).
    *   `direction` (string): Filter by SMS direction (`inbound`, `outbound`, `outbound-api`).
    *   `date_from` (string): Filter messages created on or after this date (YYYY-MM-DD).
    *   `date_to` (string): Filter messages created on or before this date (YYYY-MM-DD).
    *   `limit` (integer): Maximum number of records to return (e.g., 50). Default: 20. Max: 100.
    *   `offset` (integer): Number of records to skip for pagination. Default: 0.
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "SMxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "to": "+15005550007",
                "from": "+15005550006",
                "body": "Hello from AgbaraVOIP API!",
                "status": "sent",
                "direction": "outbound-api",
                // ... other fields ...
                "created_at": "2023-10-28T10:00:00Z",
                "updated_at": "2023-10-28T10:00:05Z"
            },
            {
                "sid": "SMaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "to": "+15005550006",
                "from": "+15005550008",
                "body": "Inbound reply",
                "status": "received",
                "direction": "inbound",
                // ... other fields ...
                "created_at": "2023-10-28T09:50:00Z",
                "updated_at": "2023-10-28T09:50:00Z"
            }
        ]
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `500 Internal Server Error`: Error retrieving SMS messages.

### Get SMS Message Details

*   **GET** `/api/v1/accounts/{account_sid}/sms/messages/{sms_sid}`
*   **Description:** Retrieves details for a specific SMS message.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `sms_sid` (string, required): The SID of the SMS message to retrieve.
*   **Responses:**
    *   `200 OK`: (Structure similar to `SMSMessageResponse` example in Send SMS)
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `404 Not Found`: SMS message SID not found for this account.
    *   `500 Internal Server Error`: Error retrieving SMS message.

---

## 5. Recordings

Manage metadata for call and conference recordings. These endpoints allow you to list and retrieve information about your recordings and delete their metadata. Deleting recording metadata does not delete the actual audio file from storage.

### List Recordings

*   **GET** `/api/v1/accounts/{account_sid}/recordings`
*   **Description:** Retrieves a list of recordings associated with your account. Supports filtering.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
*   **Query Parameters (Optional):**
    *   `call_sid` (string): Filter recordings belonging to a specific Call SID.
    *   `conference_sid` (string): Filter recordings belonging to a specific Conference SID.
    *   `format` (string): Filter by recording format (e.g., `wav`, `mp3`).
    *   `date_from` (string): Filter recordings created on or after this date (YYYY-MM-DD).
    *   `date_to` (string): Filter recordings created on or before this date (YYYY-MM-DD).
    *   `limit` (integer): Maximum number of records to return. Default: 20. Max: 100.
    *   `offset` (integer): Number of records to skip for pagination. Default: 0.
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "RExxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "call_sid": "CAyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
                "conference_sid": null,
                "duration_seconds": 35,
                "file_path": "/var/lib/freeswitch/recordings/ACxxxx/CAyyyy_timestamp.wav",
                "format": "wav",
                "size_bytes": 560000,
                "created_at": "2023-10-28T14:30:00Z",
                "updated_at": "2023-10-28T14:30:35Z"
            }
        ]
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `500 Internal Server Error`: Error retrieving recordings.

### Get Recording Details

*   **GET** `/api/v1/accounts/{account_sid}/recordings/{recording_sid}`
*   **Description:** Retrieves details for a specific recording.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `recording_sid` (string, required): The SID of the recording to retrieve.
*   **Responses:**
    *   `200 OK`:
        ```json
        {
            "sid": "RExxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "call_sid": "CAyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
            "conference_sid": null,
            "duration_seconds": 35,
            "file_path": "/var/lib/freeswitch/recordings/ACxxxx/CAyyyy_timestamp.wav",
            "format": "wav",
            "size_bytes": 560000,
            "created_at": "2023-10-28T14:30:00Z",
            "updated_at": "2023-10-28T14:30:35Z"
        }
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `404 Not Found`: Recording SID not found for this account.
    *   `500 Internal Server Error`: Error retrieving recording.

### Delete Recording Metadata

*   **DELETE** `/api/v1/accounts/{account_sid}/recordings/{recording_sid}`
*   **Description:** Deletes the metadata for a specific recording. **This does not delete the actual audio file from storage.**
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `recording_sid` (string, required): The SID of the recording metadata to delete.
*   **Responses:**
    *   `204 No Content`: Successfully deleted metadata.
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `404 Not Found`: Recording SID not found for this account.
    *   `500 Internal Server Error`: Error deleting recording metadata.

---

## 6. Conferences

Manage multi-party conference calls. Conferences can be controlled via AgbaraXML (`<Dial><Conference>...</Conference></Dial>`) or through these API endpoints for listing, details, and live control.

### List Conferences

*   **GET** `/api/v1/accounts/{account_sid}/conferences`
*   **Description:** Retrieves a list of conferences associated with your account.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
*   **Query Parameters (Optional):**
    *   `status` (string): Filter by conference status (e.g., `in-progress`, `completed`, `initializing`).
    *   `friendly_name` (string): Filter by conference friendly name (supports partial match).
    *   `date_from` (string): Filter conferences created on or after this date (YYYY-MM-DD).
    *   `date_to` (string): Filter conferences created on or before this date (YYYY-MM-DD).
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "friendly_name": "DailyStandup_Room123",
                "status": "in-progress",
                "start_time": "2023-10-28T09:00:00Z",
                "end_time": null,
                "created_at": "2023-10-28T08:59:00Z",
                "updated_at": "2023-10-28T09:00:00Z"
            }
        ]
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `500 Internal Server Error`: Error retrieving conferences.

### Get Conference Details

*   **GET** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}`
*   **Description:** Retrieves details for a specific conference.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the conference to retrieve.
*   **Responses:**
    *   `200 OK`:
        ```json
        {
            "sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "friendly_name": "DailyStandup_Room123",
            "status": "in-progress",
            "start_time": "2023-10-28T09:00:00Z",
            "end_time": null,
            "created_at": "2023-10-28T08:59:00Z",
            "updated_at": "2023-10-28T09:00:00Z"
        }
        ```
    *   `401 Unauthorized`: Invalid or missing JWT.
    *   `404 Not Found`: Conference SID not found for this account.
    *   `500 Internal Server Error`: Error retrieving conference.

### Live Conference Control

These endpoints allow real-time manipulation of an active conference.

#### Play Audio in Conference

*   **POST** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/play`
*   **Description:** Plays an audio file to all participants in the conference.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the live conference.
*   **Request Body:** `application/json` (`ConferenceControlPlayRequest`, alias of `CallPlayRequest`)
    ```json
    {
        "url": "http://example.com/audio/announce.wav",
        "loop": 1
    }
    ```
    *   `url` (string, required): URL of the audio file.
    *   `loop` (integer, optional): Number of times to play. `0` or `1` for once. Default `1`.
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Play audio initiated.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   Common errors: 400, 401, 404 (if conference not found or not in-progress), 503.

#### Speak Text in Conference

*   **POST** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/say`
*   **Description:** Speaks text to all participants in the conference using TTS.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:** As above.
*   **Request Body:** `application/json` (`ConferenceControlSayRequest`, alias of `CallSayRequest`)
    ```json
    {
        "text": "This conference will now be recorded.",
        "language": "en-US",
        "voice": "slt"
    }
    ```
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Say text initiated.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   Common errors: 400, 401, 404, 503.

#### Manage Conference Recording (Start/Stop)

*   **POST** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/record`
*   **Description:** Starts or stops recording the conference.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:** As above.
*   **Request Body:** `application/json` (`ConferenceControlRecordRequest`, alias of `CallRecordRequest`)
    *   To **start**: `{"action": "start", "file_name": "conf_rec_xyz", "format": "mp3"}`
    *   To **stop**: `{"action": "stop", "file_name": "conf_rec_xyz.mp3"}` (or specific name from start)
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Conference recording started: CFxxxx_timestamp.mp3", // Or stop message
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
            "recording_name": "CFxxxx_timestamp.mp3" // If action was start
        }
        ```
    *   Common errors: 400, 401, 404, 503.

### Conference Participants

Manage participants within a specific conference.

#### List Participants

*   **GET** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants`
*   **Description:** Retrieves a list of current participants in the specified conference.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the conference.
*   **Responses:**
    *   `200 OK`:
        ```json
        [
            {
                "sid": "CPyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
                "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "call_sid": "CAzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
                "account_sid": "ACxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
                "is_muted": false,
                "is_moderator": true,
                "join_time": "2023-10-28T09:00:05Z",
                "leave_time": null
            }
        ]
        ```
    *   `401 Unauthorized`.
    *   `404 Not Found` (Conference not found).
    *   `500 Internal Server Error`.

#### Get Participant Details

*   **GET** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_sid}`
*   **Description:** Retrieves details for a specific participant in a conference.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the conference.
    *   `participant_sid` (string, required): The SID of the participant (e.g., `CPxxxxxxxx`).
*   **Responses:**
    *   `200 OK`: (Structure similar to `ParticipantResponse` example in List Participants)
    *   `401 Unauthorized`.
    *   `404 Not Found` (Conference or Participant not found).
    *   `500 Internal Server Error`.

### Live Participant Control

Manage individual participants in an active conference.

#### Mute/Unmute a Participant

*   **PUT** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_call_sid}/mute`
*   **Description:** Sets the mute status for a specific participant in a conference. The `participant_call_sid` refers to the Call SID of the participant's leg in the conference, which often serves as their member ID in Freeswitch.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the conference.
    *   `participant_call_sid` (string, required): The Call SID of the participant to mute/unmute.
*   **Request Body:** `application/json` (`ParticipantMuteRequest`)
    ```json
    {
        "mute": true
    }
    ```
    *   `mute` (boolean, required): `true` to mute, `false` to unmute.
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Mute action initiated for participant CAzzzzzzzzzzzzzzzzzzzzzzzzzzzz.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   Common errors: 400, 401, 404, 503.

#### Kick a Participant

*   **POST** `/api/v1/accounts/{account_sid}/conferences/{conf_sid}/participants/{participant_call_sid}/kick`
*   **Description:** Removes (kicks) a specific participant from a conference. The `participant_call_sid` refers to the Call SID of the participant's leg.
*   **Authentication:** JWT Bearer Token
*   **Path Parameters:**
    *   `account_sid` (string, required): Your Account SID.
    *   `conf_sid` (string, required): The SID of the conference.
    *   `participant_call_sid` (string, required): The Call SID of the participant to kick.
*   **Request Body:** None.
*   **Responses:**
    *   `202 Accepted`:
        ```json
        {
            "conference_sid": "CFxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
            "success": true,
            "message": "Kick action initiated for participant CAzzzzzzzzzzzzzzzzzzzzzzzzzzzz.",
            "job_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
        }
        ```
    *   Common errors: 401, 404, 503.


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
