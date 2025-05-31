## AgbaraAPI

**Purpose:**
AgbaraAPI is the primary web API for the AgbaraVOIP system. It is built using the NancyFX framework and is responsible for handling incoming HTTP requests and exposing the core functionalities of the platform.

**Hosting:**
The API is self-hosted using `Nancy.Hosting.Self`. By default, it listens on `http://127.0.0.1:8082`. The main entry point for the self-hosted server is defined in `src/AgbaraAPI/Program.cs`.

**Core Logic (`src/AgbaraAPI/Core/ApiServer.cs`):**
The `ApiServer` class encapsulates a significant portion of the API's business logic. Key responsibilities include:
- Interacting with the Freeswitch component (via `FSInbound`) for telephony operations.
- Coordinating with various services from the `Domain` layer (e.g., `ICallService`, `IFSService`).
- Handling operations such as:
    - Initiating outbound calls.
    - Playing audio files or synthesized speech to calls.
    - Recording calls.
    - Managing conference calls and participants.

**Modules (`src/AgbaraAPI/Modules/`):**
The API is organized into several modules, each responsible for a specific set of functionalities. NancyFX modules define routes and handlers for different API endpoints. Key modules include:

-   **`AccountModule.cs`**: Manages user accounts and sub-accounts, including creation, retrieval, and status modification.
-   **`ApplicationModule.cs`**: Handles the configuration and management of voice applications.
-   **`CallModule.cs`**: Provides endpoints for call control operations, such as:
    - Making outbound calls.
    - Retrieving call history and details.
    - Playing audio or speaking text to an active call.
    - Recording active calls.
    - Sending DTMF tones (digits) during a call.
    - Hanging up calls.
-   **`ConferenceModule.cs`**: Manages multi-party conference calls, including creating conferences, adding/removing participants, muting/unmuting, and recording conferences.
-   **`RecordingModule.cs`**: Provides access to and management of call and conference recordings.
-   **`SMSModule.cs`**: Handles sending and receiving SMS messages.
-   **`DefaultModule.cs`**: Likely handles root or default routes.
-   **`TestModule.cs`**: May contain endpoints for testing purposes.

**Authentication (`src/AgbaraAPI/AuthenticationBootstrapper.cs`, `src/AgbaraAPI/Bootstrappers/UserValidator.cs`):**
-   Authentication is implemented using HTTP Basic Authentication.
-   The `AuthenticationBootstrapper` class configures the basic authentication pipeline.
-   An `IUserValidator` implementation (likely `UserValidator.cs`) is responsible for validating user credentials against a data store.
-   Once authenticated, the `Context.CurrentUser.UserName` (which typically holds the `AccountSid`) is used for authorization purposes within the modules, ensuring users can only access their own resources.

**Request/Response Handling:**
-   API modules can typically respond in both XML and JSON formats. The desired format is often selected by appending `.json` or `.xml` to the request URI, or by setting the `Accept` header.
-   NancyFX's model binding capabilities are used to automatically map incoming request data (query parameters, form data, or request body) to C# request objects defined in the `src/AgbaraAPI/Model/` subdirectories.

**Dependency Injection (`src/AgbaraAPI/Bootstrappers/NancyBootstrapper.cs`):**
-   The application utilizes NancyFX's built-in TinyIoC container for dependency injection.
-   The `NancyBootstrapper` class is responsible for registering dependencies. For example, it registers `IInboundDependency` (implemented by `InboundDependency`), which provides modules access to the central `ApiServer` instance.

**Key Files:**
-   `src/AgbaraAPI/Program.cs`: Main entry point for the self-hosted API.
-   `src/AgbaraAPI/Core/ApiServer.cs`: Contains core API logic and Freeswitch interaction.
-   `src/AgbaraAPI/Bootstrappers/NancyBootstrapper.cs`: Configures NancyFX, including dependency injection.
-   `src/AgbaraAPI/AuthenticationBootstrapper.cs`: Configures basic authentication.
-   `src/AgbaraAPI/Modules/`: Directory containing individual API modules.
-   `src/AgbaraAPI/Model/`: Directory containing request and response model classes.

## AgbaraConsole

**Purpose:**
`AgbaraConsole` is a console application that acts as the main launcher and host for the AgbaraVOIP backend services. It is not an interactive command-line interface for administrative tasks, but rather the primary process that initializes and runs the core components of the system.

**Key Responsibilities:**

1.  **Database Setup:**
    *   On startup, `AgbaraConsole` checks if the required database schema exists.
    *   The `Program.Main` method calls `IsDBOk()` (which in the current implementation seems to always trigger a setup) and then `SetupDB()`.
    *   The `SetupDB()` method utilizes `OrmBase` (likely from the `Domain.Services.SQL` namespace) to create or verify the necessary database tables and structures required by the application.

2.  **Service Initialization:**
    *   The primary function of `AgbaraConsole` is to start the essential backend services. This is handled by the `AgbaraVOIPService.Start()` method (defined in `src/AgbaraConsole/AgbaraVoip.cs`).
    *   The `AgbaraVOIPService` starts the following key components:
        *   **AgbaraAPI:** It initiates the self-hosted NancyFX web API by calling a method like `AgbaAPISelfHostingServer.Start()` (from the `Emmanuel.AgbaraVOIP.AgbaraAPI` namespace). This makes the API endpoints available for client applications.
        *   **AgbaraXML Server:** It starts the AgbaraXML processing server by calling a method like `AgbaMLServer.Start()` (from the `Emmanuel.AgbaraVOIP.AgbaraXML` namespace). This server is likely responsible for processing XML-based call control instructions, possibly interacting with Freeswitch or handling specific telephony logic defined in AgbaraXML.

**Logging:**
The application uses `log4net` for logging diagnostic information and errors. Configuration for log4net would typically be found in an `app.config` or a dedicated `log4net.config` file.

**Usage:**
To run the AgbaraVOIP system, `AgbaraConsole.exe` would be executed. It will perform the database checks and then launch the API and XML processing services, keeping them running until the console application is terminated.

**Key Files:**
-   `src/AgbaraConsole/Program.cs`: Contains the `Main` entry point, database setup logic.
-   `src/AgbaraConsole/AgbaraVoip.cs`: Defines the `AgbaraVOIPService` responsible for starting the core API and XML services.
-   `src/AgbaraConsole/app.config`: Likely contains application configuration, including database connection strings and log4net settings.

## AgbaraService

**Purpose:**
`AgbaraService` allows the AgbaraVOIP application to be deployed and run as a Windows Service. This enables the application to operate in the background, start automatically when the system boots, and be managed using standard Windows service management tools (e.g., `services.msc`).

**Framework:**
The project utilizes the **Topshelf** library, which simplifies the creation of Windows services in .NET. Topshelf handles the complexities of service installation, uninstallation, and lifetime management.

**Service Implementation (`src/AgbaraService/AgbaraVOIPService.cs`):**
-   The `AgbaraVOIPService` class contains the core logic for the Windows Service. It implements methods to start and stop the AgbaraVOIP application components.
-   **`Start()` method:**
    -   This method is invoked when the Windows Service starts.
    -   Similar to `AgbaraConsole`, it initializes and launches the main backend components:
        -   **AgbaraAPI:** It starts the AgbaraAPI. The code refers to `AgbaAPIWcfHostingServer.Start()`. This might indicate a WCF hosting mechanism for the API when run as a service, or it could be a different naming for the same self-hosted NancyFX server used in `AgbaraConsole`. The API is started asynchronously using `Task.Factory.StartNew()`.
        -   **AgbaraXML Server:** It starts the `AgbaMLServer` (from the `Emmanuel.AgbaraVOIP.AgbaraXML` namespace), also asynchronously. This component is responsible for processing XML-based call control logic.
-   **`Stop()` method:**
    -   This method is called when the Windows Service is stopped.
    -   The provided code shows an empty `Stop()` method. A complete implementation should include logic to gracefully shut down the AgbaraAPI and AgbaraXML server, releasing any resources they hold.

**Configuration and Management (`src/AgbaraService/Program.cs`):**
-   The `Program.Main` method uses `HostFactory.New()` from Topshelf to configure the service properties:
    -   **Service Name:** "AgbaraVOIPService"
    -   **Display Name:** "AgbaraVOIP Window Service"
    -   **Description:** "AgbaraVOIP Window Service"
    -   **Run Account:** Configured to run as `LocalSystem`.
-   Topshelf allows easy installation and uninstallation of the service from the command line (e.g., `AgbaraService.exe install`, `AgbaraService.exe uninstall`).

**Database Setup:**
-   The `Program.cs` file contains `IsDBOk()` and `SetupDB()` methods.
-   The `SetupDB()` method is intended to be called during service installation (`args[0] == "install"`) if `IsDBOk()` indicates the database is not set up.
-   However, in the provided code for `AgbaraService`, `IsDBOk()` always returns `true`, and `SetupDB()` is an empty method. This suggests that the database setup responsibility primarily lies with `AgbaraConsole` or is handled through a different mechanism when deploying as a service.

**Logging:**
-   `AgbaraService` uses `log4net` for logging.
-   The logging configuration is loaded from `src/AgbaraService/log4net.config`. The `XmlConfigurator.ConfigureAndWatch` method is used, allowing for logging configuration changes without restarting the service.

**Key Files:**
-   `src/AgbaraService/Program.cs`: Main entry point, uses Topshelf to configure and run the Windows Service.
-   `src/AgbaraService/AgbaraVOIP.cs`: Implements the `AgbaraVOIPService` class with `Start` and `Stop` methods for the service logic.
-   `src/AgbaraService/app.config`: Application configuration file.
-   `src/AgbaraService/log4net.config`: Configuration for `log4net` logging.

## AgbaraUtil (AgbaraCommon.csproj)

**Purpose:**
`AgbaraUtil` (project name `AgbaraCommon`) is a shared utility library providing common classes, helper functions, and constants used across various components of the AgbaraVOIP system.

**Key Features and Components:**

1.  **Error Handling and Custom Exceptions:**
    *   **`Error/Error.cs`:**
        *   `LimitExceededError`: Custom exception class, likely thrown when a predefined limit or quota (e.g., number of concurrent calls, API request rate) is exceeded.
        *   `ConnectError`: Custom exception class used to indicate failures during network connection attempts, such as when making HTTP requests to external services or failing to connect to a database or Freeswitch.
    *   **`Exception/Exceptions.cs`:**
        *   `RedirectException`: A specialized exception designed to signal an HTTP redirect. It encapsulates the target URL, HTTP method (defaulting to POST), and any parameters that need to be passed along with the redirect. This is typically caught by the web framework or API layer to issue a proper redirect response to the client.

2.  **Gateway Interaction (`Gateway/` directory):**
    *   **`DialString.cs`:**
        *   Defines the `DialString` class, which represents the set of parameters required to construct a Freeswitch dial string for initiating an outbound call.
        *   Properties include `callSid`, `to` (destination number), `gw` (gateway string), `codecs`, `timeout`, and `extra_dial_string` for additional Freeswitch channel variables or options.
    *   **`HttpRequest.cs`:**
        *   Provides the `HTTPRequest` utility class for making HTTP requests (GET, POST, PUT, DELETE) to external URLs.
        *   Includes functionality for:
            - Basic Authentication: Takes an ID and token for authentication.
            - Custom Headers: Sets a "X_agbara_SIGNATURE" header (derived from the auth credentials) and a "User-Agent" of "agbara".
            - Asynchronous Operations: Appears to use `AsyncCtpExtensions` for non-blocking HTTP calls.
            - Data Formatting: Handles URL encoding for query parameters and request bodies.
    *   **`Request.cs`:**
        *   Defines the `Request` class, which seems to represent an outbound call request within the system before it's processed by Freeswitch.
        *   It contains a `CallSid`, callback URLs for various call events (`answer_url`, `ring_url`, `hangup_url`), a state flag, and importantly, a queue of `DialString` objects. This queue allows the system to attempt the call through multiple gateways if the initial attempts fail.

3.  **Status Constants (`Status/Status.cs`):**
    *   This file centralizes various status and type definitions used throughout the application, promoting consistency and maintainability by avoiding "magic strings."
    *   Includes constants for:
        *   `CallStatus`: (e.g., "queued", "ringing", "in-progress", "completed", "busy", "noanswer", "failed")
        *   `AccountStatus`: (e.g., "active", "suspended", "trial", "closed")
        *   `ConferenceStatus`: (e.g., "init", "in-progress", "completed")
        *   `AccountType`: (e.g., "trial", "full")
        *   `CallDirection`: (e.g., "inbound", "outbound-api", "outbound-dial")

4.  **Time Utilities (`Time/EpochTimeConverter.cs`):**
    *   The `EpochTimeConverter` static class provides helper methods for time conversions:
        *   `ConvertFromEpochTime(long timestamp)`: Converts a Unix epoch timestamp (apparently in microseconds, as it divides by 1,000,000) to a standard .NET `DateTime` object.
        *   `ConvertToEpochTime(DateTime datetime)`: Converts a `DateTime` object into a Unix epoch timestamp (total seconds since January 1, 1970).
        *   `GetEpochTimeDifferent(long timestampFrom, long timestampTo)`: Calculates the difference in seconds between two epoch timestamps.

**Usage:**
Classes and utilities from `AgbaraUtil` are referenced and used by other projects within the AgbaraVOIP solution (like `AgbaraAPI`, `AgbaraXML`, etc.) to perform common tasks, manage state, and ensure consistent data handling.

**Key Files:**
-   `src/AgbaraUtil/Error/Error.cs`: Defines common error exception classes.
-   `src/AgbaraUtil/Exception/Exceptions.cs`: Defines specialized exception classes like `RedirectException`.
-   `src/AgbaraUtil/Gateway/`: Contains classes related to call routing and HTTP requests.
-   `src/AgbaraUtil/Status/Status.cs`: Defines constant strings for various application statuses.
-   `src/AgbaraUtil/Time/EpochTimeConverter.cs`: Provides time conversion utilities.
