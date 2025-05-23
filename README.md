# Agbara-Go - Call Module & Account Management

This project is a Go (Golang) implementation of the Call and Account Management module functionalities, originally part of the AgbaraVOIP system. It provides a RESTful API for managing voice calls and user accounts.

This version uses PostgreSQL as its database.

## Features Implemented
*   **Call Management:**
    *   Make a new call (creates a call record).
    *   List calls for an account.
    *   Modify an existing call (currently supports updating `AnswerUrl`).
*   **Account Management:**
    *   Create Master Accounts.
    *   Create Sub-Accounts.
    *   List Sub-Accounts for an authenticated account.
    *   Get details for an authenticated account.
    *   Modify status of an authenticated account.
    *   Regenerate AuthToken for an authenticated account.


## Prerequisites
*   Go (version 1.19 or higher recommended)
*   PostgreSQL database server

## Building the Application

To build the application, navigate to the root directory of the project (`agbara-go`) and run:

```bash
go build -o agbara-server ./cmd/server/main.go
```
This will create an executable named `agbara-server` (or `agbara-server.exe` on Windows).

## Configuration

The application is configured using environment variables:

*   `AGBARA_DB_DSN`: The Data Source Name for connecting to your PostgreSQL database.
    *   Example: `postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable`
    *   If not set, the application will attempt to use a default DSN: `postgres://user:password@localhost:5432/agbaradb?sslmode=disable` (a warning will be logged).
*   `HTTP_PORT`: The port on which the HTTP server will listen.
    *   Defaults to `8080` if not set.
*   `X-Auth-User-Sid` (HTTP Header): For simulated authentication in the Account Management endpoints that require user context (e.g., listing sub-accounts, getting own account). This is a placeholder and **not for production use.**

## Database Setup

The required database schemas are provided in the `db/schema/` directory:
*   `001_create_calls_table.sql`
*   `002_create_accounts_table.sql`

To apply these schemas to your PostgreSQL database:
1.  Ensure you have access to your PostgreSQL server via `psql` or another client.
2.  Create your database if it doesn't exist (e.g., `CREATE DATABASE agbaradb;`).
3.  Run the schema files against your database in order:
    ```bash
    psql -U youruser -d yourdatabase -f db/schema/001_create_calls_table.sql
    psql -U youruser -d yourdatabase -f db/schema/002_create_accounts_table.sql
    ```
    Replace `youruser` and `yourdatabase` with your actual PostgreSQL username and database name.

## Running the Application

1.  **Set Environment Variables**:
    ```bash
    export AGBARA_DB_DSN="postgres://youruser:yourpassword@localhost:5432/yourdatabase?sslmode=disable"
    export HTTP_PORT="8080" # Optional, defaults to 8080
    ```
2.  **Run the Executable**:
    If you built the application as `agbara-server`:
    ```bash
    ./agbara-server
    ```
    The server will start, and you should see log messages indicating it's connected to the database and listening on the configured port. For Account Management endpoints that require user context, remember to pass the `X-Auth-User-Sid` header.

## API Endpoints

The following API endpoints are available, grouped under the base path `/api/v1`.

### Call Management
*   **GET `/api/v1/Accounts/{AccountSid}/Calls`**: Lists all calls for the specified `AccountSid`.
*   **POST `/api/v1/Accounts/{AccountSid}/Calls/Call`**: Creates a new call record.
    *   Request Body (JSON): `models.CallRequest`
    *   Response: The created `models.Call` object.
*   **POST `/api/v1/Accounts/{AccountSid}/Calls/{CallSid}`**: Modifies an existing call.
    *   Request Body (JSON): `models.CallRequest`
    *   Response: The updated `models.Call` object.

### Account Management
Authentication for user-specific account actions is simulated via the `X-Auth-User-Sid` HTTP header.

*   **POST `/api/v1/Accounts/Master`**: Creates a new master account.
    *   Request Body (JSON): `{"friendlyName": "My Master Account"}`
    *   Response: The created `models.Account` object.
*   **GET `/api/v1/Accounts`**: Lists sub-accounts for the account specified in `X-Auth-User-Sid`.
    *   Requires `X-Auth-User-Sid` header.
    *   Response: Array of `models.Account` objects.
*   **POST `/api/v1/Accounts`**: Creates a new sub-account under the account specified in `X-Auth-User-Sid`.
    *   Requires `X-Auth-User-Sid` header.
    *   Request Body (JSON): `{"friendlyName": "My Sub Account"}`
    *   Response: The created `models.Account` object.
*   **GET `/api/v1/Accounts/{accountSid}`**: Retrieves details for the specified `accountSid`.
    *   Requires `X-Auth-User-Sid` header, which must match `accountSid` in the path.
    *   Response: The `models.Account` object.
*   **POST `/api/v1/Accounts/{accountSid}`**: Modifies the status of the specified `accountSid`.
    *   Requires `X-Auth-User-Sid` header, which must match `accountSid` in the path.
    *   Request Body (JSON): `{"status": "suspended"}` (see `models.AccountStatus` for valid values)
    *   Response: The updated `models.Account` object.
*   **POST `/api/v1/Accounts/{accountSid}/AuthToken`**: Regenerates and returns a new authentication token for the specified `accountSid`.
    *   Requires `X-Auth-User-Sid` header, which must match `accountSid` in the path.
    *   Response: `{"accountSid": "...", "authToken": "new_plain_text_token"}`. **This token should be stored securely by the client as it will not be shown again.**

### General
*   **GET `/health`**: A health check endpoint for the service.
    *   Returns status of the application and database connectivity.

## Development Notes

*   **Testing**: Unit and integration tests for this module were skipped during this development phase due to persistent environmental issues encountered by the automated tooling with Go module dependency management (`go get` failing to find `go.mod`). This is a known gap and should be addressed in future iterations.
*   **Authentication**: The current authentication mechanism using the `X-Auth-User-Sid` header is a placeholder for development and testing purposes only. **It is not secure and should be replaced with a robust authentication system (e.g., JWT, OAuth2) in a production environment.**
