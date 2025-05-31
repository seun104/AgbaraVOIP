# AgbaraVOIP Technical Documentation

## Introduction

AgbaraVOIP is a telephony platform designed to provide programmable voice and SMS capabilities. It allows developers to build applications that can make and receive phone calls, send and receive SMS messages, and manage complex call flows. The system is composed of several key components that work together:

*   **AgbaraAPI:** The main entry point for controlling the platform via a RESTful API.
*   **AgbaraXML:** An XML-based language used to define call control logic, processed when Freeswitch interacts with the system for call handling.
*   **Domain:** Contains the core business logic, data models (entities like Accounts, Calls, Applications), and services for data persistence (using MongoDB).
*   **Freeswitch Connector:** A low-level library for interacting with the Freeswitch telephony engine via its Event Socket Layer (ESL).
*   **AgbaraConsole:** A console application to launch and host the backend services.
*   **AgbaraService:** A Windows Service wrapper to run AgbaraVOIP in the background.
*   **AgbaraUtil:** A shared library of common utilities and constants.
*   **Sample Application:** Demonstrates how to use the AgbaraAPI.

This document details the architecture and implementation of each of these components.
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
        -   **AgbaraAPI:** It starts the AgbaraAPI. The code refers to `AgbaAPIWcfHostingServer.Start()`. Given that `AgbaraAPI` is built with NancyFX for self-hosting, this is likely a class name that still invokes the same Nancy-based self-hosting mechanism, rather than indicating a separate WCF host. The API is started asynchronously using `Task.Factory.StartNew()`.
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
-   However, in the provided code for `AgbaraService`, `IsDBOk()` always returns `true`, and `SetupDB()` is an empty method. This suggests that the database setup responsibility primarily lies with `AgbaraConsole` or is handled through a different mechanism when deploying as a service, with this part being likely inactive in `AgbaraService`.

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
        *   `ConvertFromEpochTime(long timestamp)`: Converts a Unix epoch timestamp to a standard .NET `DateTime` object. Note: The implementation divides the input timestamp by 1,000,000, suggesting it expects the input timestamp to be in microseconds, which is a less common convention than seconds or milliseconds for Unix epoch times.
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

## AgbaraXML

**Purpose:**
`AgbaraXML` is the component responsible for defining and processing call control logic using an XML-based language, similar in concept to Twilio's TwiML. It acts as the bridge between high-level application instructions (often triggered via the `AgbaraAPI`) and the low-level telephony operations of the Freeswitch platform. When Freeswitch handles a call, it can be configured to connect to the `AgbaraXMLServer` to fetch and execute these XML instructions, dictating the call flow.

**Core Components:**

1.  **`AgbaraXMLServer.cs`:**
    *   This class implements a TCP server that listens for outbound connections from Freeswitch (typically on port 8085).
    *   When Freeswitch connects (e.g., upon receiving an incoming call that needs to be handled by an Agbara application, or when an outbound call needs to execute an XML application via the API), the `AgbaraXMLServer` accepts the connection and instantiates an `FSOutbound` object to handle that specific call session.

2.  **`FSOutbound.cs`:**
    *   This is the central class for managing an individual call session controlled by AgbaraXML. An instance is created for each connection from Freeswitch.
    *   **Session Initialization:**
        *   Establishes a connection with the Freeswitch event socket for the specific call channel.
        *   Subscribes to necessary Freeswitch events (e.g., `CHANNEL_EXECUTE_COMPLETE`, `CHANNEL_HANGUP_COMPLETE`, `CUSTOM` events for `Dial` and `Conference` interactions).
        *   Retrieves initial call parameters (e.g., `AccountSid`, `CallSid`, `To`, `From`, and the crucial `agbara_answer_url` or `agbara_transfer_url`) from Freeswitch channel variables. This URL dictates where the first AgbaraXML document will be fetched.
    *   **XML Processing Loop:**
        *   The `ProcessCall()` method orchestrates the fetching, parsing, and execution of AgbaraXML documents.
        *   **Fetch (`FetchResponse`)**: Makes an HTTP GET or POST request to the current `target_url` (initially the `agbara_answer_url`). It sends current call session parameters (e.g., `CallSid`, `AccountSid`, `From`, `To`, `Digits` if any were collected) with the request. It expects an XML document in the HTTP response. This HTTP request is made using the `HTTPRequest` utility from `AgbaraUtil`.
        *   **Lex (`LexXml`)**: Parses the received XML string into a list of `XElement` objects. It validates that the root element is `<Response>` and that the child elements are recognized AgbaraXML verbs.
        *   **Parse (`ParseXml`)**: Converts the `XElement` objects into specific C# element instances (e.g., `<Say>` becomes a `Say` object). It uses `Util/ElementTypeLoader` to dynamically find and instantiate the correct class based on the XML tag name. It also handles the parsing of nested elements (e.g., `<Play>` inside `<Gather>`).
        *   **Execute (`ExecuteResponse`)**: Iterates through the list of parsed C# element objects and calls the `Run()` method on each. The `Run()` method of each element contains the logic to interact with Freeswitch (e.g., by calling `outboundClient.playback()`, `outboundClient.bridge()`, etc.).
    *   **Event-Driven Control Flow:**
        *   `FSOutbound` uses a `BlockingCollection<Event>` (`_actionEventQueue`) to wait for Freeswitch to complete blocking operations (like playing a file or bridging a call) before proceeding to the next XML element or fetching new XML.
        *   Event handlers (`OnChannelExecuteComplete`, `OnCustomEvent`, etc.) process events from Freeswitch, update call state, and signal the completion of actions.
    *   **Redirection and State:**
        *   If an AgbaraXML document contains a `<Redirect>` element, `FSOutbound` will update its `target_url` and fetch a new XML document from that URL, allowing for dynamic call flow changes.
        *   It maintains session parameters (`session_params`) which are updated throughout the call (e.g., with collected DTMF digits) and sent with each request for new XML.

3.  **XML Elements (`src/AgbaraXML/Element/` directory):**
    *   These classes define the vocabulary of AgbaraXML. Each class typically corresponds to a Freeswitch application or a call control action.
    *   **`Element.cs` (Base Class):**
        *   An abstract base class providing common functionality for all XML elements, including attribute parsing, text content extraction, and a virtual `Execute(FSOutbound client)` method that derived classes override to implement their specific logic.
        *   `Nestables`: Defines which other elements can be legally nested within the current element.
    *   **Key AgbaraXML Verbs:**
        *   **`<Say>`**: Synthesizes speech from text and plays it to the caller. Attributes: `voice`, `language`, `loop`, `engine`.
        *   **`<Play>`**: Plays an audio file to the caller. Attributes: `loop`. Can handle local file paths or URLs.
        *   **`<Gather>`**: Plays prompts (using nested `<Play>`, `<Say>`, or `<Pause>`) and then collects a sequence of DTMF digits entered by the caller. Attributes: `action` (URL to send digits to), `method`, `numDigits`, `timeout`, `finishOnKey`.
        *   **`<Record>`**: Records the call audio. Attributes: `action`, `method`, `maxLength`, `timeout`, `finishOnKey`, `playBeep`. The recording file path and duration are typically sent to the `action` URL.
        *   **`<Dial>`**: Attempts to connect the current call to another party (or parties). Can contain text (a single number to dial) or nested `<Number>` or `<Conference>` elements. Attributes: `action`, `method`, `callerId`, `timeout`, `timeLimit`, `hangupOnStar`, `record`.
            *   **`<Number>`**: (Nested in `<Dial>`) Specifies a phone number or SIP URI to dial. Attributes: `sendDigits` (DTMF to send upon answer).
        *   **`<Conference>`**: Adds the caller to a conference room. Attributes: `muted`, `beep`, `startConferenceOnEnter`, `endConferenceOnExit`, `maxMembers`, `waitUrl` (for music on hold, which can be another AgbaraXML document), `waitMethod`. The text content of the element is the conference room name.
        *   **`<Hangup>`**: Terminates the call.
        *   **`<Pause>`**: Pauses call execution for a specified duration. Attribute: `length` (in seconds).
        *   **`<Redirect>`**: Instructs `FSOutbound` to fetch and process a new AgbaraXML document from the URL specified in the text content of this element. Attribute: `method`. This is key for building multi-step call flows.
        *   **`<Reject>`**: Rejects an incoming call, optionally providing a reason. Attribute: `reason`.
        *   **`<PreAnswer>`**: If the call has not yet been answered, this element will answer it.
        *   **`<Sms>`**: (Details less clear from snippets) Likely used for sending an SMS message. Attributes: `to`, `from`, `action`, `method`.
        *   **`<Client>`**: (Details less clear from snippets) Potentially for interacting with SIP clients registered with Freeswitch.

4.  **Utilities (`src/AgbaraXML/Util/` directory):**
    *   **`ElementTypeLoader.cs`**: Dynamically discovers all available AgbaraXML element classes (those inheriting from `Element`) in the assembly. This allows `FSOutbound` to instantiate the correct C# object based on the XML tag name encountered during parsing.
    *   **`Helpers.cs`**: (Assumed) Contains miscellaneous helper functions for XML processing or string manipulation.
    *   **`LogWriter.cs`**: (Assumed) Provides logging capabilities specific to the AgbaraXML component.

**Workflow Example:**
1.  Freeswitch receives an incoming call and is configured to connect to `AgbaraXMLServer` (e.g., `socket://127.0.0.1:8085`).
2.  `AgbaraXMLServer` creates an `FSOutbound` instance.
3.  `FSOutbound` gets the initial `answer_url` from Freeswitch channel variables.
4.  `FSOutbound` makes an HTTP request to this `answer_url`.
5.  The application at `answer_url` (e.g., an `AgbaraAPI` endpoint) returns an AgbaraXML document, for example:
    ```xml
    <Response>
        <Say voice="female">Welcome to Agbara VOIP.</Say>
        <Gather action="/handle_digits" numDigits="1">
            <Say>Press 1 for sales, 2 for support.</Say>
        </Gather>
        <Hangup/>
    </Response>
    ```
6.  `FSOutbound` parses this XML.
7.  It executes `<Say>`, making Freeswitch play "Welcome...".
8.  It executes `<Gather>`, making Freeswitch play "Press 1..." and wait for 1 digit.
9.  User presses a digit. `FSOutbound` makes an HTTP request to `/handle_digits` with the pressed digit.
10. The application at `/handle_digits` returns new AgbaraXML, and the process continues until a `<Hangup>` is executed or the call otherwise ends.

This component is fundamental to the dynamic call control capabilities of the AgbaraVOIP system.

## Domain (AgbaraDomain.csproj)

**Purpose:**
The `Domain` project is the heart of the AgbaraVOIP application, encapsulating its core business logic, data models (entities), and service layer. It defines the fundamental data structures the application operates on and the business rules that govern them. This project promotes a separation of concerns, isolating the core domain from the presentation (e.g., `AgbaraAPI`) and infrastructure (e.g., Freeswitch interaction in `AgbaraXML`) layers.

**Key Components:**

1.  **Entities (`src/Domain/Objects/` directory):**
    *   These are Plain Old CLR Objects (POCOs) that model the primary data elements of the AgbaraVOIP system.
    *   **`EntityBase.cs`**: An abstract base class, though in the provided snippets it's empty. Entities generally define their own string-based unique identifiers (`Sid`) which are typically GUIDs prefixed with a two-letter entity code (e.g., "AC" for Account, "CA" for Call).
    *   **Key Entities:**
        *   **`Account.cs`**: Represents user or sub-user accounts. Includes properties like `Sid`, `ParentSid` (for sub-accounts), `FriendlyName`, `AuthToken`, `Status` (active, suspended), and `Type` (trial, full).
        *   **`Application.cs`**: Defines user-configurable voice and SMS applications. Contains URLs (`VoiceUrl`, `SmsUrl`, fallbacks, status callbacks) that AgbaraVOIP will request to fetch AgbaraXML instructions or send status updates.
        *   **`Call.cs`**: Represents a voice call, tracking details like `Sid`, `AccountSid`, `CallerId` (From), `CallTo` (To), `AnswerUrl` (for AgbaraXML), `Status` (queued, ringing, in-progress, completed), `Direction`, `Duration`, `Price`, `StartTime`, `EndTime`.
        *   **`Conference.cs`**: (Inferred) Models a multi-party conference call, including its status and participants.
        *   **`FSServer.cs`**: Stores configuration details for Freeswitch servers (host, port, password).
        *   **`Gateway.cs`**: Defines configurations for outbound VOIP gateways, including dial strings, codecs, retry counts, and routing rules.
        *   **`Notification.cs`**: (Inferred) Represents system notifications or alerts.
        *   **`Participant.cs`**: (Inferred) Represents a participant within a conference call.
        *   **`Recording.cs`**: (Inferred) Stores metadata about call or conference recordings, such as duration and the URL/path to the recording file.
        *   **`SMS.cs`**: (Inferred) Represents an SMS message, including sender, recipient, body, status, and price.
    *   Many entities use status constants defined in `AgbaraUtil` (e.g., `AgbaraCommon.CallStatus`, `AgbaraCommon.AccountStatus`).

2.  **Services (`src/Domain/Services/` directory):**
    *   This layer abstracts the data access and business logic operations. It's structured using service interfaces and their concrete implementations.
    *   **Service Interfaces (e.g., `IAccountService.cs`, `ICallService.cs`):**
        *   These define the contracts for business operations related to each entity. For example:
            *   `IAccountService`: `Validate(sid, authToken)`, `CreateAccount(...)`, `GetAccount(sid)`, `ChangeAccountStatus(...)`.
            *   `ICallService`: `AddCallLog(...)`, `UpdateCallStatus(callSid, status)`, `GetAllCalls(accountSid)`.
        *   Similar interfaces exist for other entities: `IApplicationService`, `IConferenceService`, `IFSService` (for Freeswitch server and gateway configurations), `INotificationService`, `IRecordingService`, `ISMSService`.
    *   **Service Implementations (`src/Domain/Services/MongoDB/` directory):**
        *   This directory provides concrete implementations of the service interfaces, specifically tailored for MongoDB as the data persistence mechanism.
        *   **Repository Pattern:** The implementations leverage the `DreamSongs.MongoRepository.MongoRepository<T>` generic repository. This abstracts direct MongoDB driver calls for common CRUD (Create, Read, Update, Delete) operations. For instance, `AccountService` uses a `MongoRepository<Account>`.
        *   Each service (e.g., `AccountService.cs`, `CallService.cs`) implements its corresponding interface by using the MongoRepository to interact with the appropriate MongoDB collection.
        *   Some services, like `CallService`, use `callRepo.RequestStart()` which might be a feature of `MongoRepository` to manage connections or transactions within the scope of a request.

**Design Principles:**
*   **Separation of Concerns:** The Domain project clearly separates business entities and logic from how data is stored or how the system interacts with external services.
*   **Interface-Based Services:** Using interfaces for services allows for dependency injection and makes the system more modular and testable. For example, `AgbaraAPI` modules would depend on `IAccountService` rather than the concrete `Mongo.AccountService`.
*   **Data Persistence Abstraction:** While MongoDB is the chosen implementation here, the use of service interfaces and potentially a more abstract repository pattern (even if `MongoRepository` is used directly in services) could allow for switching to a different database system with less impact on the core application logic.

This project forms the foundational layer upon which other components like `AgbaraAPI` build their functionality, ensuring that business rules and data management are handled consistently.

## Freeswitch Connector (`Freeswitch.csproj`)

**Purpose:**
The `Freeswitch` project provides the low-level C# bindings and abstractions necessary for interacting with the Freeswitch telephony platform via its Event Socket Layer (ESL). This component is crucial for enabling the AgbaraVOIP application to programmatically send commands to Freeswitch, receive and process events from it, and manage call sessions in real-time.

**Core Classes and Functionality:**

1.  **`Transport.cs`:**
    *   Defines an `ITransport` interface for sending and receiving data over a network connection.
    *   **`InboundTransport`**: Implements `ITransport` for scenarios where the Agbara application initiates a TCP connection *to* Freeswitch (e.g., for sending API commands or originating calls).
    *   **`OutboundTransport`**: Implements `ITransport` for scenarios where Freeswitch initiates a TCP connection *to* the Agbara application (e.g., when Freeswitch's dialplan uses the `socket` application to connect to `AgbaraXMLServer`). It wraps an existing `Socket`.

2.  **`Event.cs`:**
    *   Represents a Freeswitch event or message received over the ESL.
    *   Parses raw ESL messages (which are typically multi-line key-value pairs, followed by an optional body) into a dictionary of headers and a body string.
    *   Provides methods for accessing event headers (e.g., `GetHeader("Event-Name")`, `GetContentLength()`, `GetReplyText()`) and the event body.
    *   Includes specialized derived classes to represent different types of ESL messages:
        *   **`APIResponse`**: For responses to synchronous `api` commands.
        *   **`BgApiResponse`**: For responses to asynchronous `bgapi` (background API) commands, includes `GetJobUUID()`.
        *   **`CommandResponse`**: For general command replies from Freeswitch.
        *   **`JsonEvent`**: For events formatted by Freeswitch as JSON. It uses the `fastJSON` library to deserialize the JSON payload into the event's header dictionary.

3.  **`Commands.cs`:**
    *   An abstract class defining a comprehensive set of methods that map to Freeswitch ESL commands and dialplan applications (e.g., `APICommand`, `BgAPICommand`, `Exit`, `Answer`, `Bridge`, `Hangup`, `Playback`, `Record`, `Conference`, `SetVar`, `GetVar`, `Speak`, `PlayAndGetDigits`).
    *   These methods typically wrap lower-level `ProtocolSend` or `ProtocolSendMsg` calls.

4.  **`EventSocket.cs`:**
    *   This is an abstract base class that forms the core of the ESL communication logic. It inherits from `Commands`.
    *   **Connection Management**: Handles basic connection state.
    *   **Message Parsing and Handling**:
        *   Uses a `_response_callbacks` dictionary to map Freeswitch message `Content-Type` (e.g., `text/event-json`, `command/reply`, `api/response`) to specific internal parsing methods.
        *   `read_event()`: Reads data from the underlying `Transport` to construct an `Event` object.
    *   **Event Dispatching**:
        *   Maintains an `event_handlers` dictionary mapping Freeswitch `Event-Name` strings (like "CHANNEL_CREATE", "CHANNEL_HANGUP", "CUSTOM") to `EventHandlers` delegates.
        *   Provides public C# events (e.g., `OnCUSTOM`, `OnCHANNEL_HANGUP_COMPLETE`) that allow consuming code to subscribe to specific Freeswitch events. When an event is received from Freeswitch, `dispatch_event` invokes the appropriate C# event.
    *   **Synchronous Command Execution**:
        *   Implements `ProtocolSend` (for simple commands) and `ProtocolSendMsg` (for `sendmsg` type commands which execute applications on a channel).
        *   Uses a `BlockingCollection<Event> _response_queue` to manage synchronous command execution. When a command is sent, the calling thread blocks waiting for a response to be added to this queue by the message handling loop.
    *   **Background Event Processing**:
        *   `start_event_handler()`: Launches a background task (`handle_events()`) that continuously reads and dispatches incoming events from Freeswitch.

5.  **`InboundSocket.cs`:**
    *   Inherits from `EventSocket`. Represents an "inbound" ESL connection, where the Agbara application initiates the connection to Freeswitch.
    *   Used by `AgbaraAPI` (specifically `ApiServer.fsInbound`) to send commands to Freeswitch (e.g., to originate calls).
    *   The `connect()` method establishes the TCP connection, sends the `auth` command with the password, and subscribes to desired events (e.g., `event json ALL`).
    *   `serve_forever()` can be used to keep the connection active for continuous event listening.

6.  **`OutboundSocket.cs`:**
    *   Contains classes for "outbound" ESL mode, where Freeswitch initiates the connection to a listening Agbara application.
    *   **`OutboundClient`**: Inherits from `EventSocket`. An instance is created for each incoming connection from Freeswitch.
        *   The `AgbaraXML.FSOutbound` class (which handles AgbaraXML processing) is a subclass of `OutboundClient`.
        *   Its `connect()` method sends the initial `connect` command to Freeswitch over the already established socket and subscribes to events. It captures the channel's unique ID.
        *   The `Run()` method is intended to be overridden by subclasses (like `FSOutbound`) to implement the specific call handling logic.
    *   **`OutboundServer`**: A TCP server (`TcpListener`) that listens for incoming ESL connections from Freeswitch.
        *   The `AgbaraXML.AgbaraXMLServer` is an `OutboundServer`.
        *   When a connection is accepted, it typically instantiates an `OutboundClient` subclass (e.g., `FSOutbound`) to handle the call session.

**Interaction Summary:**
*   **Controlling Freeswitch (Inbound Mode):** Components like `ApiServer` use an `InboundSocket` to connect to Freeswitch, send commands (e.g., originate a call, hangup a call), and react to specific events related to those commands.
*   **Being Controlled by Freeswitch (Outbound Mode):** When Freeswitch's dialplan executes the `socket` application pointing to an `OutboundServer` (like `AgbaraXMLServer`), Freeswitch connects, and an `OutboundClient` instance (like `AgbaraXML.FSOutbound`) takes control of that specific call channel, typically by sending commands to execute applications (play audio, bridge, etc.) based on its own logic (e.g., parsing AgbaraXML).

This `Freeswitch` project provides a robust C# interface to the Freeswitch Event Socket, abstracting many of the raw protocol details and offering an event-driven model for call control and management.

## Sample Application (`Sample.csproj`)

**Purpose:**
The `Sample` project is a C# console application that serves as a practical example of how to interact with the `AgbaraAPI`. It demonstrates the basic steps involved in making an API request to initiate an outbound phone call, showcasing a typical client-side integration.

**Key Components and Workflow:**

1.  **`Program.cs`:**
    *   The main entry point of the sample application.
    *   It briefly pauses execution (presumably to allow the AgbaraAPI server to start if launched concurrently) and then invokes the `ApiSample.Sample1()` method in a separate task.
    *   It keeps the console window open (`Console.ReadLine()`) to display the output from the API call.

2.  **`RestfulClient.cs`:**
    *   Defines the `AgbaraRESTAPIClient` class, a simple HTTP client tailored for interacting with the AgbaraAPI (or any RESTful API requiring basic authentication).
    *   **Authentication**: The client is instantiated with an Account SID and Auth Token, which are used for HTTP Basic Authentication in each request.
    *   **HTTP Methods**: It supports GET, POST, PUT, and DELETE requests.
    *   **Request Construction**: It formats request parameters (from a `Hashtable`) into query strings for GET requests or into `application/x-www-form-urlencoded` bodies for other methods.
    *   **URL Building**: It prepends a base API URL (hardcoded as `http://127.0.0.1:8082`) to the provided API paths.
    *   **Error Handling**: Includes basic error handling for `WebException` to display error messages and response content.

3.  **`ApiSample.cs`:**
    *   Contains the core logic for the sample API interaction.
    *   **`RestSample()` method:**
        *   **Credentials**: Uses hardcoded `ACCOUNT_SID` ("seun104") and `ACCOUNT_TOKEN` ("agbara") for the demonstration. In a real application, these would be managed securely.
        *   **API Client Instantiation**: Creates an instance of `AgbaraRESTAPIClient`.
        *   **Request Parameters**: Constructs a `Hashtable` containing parameters required to initiate an outbound call. This includes:
            *   `AuthId` and `AuthToken` (though the client handles authentication via headers, these might be expected by the specific `/test/call` endpoint).
            *   `from`: The caller ID to be displayed.
            *   `to`: The destination phone number. This is read from the `App.config` file (`AppSettings["ToNumber"]`).
            *   `AnswerUrl`: The URL that AgbaraVOIP will request to fetch AgbaraXML instructions when the outbound call is answered. This is also read from `App.config` (from `AppSettings["AnswerlUrl"]` - noting that "AnswerlUrl" in the config key is likely a typo and should be "AnswerUrl").
        *   **API Call**: Makes a POST request to the `/test/call` endpoint of the AgbaraAPI. This endpoint seems to be a specific test or sample endpoint, possibly defined in `AgbaraAPI/Modules/TestModule.cs`.
        *   The method returns the raw string response from the API.
    *   **`Sample1()` method:**
        *   Calls `RestSample()` to perform the API request.
        *   Prints the API response to the console.

4.  **`App.config`:**
    *   An XML configuration file used to store application settings that can be changed without recompiling the code.
    *   For this sample, it is expected to contain:
        *   `ToNumber`: The phone number to which the sample application will attempt to make a call.
        *   `AnswerlUrl`: The URL that will serve the AgbaraXML to control the call flow once answered. (Note: "AnswerlUrl" is likely a typo in the original codebase and was intended to be "AnswerUrl").

**How it Demonstrates API Usage:**
The sample application effectively illustrates:
*   **Client Initialization**: How to set up an HTTP client for the AgbaraAPI.
*   **Authentication**: How to provide credentials for Basic Authentication.
*   **API Endpoint Interaction**: Making a POST request to a specific API endpoint (`/test/call`).
*   **Parameter Passing**: Sending necessary data (caller ID, recipient, Answer URL) to the API.
*   **Response Handling**: Retrieving and displaying the raw response from the API.

This project provides a valuable starting point for developers looking to build applications that leverage the AgbaraVOIP platform's capabilities through its API.
