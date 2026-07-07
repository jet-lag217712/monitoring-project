# SNMP Collector Architecture

## Purpose

The SNMP Collector gathers network telemetry from managed devices.

## Responsibilities

The collector:

-   Polls SNMP endpoints.
-   Queries configured OIDs.
-   Parses responses.
-   Publishes telemetry.

## Supported Model

Initial deployment: - SNMP v2 - Network switches

## Polling

Polling intervals should be configurable.

Examples: - Device health checks - Interface statistics - Resource
utilization

## Non-Responsibilities

The collector does not:

-   Store telemetry permanently.
-   Generate alerts.
-   Serve dashboard requests.
