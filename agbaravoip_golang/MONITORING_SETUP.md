# AgbaraVOIP GoLang Service: Basic Monitoring Setup

This document describes the basic monitoring setup provided via Docker Compose using Prometheus and Grafana.

## Overview

The `docker-compose.yml` in this directory includes services for:
*   **Prometheus:** Collects metrics from the AgbaraVOIP application.
*   **Grafana:** Allows for visualization of the collected metrics through dashboards.

The AgbaraVOIP application itself exposes metrics at its `/api/metrics` endpoint, which Prometheus is configured to scrape.

## Prerequisites

*   Docker and Docker Compose installed.
*   The AgbaraVOIP application running (e.g., via `docker-compose up app`).

## Running the Monitoring Stack

1.  **Ensure `prometheus.yml` is configured:**
    The `agbaravoip_golang/prometheus.yml` file configures Prometheus to scrape the `app` service (your Go application) on port `8080` at the `/api/metrics` path. Adjust the `targets` in `scrape_configs` if your Go application service name or port in `docker-compose.yml` is different.

2.  **Start the stack:**
    From the `agbaravoip_golang` directory, run:
    ```bash
    docker-compose up -d prometheus grafana
    ```
    If you want to run all services including the app, db, etc.:
    ```bash
    docker-compose up -d
    ```

## Accessing Services

*   **Prometheus:**
    *   URL: `http://localhost:9090`
    *   You can explore metrics, view targets, and see alert statuses here.
*   **Grafana:**
    *   URL: `http://localhost:3000`
    *   Default Credentials: `admin` / `grafana` (change this for any production-like environment).
    *   The Prometheus datasource should be auto-provisioned (see `grafana_datasources.yml`). You can verify this under Configuration -> Data Sources in Grafana.

## Dashboards in Grafana

*   **Importing Dashboards:** Grafana Labs hosts many pre-built dashboards. You can search for dashboards relevant to Go applications, Gin web framework, or general system performance.
    *   Example: Go Processes Dashboard (ID: `12816` on grafana.com/grafana/dashboards) - requires `go_exporter` or similar process metrics from your app, or use it as inspiration.
    *   The metrics exposed by this application (`agbaravoip_http_requests_total`, `agbaravoip_http_request_duration_seconds`) can be used to build custom dashboards.
*   **Creating Custom Dashboards:** Use the "Explore" feature in Grafana with the Prometheus datasource to build queries and visualize metrics like request rates, latencies, and error counts.

## Alerting

*   **Prometheus Alert Rules:** Example alert rules are defined in `agbaravoip_golang/alert.rules.yml`. These rules are loaded by Prometheus if referenced in its main configuration (`prometheus.yml` - currently commented out).
    ```yaml
    # In prometheus.yml, uncomment or add:
    # rule_files:
    #   - "alert.rules.yml"
    ```
*   **Alertmanager (Not Included in Basic Setup):** For alerts to be routed (e.g., email, Slack, PagerDuty), Prometheus needs to be configured to send alerts to an Alertmanager instance. Setting up Alertmanager is beyond this basic monitoring scope but is the typical next step for a full alerting pipeline.
    ```yaml
    # In prometheus.yml, uncomment or add and configure:
    # alerting:
    #   alertmanagers:
    #     - static_configs:
    #         - targets: ['alertmanager:9093'] # If Alertmanager runs as a Docker service
    ```

## Structured Logging

The application uses structured logging (logrus). In a production environment, these logs should be collected, aggregated, and made searchable using a log management solution like:
*   ELK Stack (Elasticsearch, Logstash, Kibana)
*   Grafana Loki (integrates well with Prometheus and Grafana)
*   Cloud-specific logging services (e.g., AWS CloudWatch Logs, Google Cloud Logging).

This basic setup does not include a log aggregation solution.

## Further Steps

*   Integrate Alertmanager for notifications.
*   Add more specific metrics from the application (e.g., ESL command stats, database performance).
*   Create more detailed Grafana dashboards tailored to AgbaraVOIP.
*   Consider adding `node_exporter` to Docker Compose to get system-level metrics from the host running Docker, and `cadvisor` for container metrics.
