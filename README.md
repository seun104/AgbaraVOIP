# Agbara-Go: Voice & Communication API

Agbara-Go is a Golang-based RESTful API providing functionalities for managing voice calls, user accounts, multi-participant conferences, and application configurations for voice/SMS webhooks. This project is inspired by aspects of a VOIP API system and uses PostgreSQL as its database.

**Modules Implemented:**
*   **Call Management**: Create, list, and modify voice call records.
*   **Account Management**: Manage master and sub-accounts, including authentication tokens.
*   **Conference Management**: Create conferences, manage participants (list, get, mute, kick - with stubbed telephony actions), and conference-level actions like record/play (stubbed).
*   **Application Management**: Configure applications with webhook URLs for voice and SMS handling.

## Prerequisites
*   Go (version 1.19 or higher recommended)
*   PostgreSQL database server (version 12 or higher recommended)

## Building the Application

Navigate to the root directory of the project (`agbara-go`) and run:
```bash
go build -o agbara-server ./cmd/server/main.go
```
This creates an executable named `agbara-server` (or `agbara-server.exe` on Windows).

## Configuration

The application is configured using environment variables:

*   `AGBARA_DB_DSN`: The Data Source Name for PostgreSQL.
    *   Example: `postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable`
    *   Default (if not set, logged with a warning): `postgres://user:password@localhost:5432/agbaradb?sslmode=disable`
*   `HTTP_PORT`: Port for the HTTP server.
    *   Default: `8080`
*   `X-Auth-User-Sid` (HTTP Header): **Placeholder for authentication.** Used by most API endpoints to identify the authenticated user/account. **This is NOT for production use.**

## Database Setup

Database schemas are in the `db/schema/` directory. Apply them in order:

1.  `001_create_calls_table.sql`: Defines the `calls` table and a shared `update_modified_column` trigger function.
2.  `002_create_accounts_table.sql`: Defines the `accounts` table.
3.  `003_create_conference_tables.sql`: Defines `conferences` and `participants` tables.
4.  `004_create_applications_table.sql`: Defines the `applications` table.

**Example `psql` commands:**
```bash
# Connect to your PostgreSQL instance and create the database if it doesn't exist
# PGPASSWORD=yourpassword psql -U youruser -h localhost -c "CREATE DATABASE yourdatabase;"

PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/001_create_calls_table.sql
PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/002_create_accounts_table.sql
PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/003_create_conference_tables.sql
PGPASSWORD=yourpassword psql -U youruser -h localhost -d yourdatabase -f db/schema/004_create_applications_table.sql
```
Replace `youruser`, `yourpassword`, `localhost`, and `yourdatabase` with your actual PostgreSQL details.

## Running the Application

1.  **Set Environment Variables** (example for bash/zsh):
    ```bash
    export AGBARA_DB_DSN="postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable"
    export HTTP_PORT="8080" # Optional
    ```
2.  **Run the Executable**:
    ```bash
    ./agbara-server
    ```
    The server will log its startup, database connection status, and listening port. For most API calls, include the `X-Auth-User-Sid` header with a valid Account SID.

## API Endpoints

All API endpoints are prefixed with `/api/v1`.
Authentication for user-specific actions is simulated via the `X-Auth-User-Sid` HTTP header.

### Account Management
Base Path: `/api/v1/Accounts`

*   **POST `/Master`**: Creates a new master account.
    *   Request: `{"friendlyName": "My Master Account"}`
    *   Response: `models.Account`
*   **GET `` (relative to `/api/v1/Accounts`)**: Lists sub-accounts for the account in `X-Auth-User-Sid`.
    *   Requires: `X-Auth-User-Sid` header.
    *   Response: `[]models.Account`
*   **POST `` (relative to `/api/v1/Accounts`)**: Creates a sub-account under `X-Auth-User-Sid`.
    *   Requires: `X-Auth-User-Sid` header.
    *   Request: `{"friendlyName": "My Sub Account"}`
    *   Response: `models.Account`
*   **GET `/{accountSid}`**: Retrieves details for `accountSid`.
    *   Requires: `X-Auth-User-Sid` header (must match `accountSid`).
    *   Response: `models.Account`
*   **POST `/{accountSid}`**: Modifies status of `accountSid`.
    *   Requires: `X-Auth-User-Sid` header (must match `accountSid`).
    *   Request: `{"status": "suspended"}` (see `models.AccountStatus`)
    *   Response: `models.Account`
*   **POST `/{accountSid}/AuthToken`**: Regenerates AuthToken for `accountSid`.
    *   Requires: `X-Auth-User-Sid` header (must match `accountSid`).
    *   Response: `{"accountSid": "...", "authToken": "new_plain_text_token"}`

---
Base Path for Calls, Conferences, Applications: `/api/v1/Accounts/{accountSidInPath}`
(Requires `X-Auth-User-Sid` header matching `{accountSidInPath}`)

### Call Management
Relative Path: `/Calls` (i.e., `/api/v1/Accounts/{accountSidInPath}/Calls`)

*   **GET ``**: Lists calls for `{accountSidInPath}`.
    *   Response: `[]models.Call`
*   **POST `/Call`**: Creates a call record for `{accountSidInPath}`.
    *   Request: `models.CallRequest` (e.g., `{"to": "number", "answerUrl": "url"}`)
    *   Response: `models.Call`
*   **POST `/{callSid}`**: Modifies call `callSid`.
    *   Request: `models.CallRequest` (e.g., `{"answerUrl": "new_url"}`)
    *   Response: `models.Call`

### Conference Management
Relative Path: `/Conferences` (i.e., `/api/v1/Accounts/{accountSidInPath}/Conferences`)

*   **POST ``**: Creates a conference for `{accountSidInPath}`.
    *   Request: `models.CreateConferenceRequest` (e.g., `{"friendlyName": "My Conf"}`)
    *   Response: `models.Conference`
*   **GET ``**: Lists conferences for `{accountSidInPath}`.
    *   Response: `[]models.Conference`
*   **GET `/{conferenceSid}`**: Gets details for `conferenceSid`.
    *   Response: `models.Conference`
*   **GET `/{conferenceSid}/Participants`**: Lists participants in `conferenceSid`.
    *   Response: `[]models.Participant`
*   **GET `/{conferenceSid}/Participants/{callSid}`**: Gets participant `callSid` in `conferenceSid`.
    *   Response: `models.Participant`
*   **POST `/{conferenceSid}/Participants/{callSid}`**: Mute/unmute participant `callSid`.
    *   Request: `models.MuteParticipantRequest` (e.g., `{"muted": true}`)
    *   Response: Updated `models.Participant` (DB record updated, telephony action stubbed)
*   **DELETE `/{conferenceSid}/Participants/{callSid}`**: Kicks participant `callSid`.
    *   Response: `204 No Content` (DB record deleted, telephony action stubbed)
*   **POST `/{conferenceSid}/Record`**: Start recording (stubbed).
    *   Request: `models.ConferenceRecordRequest` (optional)
    *   Response: `models.ConferenceActionResponse`
*   **DELETE `/{conferenceSid}/Record`**: Stop recording (stubbed).
    *   Response: `models.ConferenceActionResponse`
*   **POST `/{conferenceSid}/Play`**: Play audio to conference (stubbed).
    *   Request: `models.ConferencePlayRequest` (e.g., `{"url": "audio_url"}`)
    *   Response: `models.ConferenceActionResponse`
*   **DELETE `/{conferenceSid}/Play`**: Stop audio (stubbed).
    *   Response: `models.ConferenceActionResponse`

### Application Management
Relative Path: `/Applications` (i.e., `/api/v1/Accounts/{accountSidInPath}/Applications`)

*   **POST ``**: Creates an application for `{accountSidInPath}`.
    *   Request: `models.ApplicationRequest` (see model for fields)
    *   Response: `models.Application`
*   **GET ``**: Lists applications for `{accountSidInPath}`.
    *   Response: `[]models.Application`
*   **GET `/{applicationSid}`**: Gets details for `applicationSid`.
    *   Response: `models.Application`
*   **POST `/{applicationSid}`**: Updates `applicationSid`.
    *   Request: `models.ApplicationRequest`
    *   Response: `models.Application`
*   **DELETE `/{applicationSid}`**: Deletes `applicationSid`.
    *   Response: `204 No Content`

### General
*   **GET `/health`**: Health check endpoint.

## Development Notes

*   **Testing**: Unit and integration tests for all modules were skipped during this development phase due to persistent environmental issues encountered by the automated tooling with Go module dependency management. This is a known gap and should be addressed in future iterations.
*   **Authentication**: The current authentication mechanism using the `X-Auth-User-Sid` header is a placeholder. **It is not secure and must be replaced with a robust authentication system (e.g., JWT, OAuth2) in a production environment.**
*   **Telephony Actions**: For Conference Management, actions like Mute, Kick, Record, and Play currently update database records where applicable but their actual interaction with a telephony server (e.g., FreeSWITCH) is stubbed. Full implementation would require an ESL client or similar.
