# AgbaraVOIP Reimplementation: GoLang & PostgreSQL - Technical Implementation Plan

## 1. Introduction
    - Purpose of this document: To outline the technical plan for re-implementing the AgbaraVOIP platform using GoLang for backend services and PostgreSQL for the database.
    - Brief overview of the goal: Port the existing C#/MongoDB AgbaraVOIP system to a GoLang/PostgreSQL stack to leverage GoLang's performance for concurrent operations, its suitability for network services, and PostgreSQL's robustness as a relational database.
    - Summary of key benefits: Improved performance and scalability, potentially lower resource consumption, modernized technology stack, strong typing and concurrency features of GoLang, and the reliability and data integrity features of PostgreSQL.

## 2. Analysis of Existing System & Core Functionality to Reimplement
    - This plan is based on the analysis of the `TECHNICAL_DOCUMENTATION.md` for the original AgbaraVOIP system.
    - **Core Entities to Reimplement:**
        - Account: User accounts, sub-accounts, authentication tokens, status, type.
        - Application: User-defined voice/SMS applications defining URLs for call/message handling logic.
        - Call: Call Detail Records (CDRs), status, direction, duration, price, linkage to Accounts.
        - Conference: Details of conference rooms, status, and participants.
        - FSServer (Freeswitch Server): Configuration for Freeswitch instances.
        - Gateway: Configuration for outbound VOIP gateways.
        - Recording: Metadata for call and conference recordings (URL, duration).
        - SMSMessage: Details of SMS messages (sender, recipient, body, status, price).
        - ConferenceParticipant: Information about participants in a conference.
    - **Key API Functionalities to Reimplement:**
        - Account Management: CRUD operations for accounts and sub-accounts.
        - Application Management: CRUD operations for voice/SMS applications.
        - Call Control: Initiation of outbound calls, retrieval of call history/details, in-call modifications (hangup, play audio, speak text, send DTMF, record).
        - Conference Management: Creation of conferences, management of participants (add, remove, mute, kick), conference recording.
        - Recording Management: Listing and retrieval of recordings.
        - SMS Handling: Sending and receiving SMS messages.
    - **AgbaraXML Core Logic to Address:**
        - The new system must provide equivalent functionality for core AgbaraXML verbs such as `<Say>`, `<Play>`, `<Gather>`, `<Record>`, `<Dial>` (including nested `<Number>` and `<Conference>`), `<Hangup>`, `<Pause>`, `<Redirect>`, `<Reject>`, and `<PreAnswer>`. The method of achieving this (direct XML parsing or Go-native logic) is discussed in the architecture section.
    - **Freeswitch Interaction Points:**
        - **Inbound ESL (Go App to Freeswitch):** For originating calls and sending commands to active calls via API.
        - **Outbound ESL (Freeswitch to Go App):** For Freeswitch to delegate call control to the Go application (e.g., for incoming calls or calls requiring dynamic XML-like logic).
        - **Event Handling:** Robust handling of Freeswitch events (e.g., `CHANNEL_EXECUTE_COMPLETE`, `CHANNEL_HANGUP_COMPLETE`, `DTMF`) is critical.

## 3. GoLang Service Architecture
    - **Proposed Structure:** A modular monolithic application for the initial reimplementation. This application will be internally structured into distinct GoLang packages. This structure can be evolved into microservices if future scalability requirements dictate.
    - **Key GoLang Packages/Modules:**
        - **`api` Package:**
            - Handles all incoming HTTP REST API requests.
            - Recommended Framework: **Gin Gonic (Gin)** (`github.com/gin-gonic/gin`) for routing, request/response marshalling, and middleware.
            - Interacts with the `services` package for business logic and the `callcontrol` package for real-time call operations.
        - **`callcontrol` Package:**
            - Manages real-time call flow logic and all Freeswitch ESL interactions.
            - ESL Library: **`github.com/fiorix/go-eventsocket` (fsesl)** is a strong candidate. If it proves unsuitable, a custom ESL interaction module might be needed, built upon `net` and Go's concurrency primitives.
            - Outbound ESL Server: Listens for TCP connections from Freeswitch (via the `socket` dialplan application). Each connection is handled in a separate goroutine.
            - Inbound ESL Client: Provides an interface for the `api` package to send commands to Freeswitch (e.g., originate calls). May include connection pooling.
            - AgbaraXML Equivalence:
                - **Initial Approach (Option A):** Retain the ability to process AgbaraXML-like documents. The Go service will fetch XML from URLs (defined in `Application` entities) and parse it using `encoding/xml`. Go functions corresponding to each AgbaraXML verb will execute ESL commands. This ensures closer functional parity with the original system.
                - **Future Evolution (Option B):** For performance-critical or complex flows, logic could be implemented directly in Go, triggered by application identifiers rather than dynamic XML fetching.
        - **`services` Package (or `domain`):**
            - Encapsulates business logic and data persistence operations, abstracting database interactions.
            - Database Interaction: Use the standard `database/sql` package with the PostgreSQL driver `github.com/lib/pq`.
            - ORM/Query Builder: **GORM** (`gorm.io/gorm`) is recommended for simplifying database operations, struct-to-table mapping, and managing schema migrations. Alternatively, `github.com/jmoiron/sqlx` can be used for a lighter-weight abstraction over `database/sql`.
            - Defines service interfaces (e.g., `AccountService`, `CallService`) and their implementations. Go structs will represent database entities.
        - **`utils` Package (or `common`):**
            - Contains shared utility functions (e.g., time converters, string manipulation), custom error types, and application-wide constants (using Go's typed constants or enums where appropriate).
    - **Concurrency Model:**
        - API requests handled by Gin will typically run in separate goroutines.
        - Each Freeswitch ESL connection (especially in outbound mode) will be managed in its own goroutine.
        - Go channels will be used for safe inter-goroutine communication, particularly for dispatching ESL events or managing results from asynchronous operations.
    - **Main Application (`cmd/<appname>/main.go`):**
        - Initializes global configurations (e.g., using Viper from YAML/JSON files or environment variables).
        - Sets up structured logging (e.g., Logrus or Zap).
        - Establishes and manages database connection pools.
        - Initializes and starts ESL client/server components.
        - Sets up and runs the Gin HTTP server.
        - Implements graceful shutdown mechanisms to release resources properly.

## 4. PostgreSQL Database Schema
    - **General Conventions:**
        - Table and column names: `snake_case`.
        - Primary Keys: Each table will have an `id BIGSERIAL PRIMARY KEY`.
        - External Identifiers: The existing `sid VARCHAR(64) UNIQUE NOT NULL` concept (e.g., "ACxxxxx") will be retained as a unique, indexed column for external references.
        - Audit Timestamps: `created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC')` and `updated_at TIMESTAMP WITHOUT TIME ZONE DEFAULT (NOW() AT TIME ZONE 'UTC')` (the latter updated by a trigger).
    - **Table Definitions:**
        - **`accounts`**: `id`, `sid`, `parent_sid` (self-referential FK), `friendly_name`, `phone_number`, `auth_token` (to be hashed), `type` (ENUM: 'trial', 'full'), `status` (ENUM: 'active', 'suspended', 'closed'), `created_at`, `updated_at`. Indexes on `sid`, `parent_sid`, `auth_token`.
        - **`applications`**: `id`, `sid`, `account_sid` (FK to `accounts.sid`), `friendly_name`, `voice_url`, `voice_method`, `voice_fallback_url`, `voice_fallback_method`, `status_callback_url`, `status_callback_method`, `sms_url`, `sms_method`, `sms_fallback_url`, `sms_fallback_method`, `sms_status_callback_url`, `sms_status_callback_method`, `heartbeat_url`, `created_at`, `updated_at`. Indexes on `sid`, `account_sid`.
        - **`calls`**: `id`, `sid`, `account_sid` (FK), `caller_id`, `callee_id`, `application_sid` (FK to `applications.sid`, nullable), `answer_url`, `status` (ENUM: 'queued', 'ringing', 'in-progress', 'completed', 'failed', 'busy', 'no-answer'), `direction` (ENUM: 'inbound', 'outbound-api', 'outbound-dial'), `duration_seconds`, `price` (NUMERIC), `answered_by`, `timeout_seconds`, `start_time`, `answer_time`, `end_time`, `hangup_cause` (VARCHAR), `created_at`, `updated_at`. Indexes on `sid`, `account_sid`, `status`, `direction`, `start_time`.
        - **`freeswitch_servers`**: `id`, `sid`, `host`, `port`, `password` (encrypted), `outbound_address`, `is_active`, `created_at`, `updated_at`. Index on `sid`.
        - **`gateways`**: `id`, `sid`, `freeswitch_server_sid` (FK, nullable), `gateway_string`, `codecs`, `retry_count`, `timeout_seconds`, `routes` (TEXT or JSONB), `friendly_name`, `created_at`, `updated_at`. Index on `sid`.
        - **`recordings`**: `id`, `sid`, `account_sid` (FK), `call_sid` (FK, nullable), `conference_sid` (FK to `conferences.sid`, nullable), `duration_seconds`, `file_path` (URL or path), `format`, `size_bytes`, `created_at`, `updated_at`. Indexes on `sid`, `account_sid`, `call_sid`, `conference_sid`.
        - **`sms_messages`**: `id`, `sid`, `account_sid` (FK), `from_number`, `to_number`, `body` (TEXT), `status` (ENUM: 'queued', 'sent', 'failed', 'delivered', 'undelivered'), `direction` (ENUM), `price` (NUMERIC), `error_code`, `error_message`, `sent_at`, `created_at`, `updated_at`. Indexes on `sid`, `account_sid`, `status`.
        - **`conferences`**: `id`, `sid`, `account_sid` (FK), `friendly_name`, `status` (ENUM: 'init', 'in-progress', 'completed'), `start_time`, `end_time`, `created_at`, `updated_at`. Indexes on `sid`, `account_sid`.
        - **`conference_participants`**: `id`, `sid`, `conference_sid` (FK), `call_sid` (FK, unique), `account_sid` (FK), `is_muted`, `is_moderator`, `join_time`, `leave_time`. Indexes on `sid`, `conference_sid`, `call_sid`.
    - **Database Migrations:** Use a tool like `golang-migrate/migrate` (`github.com/golang-migrate/migrate`) to manage schema evolution through versioned SQL migration files.

## 5. Freeswitch Interaction in GoLang
    - **ESL Library/Package:** Primary choice: `github.com/fiorix/go-eventsocket` (fsesl). If limitations are found, a custom wrapper or minimal library focusing on core needs might be developed.
    - **Inbound ESL Mode (Go App to Freeswitch):**
        - The `callcontrol` or `api` service will manage a connection (or pool of connections) to Freeswitch.
        - API calls requiring Freeswitch actions (e.g., originate call) will translate to ESL commands (`originate`, `uuid_bridge`, `uuid_playback`, `uuid_kill`) sent over this connection.
        - Synchronous responses will be handled to confirm command execution.
    - **Outbound ESL Mode (Freeswitch to Go App):**
        - The `callcontrol` service will run a TCP server (e.g., using `net.Listen`) on a configured port.
        - Each incoming Freeswitch ESL connection (representing a call leg) will be handled in a dedicated goroutine.
        - Initial handshake: Send `connect` ESL command, process response to get channel variables (UUID, Answer URL, Account SID, etc.). Subscribe to necessary events (`CHANNEL_HANGUP_COMPLETE`, `CHANNEL_EXECUTE_COMPLETE`, `DTMF`, relevant `CUSTOM` events).
    - **Call Control Logic (AgbaraXML Equivalence):**
        - **Initial Phase (Option A):** The Go `callcontrol` service will fetch an XML document from the `answer_url`. It will parse this XML using `encoding/xml`. For each AgbaraXML verb, a corresponding Go function will execute the necessary ESL commands via the active ESL connection for that call. `<Redirect>` will trigger fetching a new XML document.
    - **Event Handling:**
        - The chosen ESL library (or custom implementation) must provide a way to receive and parse all asynchronous events from Freeswitch.
        - Events will be dispatched (likely via channels) to the goroutine managing the specific call session (identified by Freeswitch Channel UUID).
        - Specific handlers for `CHANNEL_HANGUP_COMPLETE`, `CHANNEL_EXECUTE_COMPLETE`, `DTMF`, etc., will update call state and drive the logic flow (e.g., proceed to next XML verb after playback completion).
    - **State Management for Active Calls:** State for each call controlled via outbound ESL (e.g., current XML processing state, collected digits) will be managed within its handling goroutine, potentially using structs. For distributed setups or resilience, this state might need caching in Redis or similar.

## 6. API Endpoint Plan
    - **General Principles:** Stateless, JSON request/response, authentication on protected routes, versioned (e.g., `/v1`).
    - **Endpoints (Grouped by Resource):**
        - **Account Management:** `POST /v1/accounts`, `POST /v1/accounts/{account_sid}/subaccounts`, `GET /v1/accounts/{account_sid}`, `GET /v1/accounts/{account_sid}/subaccounts`, `PUT /v1/accounts/{account_sid}`.
        - **Application Management:** `POST /v1/accounts/{account_sid}/applications`, `GET /v1/accounts/{account_sid}/applications/{app_sid}`, `GET /v1/accounts/{account_sid}/applications`, `PUT /v1/accounts/{account_sid}/applications/{app_sid}`, `DELETE /v1/accounts/{account_sid}/applications/{app_sid}`.
        - **Call Management:** `POST /v1/accounts/{account_sid}/calls` (originate), `GET /v1/accounts/{account_sid}/calls/{call_sid}`, `GET /v1/accounts/{account_sid}/calls` (list with filters), `PUT /v1/accounts/{account_sid}/calls/{call_sid}` (e.g., hangup/redirect).
        - **In-Call Control (Sub-resources):** `POST .../calls/{call_sid}/play`, `POST .../calls/{call_sid}/say`, `POST .../calls/{call_sid}/dtmf`, `POST .../calls/{call_sid}/record/start`, `POST .../calls/{call_sid}/record/stop`.
        - **Conference Management:** `GET .../conferences`, `GET .../conferences/{conf_sid}`. Participants: `GET .../{conf_sid}/participants`, `GET .../{conf_sid}/participants/{participant_call_sid}`, `PUT .../{conf_sid}/participants/{participant_call_sid}` (mute/kick), `POST .../{conf_sid}/participants` (add via dial-out). Conference Actions: `POST .../{conf_sid}/play`, `POST .../{conf_sid}/say`, `POST .../{conf_sid}/record/start`, `POST .../{conf_sid}/record/stop`.
        - **Recording Management:** `GET .../recordings`, `GET .../recordings/{rec_sid}`, `DELETE .../recordings/{rec_sid}`.
        - **SMS Management:** `POST .../sms/messages` (send), `GET .../sms/messages/{sms_sid}`, `GET .../sms/messages` (list).
        - **(Potential Admin Endpoints):** For Freeswitch Server & Gateway CRUD.

## 7. Authentication and Authorization Strategy
    - **Authentication:**
        - **Initial:** HTTP Basic Authentication. Middleware in Gin to parse `Authorization: Basic <base64(account_sid:auth_token)>` and validate against `accounts` table.
        - **Future Enhancement:** JWT-based authentication. Add `POST /v1/auth/token` endpoint. API expects `Authorization: Bearer <jwt>`.
    - **Authorization:**
        - **Ownership-Based:** Primarily based on the authenticated `account_sid` extracted from the token/credentials. API handlers and service layer functions will ensure users can only access/modify their own resources. Database queries will be scoped by `account_sid`.
        - **Parent/Subaccount:** Master accounts can manage their subaccounts. Subaccounts operate within their own scope.
        - **(Future) RBAC:** For administrative roles, a more detailed RBAC system might be needed (Users, Roles, Permissions).
    - **Implementation:** Gin middleware for authentication. Authorization checks within service logic and database queries.

## 8. Deployment and Configuration
    - **Deployment Strategy:**
        - **Containerization:** GoLang application (statically linked binary in a minimal Docker image like Alpine or Scratch), PostgreSQL (official image), and Freeswitch (official or community image) will all be containerized using Docker.
        - **Orchestration:**
            - **Development/Testing/Simple Production:** `docker-compose.yml` for managing the multi-container setup.
            - **Advanced Production:** Kubernetes for scalability, high availability, and automated management (Deployments, StatefulSets for DB, Services, ConfigMaps, Secrets, PersistentVolumeClaims).
    - **Configuration Management:**
        - **Primary:** Environment variables (following 12-factor app principles).
        - **Secondary/Defaults:** Configuration files (e.g., `config.yml` or `config.json`) loaded by the Go application using a library like **Viper** (`github.com/spf13/viper`). Environment variables will override file values.
    - **Key Configuration Parameters:** API port, PostgreSQL connection details (host, port, user, pass, dbname, pool settings), Freeswitch ESL connection details (inbound & outbound), log level, JWT secrets (if used).
    - **Database Deployment:** Use Docker volumes or Kubernetes PersistentVolumes for PostgreSQL data persistence. Implement regular backup strategy. Use `golang-migrate/migrate` for schema migrations, applied during deployment.

## 9. Phased Implementation Roadmap
    - **Phase 0: Setup and Core Foundation:** Project structure, Docker setup, basic config/logging, initial DB schema (accounts, applications) & migrations, basic ESL connectivity tests.
    - **Phase 1: Account Management & Authentication:** API for Account CRUD, Basic Auth implementation.
    - **Phase 2: Application Management & Basic Call Origination:** API for Application CRUD, API to originate calls (Freeswitch fetches static XML via `answer_url`), basic CDR logging.
    - **Phase 3: Core AgbaraXML-like Processing (Outbound ESL):** Go ESL server, fetch & parse XML, implement `<Say>`, `<Play>`, `<Hangup>`, `<Pause>`, `<Redirect>`.
    - **Phase 4: Advanced Call Control Features:** Implement `<Gather>`, `<Record>`, basic `<Dial>` (single number). `recordings` table.
    - **Phase 5: Conference Calls & Complex Dial:** Implement `<Conference>` verb, enhance `<Dial>` for conferences. Conference APIs. `conferences` & `conference_participants` tables.
    - **Phase 6: SMS Functionality:** SMS APIs, interaction with SMS gateway (mock or real), `sms_messages` table.
    - **Phase 7: Advanced Features, Security Hardening, Scalability:** Remaining features, JWT auth, admin APIs, security review, performance testing, potential microservice refactoring. Monitoring/alerting setup.
    - **Phase 8: Documentation & Production Readiness:** Finalize user/API docs, internal tech docs, CI/CD, UAT.

## 10. Future Considerations / Out of Scope for Initial Reimplementation
    - Advanced Role-Based Access Control (RBAC) for multi-tenant administration.
    - Real-time dashboards and analytics.
    - WebRTC integration for browser-based clients.
    - High-availability setup for Freeswitch itself (if not already in place).
    - Internationalization and localization for API responses and TTS/ASR.
    - Billing and payment integration.
