# Equate OGSD Monitoring Platform

Equate OGSD (Out-of-Band Service Gateway) is a cloud-native network monitoring platform designed for environments where reliable visibility and recovery access are required even during primary network failures.

The platform uses a two-plane architecture:

* Customer OOB Monitoring Plane
* UI/UX Cloud Plane

The customer environment is responsible only for collecting and securely forwarding telemetry. The cloud environment owns ingestion, persistence, APIs, and visualization.

---

# Architecture Overview

```
Customer OOB Monitoring Plane

Network Devices
      |
      v
SNMP Collector
      |
      v
Local Buffer
      |
      v
Outbound TLS Telemetry
      |
      v

Azure Cloud Plane

MQTT Broker
      |
      v
Cloud Ingestion Service
      |
      v
Backend API
      |
      v
Azure PostgreSQL

Frontend Dashboard
      |
      v
Backend API
```

---

# Customer OOB Monitoring Plane

The customer-side deployment runs inside the district or organization environment.

Its responsibilities are:

* Poll network devices using SNMP.
* Normalize device telemetry.
* Maintain temporary local buffering.
* Deliver telemetry outbound securely to the cloud.

The customer deployment is intentionally lightweight and isolated.

It does not host:

* Backend APIs.
* PostgreSQL databases.
* Frontend applications.
* Cloud ingestion services.

The customer environment acts as a telemetry collection point, not an application platform.

---

# UI/UX Cloud Plane

The cloud deployment runs in Microsoft Azure.

Responsibilities include:

* Receiving telemetry.
* Validating incoming data.
* Processing monitoring events.
* Persisting monitoring state.
* Providing APIs.
* Serving dashboard workflows.

The cloud plane contains:

* MQTT Broker.
* Cloud Ingestion Service.
* Backend API.
* PostgreSQL Database.
* Frontend Dashboard.

---

# Data Architecture

PostgreSQL is the system of record.

Stored information includes:

* Device inventory.
* Site configuration.
* User configuration.
* Telemetry history.
* Monitoring state.
* Alerts.

Local storage at customer sites is temporary only.

The data flow is:

```
SNMP Collector
      |
      v
Temporary Local Buffer
      |
      v
Cloud Ingestion
      |
      v
PostgreSQL
```

Removing the cloud database would reduce the platform to a live telemetry viewer and prevent historical monitoring, alerting, and configuration management.

---

# Initial Production Deployment

The first production deployment prioritizes operational simplicity and reliability.

Target architecture:

```
Azure Compute Host

├── MQTT Broker
├── Cloud Ingestion Service
├── Backend API
└── Reverse Proxy (Caddy/NGINX)


Separate Service:

└── Azure PostgreSQL
```

Stateless application services share compute initially.

The database remains isolated as a separate failure domain.

---

# API Gateway Strategy

Azure API Management is intentionally deferred.

APIM provides:

* Authentication enforcement.
* Rate limiting.
* API governance.
* External integration management.

It becomes valuable when Equate evolves into a multi-tenant SaaS platform.

The initial deployment uses:

```
Frontend
    |
    v
Reverse Proxy
    |
    v
Backend API
    |
    v
PostgreSQL
```

---

# Container Strategy

All services are designed as containers.

This enables future migration to:

* Azure Container Apps.
* Kubernetes / AKS.
* Larger distributed deployments.

The initial deployment avoids unnecessary infrastructure complexity while maintaining a path toward enterprise scale.

---

# Core Design Principles

## Outbound-Only Customer Connectivity

Customer environments do not require inbound cloud access.

Telemetry flows outbound from the OOB environment to Azure.

Benefits:

* Reduced attack surface.
* Easier firewall policies.
* Improved isolation.

---

## Cloud-Owned State

The cloud owns persistent state.

Customer deployments only maintain temporary delivery buffers.

---

## Separate Failure Domains

Applications and databases are separated.

Application failures should not compromise persistent monitoring data.

---

## Build for Scale, Deploy for Simplicity

The architecture supports future growth without requiring premature operational complexity.

Initial deployments optimize for:

* Low cost.
* Simple operations.
* Clear ownership boundaries.

---

# Deployments

Operational stacks live under [`deployments/`](deployments/):

| Profile | Purpose |
|---------|---------|
| [`end-to-end/`](deployments/end-to-end/) | Single-host Compose with every service (client-site smoke; real SNMP) |
| [`development/`](deployments/development/) | Mac cloud plane + OrbStack VM collector (GNS3 Cloud) |
| [`production/`](deployments/production/) | Hybrid skeleton: Azure cloud + on-site VxRail |

See [`deployments/README.md`](deployments/README.md) for commands, ports, and the promotion checklist.

---

# Development Status

Equate OGSD is currently under active development.

Current development priorities:

* SNMP collection engine.
* Telemetry normalization.
* Secure telemetry transport.
* Cloud ingestion pipeline.
* Monitoring dashboard.

---

# Technology Direction

## Customer Plane

* Go
* SNMP
* MQTT
* Docker

## Cloud Plane

* Azure
* PostgreSQL
* Containerized services
* REST API
* Web dashboard

---

# Project Goals

Equate OGSD aims to provide reliable infrastructure visibility and recovery capabilities for organizations operating distributed networks where traditional monitoring may fail during outages.

The platform is designed around one principle:

**The monitoring system must remain available when the primary network is unavailable.**
