# AgbaraVOIP GoLang Service: Performance Testing Strategy

## 1. Introduction and Objectives

This document outlines the strategy for performance testing the AgbaraVOIP GoLang service. The objectives are:

*   Assess the responsiveness, stability, and scalability of the system under various load conditions.
*   Identify performance bottlenecks in the API, service layer, database, or Freeswitch interactions.
*   Ensure the system meets defined performance targets before production deployment.
*   Provide a baseline for future performance regression testing.

## 2. Scope of Performance Testing

Testing will focus on key API endpoints and user flows that are critical or expected to handle significant traffic.

**Key Endpoints/Scenarios:**

*   **Authentication:**
    *   `POST /api/v1/auth/token`: Token generation throughput and latency.
*   **Core User Operations (Authenticated):**
    *   `POST /api/v1/accounts/{account_sid}/calls`: Call origination rate and API response time.
    *   `GET /api/v1/accounts/{account_sid}/applications`: Listing applications (simulating users checking configurations).
    *   `GET /api/v1/accounts/{account_sid}/calls`: Listing call history.
*   **AgbaraXML Processing Simulation (Indirect):**
    *   While direct load testing of the ESL-driven AgbaraXML execution path is complex via API load tests alone, the overall system stability and API responsiveness during periods of high call volume (simulated by many concurrent `OriginateCall` requests that trigger ESL interactions) will be observed.
*   **Admin Operations (Baseline Performance):**
    *   `GET /api/v1/admin/gateways`: Listing gateways.
    *   `GET /api/v1/admin/freeswitch-servers`: Listing Freeswitch servers.

**Out of Scope (for initial API-focused tests):**

*   Exhaustive testing of every single API endpoint permutation.
*   Frontend/UI performance.
*   Deep Freeswitch media processing performance (though its impact on the Go service will be observed).

## 3. Key Performance Indicators (KPIs) and Targets

The following KPIs will be measured, with example targets:

*   **Average Response Time (P95):**
    *   `/auth/token`: < 150ms
    *   Call Origination (`/calls` POST): < 300ms (API response, not call setup time)
    *   Resource Listing (GET): < 200ms
*   **Requests Per Second (RPS) / Throughput:**
    *   Define target RPS for each key endpoint based on expected load (e.g., 50-100 RPS for critical paths).
*   **Error Rate:** < 0.5% under target load.
*   **System Resource Utilization:**
    *   CPU Usage (Go service, DB, Freeswitch): < 75% average.
    *   Memory Usage: Stable, no memory leaks.
    *   Database Connection Pool: Monitor for exhaustion.

Targets should be refined based on specific deployment environment and business requirements.

## 4. Test Environment

*   **Dedicated Environment:** A staging or performance testing environment that mirrors production as closely as possible (hardware, software versions, network configuration, data volume).
*   **Load Generators:** One or more machines to run the load testing tool, separate from the application servers.
*   **Monitoring:** Real-time monitoring of application servers (Go service), database, Freeswitch, and load generators (Prometheus, Grafana, system performance tools).
*   **Data:** Sufficient and realistic test data in the database (e.g., accounts, applications).

## 5. Load Testing Tool

*   **Recommended: k6 (k6.io)**
    *   Modern, developer-friendly load testing tool.
    *   Test scripts written in JavaScript.
    *   Good for API testing, with features for checks, thresholds, and custom metrics.
    *   Example k6 scripts will be provided in the `/scripts/performance_tests/` directory.
*   Alternatives: JMeter, Vegeta, Locust.

## 6. Test Execution Process

1.  **Test Design:** Define specific test scenarios, load profiles (e.g., ramp-up, soak tests, stress tests), and duration for each scenario.
2.  **Script Development:** Create/update k6 (or chosen tool) test scripts.
3.  **Environment Setup:** Ensure the test environment is ready and instrumented for monitoring.
4.  **Baseline Test:** Run a low-load test to establish baseline performance.
5.  **Execute Load Tests:** Run the defined load test scenarios.
    *   Start with moderate load and gradually increase.
    *   Monitor KPIs and system resources in real-time.
6.  **Analyze Results:**
    *   Collect and analyze test results (response times, RPS, error rates).
    *   Correlate with system monitoring data to identify bottlenecks.
7.  **Reporting:** Document findings, including any identified issues, bottlenecks, and recommendations for optimization.
8.  **Optimization & Retesting:** If bottlenecks are found, apply optimizations and re-run tests to verify improvements.

## 7. Test Scenarios (High-Level Examples for k6)

*   **Auth Token Generation:** Simulate multiple users requesting tokens concurrently.
*   **Call Origination Burst:** Simulate many users initiating calls in a short period.
*   **Sustained Load - Mixed Scenario:** Simulate realistic concurrent usage of various API endpoints (listing resources, originating calls).
*   **Admin API Baseline:** Check response times for admin listing endpoints under light load.

See example k6 scripts in `/scripts/performance_tests/`.

## 8. Iteration

Performance testing should be an iterative process, run regularly as new features are added or significant changes are made to the system.
