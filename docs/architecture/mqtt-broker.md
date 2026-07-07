# MQTT Broker Architecture

## Purpose

The MQTT broker provides asynchronous telemetry transport.

## Responsibilities

The broker:

-   Receives telemetry from collectors.
-   Delivers messages to ingestion services.
-   Provides decoupling between producers and consumers.

## Communication Model

Publisher: - SNMP Collector

Subscriber: - Ingestion Service

## Security

Production deployment should use:

-   TLS
-   Authentication
-   Restricted topic permissions

## Scaling

Additional collectors can publish telemetry without changing downstream
services.
