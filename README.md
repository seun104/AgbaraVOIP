# Agbara-Go: Voice & Communication API

Agbara-Go is a Golang-based RESTful API providing functionalities for managing voice calls, user accounts, multi-participant conferences, and application configurations for voice/SMS webhooks. It integrates with FreeSWITCH for call handling and uses PostgreSQL as its database.

**Core Features:**
*   **Account Management**: Manage master and sub-accounts, including authentication tokens and outbound gateway preferences.
*   **Application Management**: Configure applications with webhook URLs for voice (TwiML-based) and SMS handling.
*   **Call Management**: 
    *   Originate outbound calls via FreeSWITCH.
    *   Call routing can be determined by an Agbara Application (which provides a `VoiceUrl` for TwiML control) or a direct FreeSWITCH application string.
    *   Supports account-specific outbound gateway selection.
    *   Real-time call status updates (e.g., ringing, answered, completed) by processing FreeSWITCH events.
*   **Conference Management**: 
    *   Create/list/get conferences.
    *   Manage participants: list, get.
    *   Participant actions (mute/unmute, kick) and conference actions (record, play) are stubbed at the service layer but have API endpoints. Telephony interaction for these is pending full FreeSWITCH implementation.
    *   Real-time conference/participant status updates via FreeSWITCH events.
*   **TwiML Generation**: `pkg/twiml` allows Agbara-Go to respond with XML instructions to control call flow when FreeSWITCH makes HTTP requests to it (typically to an `Application.VoiceUrl`).
*   **FreeSWITCH Event Handling**: A persistent ESL connection listens for FreeSWITCH events, which are dispatched to services to update database records in real-time.

## Prerequisites
*   Go (version 1.19 or higher recommended)
*   PostgreSQL database server (version 12 or higher recommended)
*   A running and configured FreeSWITCH instance. See [FREESWITCH_GUIDE.md](./FREESWITCH_GUIDE.md) for details on required FreeSWITCH setup for ESL connectivity, event subscription, HTTP callbacks (e.g., via Lua scripts), and gateway configuration.

## Building the Application

Navigate to the root directory of the project (`agbara-go`) and run:
```bash
go build -o agbara-server ./cmd/server/main.go
```
This creates an executable named `agbara-server`.

## Configuration

The application is configured using environment variables:

*   `AGBARA_DB_DSN`: PostgreSQL Data Source Name.
    *   Example: `postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable`
    *   Default: `postgres://user:password@localhost:5432/agbaradb?sslmode=disable`
*   `HTTP_PORT`: Port for the Agbara-Go HTTP API server. (Default: `8080`)
*   `X-Auth-User-Sid` (HTTP Header): **Placeholder for API authentication.** Identifies the user/account. **NOT FOR PRODUCTION.**
*   **FreeSWITCH ESL Connection:**
    *   `FS_HOST`: Hostname/IP of FreeSWITCH ESL. (Default: `localhost`)
    *   `FS_PORT`: ESL port. (Default: `8021`)
    *   `FS_PASSWORD`: ESL password. (Default: `YourESLPassword` - **CHANGE THIS!**)
    *   `FS_TIMEOUT_SECONDS`: ESL connection timeout. (Default: `10`)
    *   `FS_MAX_RETRIES`: ESL connection retries. (Default: `3`)
    *   `FS_EVENT_SUBSCRIPTIONS`: Space-separated list of FreeSWITCH events to subscribe to.
        *   Default: `CHANNEL_CREATE CHANNEL_ANSWER CHANNEL_HANGUP_COMPLETE CHANNEL_PROGRESS_MEDIA CUSTOM conference::maintenance RECORD_STOP`
    *   `FS_DEFAULT_GATEWAY_NAME`: (Optional) System-wide default FreeSWITCH gateway for outbound calls if no account-specific gateway is set.

## Database Setup

Database schemas are in `db/schema/`. Apply them to your PostgreSQL database in order:
1.  `001_create_calls_table.sql` (defines `calls` table, `update_modified_column` function)
2.  `002_create_accounts_table.sql` (defines `accounts` table, including gateway setting columns)
3.  `003_create_conference_tables.sql` (defines `conferences`, `participants` tables)
4.  `004_create_applications_table.sql` (defines `applications` table)

**Example `psql` commands:**
```bash
# PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/001_create_calls_table.sql
# PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/002_create_accounts_table.sql
# ... and so on for all schema files.
```

## Running the Application

1.  **Set Environment Variables** (example for bash/zsh):
    ```bash
    export AGBARA_DB_DSN="postgres://user:pass@host:port/db?sslmode=disable"
    export FS_HOST="your_freeswitch_host"
    export FS_PASSWORD="your_esl_password"
    # Optionally set other variables like HTTP_PORT, FS_PORT, FS_DEFAULT_GATEWAY_NAME, etc.
    ```
2.  **Run the Executable**: `./agbara-server`
    The server will log startup messages, including ESL connection status. For API calls, include the `X-Auth-User-Sid` header.

## API Endpoints

All API endpoints are prefixed with `/api/v1`. Placeholder authentication via `X-Auth-User-Sid` header is used.

### Account Management
Base Path: `/api/v1/Accounts`

*   **POST `/Master`**: Creates a master account. (Req: `models.CreateAccountRequest`)
*   **GET ``**: Lists sub-accounts for `X-Auth-User-Sid`.
*   **POST ``**: Creates a sub-account under `X-Auth-User-Sid`. (Req: `models.CreateAccountRequest`)
*   **GET `/{accountSid}`**: Retrieves `accountSid`. (Auth: `X-Auth-User-Sid` must match `accountSid`)
*   **POST `/{accountSid}`**: Modifies status of `accountSid`. (Req: `models.ChangeAccountStatusRequest`, Auth: self)
*   **PATCH `/{accountSid}/settings`**: Updates settings for `accountSid`. (Req: `models.UpdateAccountSettingsRequest` for `FriendlyName`, `PhoneNumber`, `Type`, `DefaultOutboundGateway`, `GatewaySelectionScript`. Auth: self)
*   **POST `/{accountSid}/AuthToken`**: Regenerates AuthToken for `accountSid`. (Auth: self)

---
Base Path for Calls, Conferences, Applications: `/api/v1/Accounts/{accountSidInPath}`
(Requires `X-Auth-User-Sid` header matching `{accountSidInPath}`)

### Call Management
Relative Path: `/Calls`

*   **GET ``**: Lists calls for `{accountSidInPath}`.
*   **POST `/Call`**: Creates and originates a call via FreeSWITCH.
    *   Request: `models.CallRequest`. Key fields:
        *   `to`, `from`
        *   `applicationSid` (recommended): SID of an Agbara Application. Its `VoiceUrl` is invoked by FreeSWITCH (via Lua script pattern) to fetch TwiML.
        *   `answerUrl`: (Alternative) Direct FreeSWITCH app string or dialplan target.
        *   `timeLimit`: Max call duration (sets `call_timeout` FS variable).
        *   `sendDigits`, `statusCallbackUrl`, `statusCallbackMethod`, `hangupOnRing`: Passed as transient channel variables to FreeSWITCH.
    *   Response: `models.Call` (includes `freeswitchCallId`). Call status updated by FreeSWITCH events.
*   **POST `/{callSid}`**: Modifies call `callSid` (e.g., `AnswerUrl`).

### Conference Management
Relative Path: `/Conferences`
*   (Endpoints for Create, List, Get Conference; List/Get Participants; Mute/Unmute, Kick Participant; Record, Play actions. Telephony actions are stubbed in service but have API endpoints. Statuses can be updated by FreeSWITCH events.)

### Application Management
Relative Path: `/Applications`
*   (Endpoints for Create, List, Get, Update, Delete Applications. Applications define `VoiceUrl` for TwiML control.)

### Voice Control (for FreeSWITCH to call into)
Base Path: `/api/v1/voice`
*   **POST/GET `/control`**: Generic endpoint for FreeSWITCH to fetch TwiML instructions.
    *   Receives parameters like `CallSid`, `AccountSid`, `Digits` from FreeSWITCH.
    *   Responds with TwiML XML (e.g., `<Say>`, `<Gather>`, `<Hangup>`).

### General
*   **GET `/health`**: Health check (DB status).

## Development Notes
*   **Testing**: All automated unit and integration tests were skipped due to tooling/environment issues. This is a critical gap for production readiness.
*   **Authentication**: `X-Auth-User-Sid` header is a placeholder and **NOT SECURE**. Replace with robust auth (JWT, OAuth2).
*   **FreeSWITCH Integration**:
    *   **Call Origination**: Uses `bgapi originate`. Gateway selection is based on Account settings (`DefaultOutboundGateway`, `GatewaySelectionScript`) or `FS_DEFAULT_GATEWAY_NAME`.
    *   **TwiML Control**: Calls handled by an `ApplicationSid` will have FreeSWITCH make HTTP requests (via a Lua script pattern detailed in `FREESWITCH_GUIDE.md`) to Agbara-Go's `/api/v1/voice/control` endpoint to fetch TwiML.
    *   **Event Handling**: A persistent ESL connection listens for FreeSWITCH events to update call/conference states in real-time. See `FREESWITCH_GUIDE.md` for event correlation tips.
    *   **Conference Telephony Actions**: Mute/Kick/Record/Play for conferences are stubbed in the service layer (DB records might be updated, but no actual FS command is sent for these yet).

```
