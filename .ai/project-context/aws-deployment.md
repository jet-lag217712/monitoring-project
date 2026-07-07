# AWS Deployment

## Purpose

Defines cloud-hosted components.

## Cloud Responsibilities

AWS hosts:

- Monitoring Dashboard
- Email Service

AWS does not host:

- SNMP Collector
- MQTT Broker
- Ingestion Service
- PostgreSQL

## Connectivity

Cloud services communicate with the on-prem Backend API.

Security requirements:
- TLS encryption
- Authentication
- Restricted API access

## Deployment Model

The cloud layer is the presentation and notification layer.
The monitoring data pipeline remains on-premises.