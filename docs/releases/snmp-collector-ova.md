
# Equate Collector Appliance PRD

**Document Version:** 1.0  
**Product:** Equate Collector Appliance  
**Status:** Draft  
**Target Audience:** Engineering, Codex/AI coding agents, infrastructure engineers

---

# 1. Product Overview

The Equate Collector Appliance is a VMware-deployable virtual appliance designed to simplify deployment of the Equate SNMP Collector within customer networks.

The appliance packages the complete collector runtime into an installable OVA file. A customer deploys the VM into VMware ESXi/vSphere, completes initial configuration through a CLI wizard, and the collector automatically registers with Equate Cloud.

The appliance is designed for environments where:

- Customer networks block inbound access.
- Only outbound HTTPS/MQTT communication is allowed.
- IT staff require simple deployment.
- Multiple customer sites must be managed securely.

The final user experience should resemble enterprise appliances such as GNS3 VM, Meraki devices, or network monitoring collectors.

---

# 2. Goals

## Primary Goal

Create a self-contained Equate Collector appliance that can be deployed by a non-developer administrator.

The deployment workflow:

```

Download OVA

↓

Deploy into VMware

↓

Boot VM

↓

Run Equate Setup

↓

Register Collector

↓

Receive Certificates

↓

Connect to Equate Cloud

↓

Begin SNMP Monitoring

```

---

# 3. Non-Goals

The first production version will not:

- Replace VMware management tooling.
- Provide a graphical configuration interface.
- Support arbitrary hypervisors.
- Manage customer network devices directly.
- Provide inbound remote administration.
- Implement automatic operating system upgrades.

---

# 4. System Architecture

```

```
                     Equate Cloud

          +----------------------------+
          | Registration Service       |
          | Certificate Authority       |
          | Update Service             |
          | MQTT Broker                |
          +-------------+--------------+
                        |
                        |
                     TLS/mTLS
                        |
                        |
          +-------------v--------------+
          | Equate Collector Appliance |
          |                            |
          | Debian Minimal             |
          | systemd                    |
          | Docker                     |
          | Equate Agent               |
          | SNMP Collector             |
          | Local SQLite Storage       |
          +-------------+--------------+
                        |
                        |
                       SNMP

                        |
                        |
              Customer Network Devices
```

```

---

# 5. Operating System

## Base Image

The appliance should use:

```

Debian 12 Minimal

```

or newer stable Debian release.

Reasons:

- Low resource overhead.
- Long lifecycle support.
- Stable package ecosystem.
- Suitable for appliance deployments.
- Minimal unnecessary services.

---

# 6. Virtual Machine Requirements

## Minimum Requirements

```

CPU:
2 vCPU

Memory:
2 GB RAM

Storage:
32 GB

Network:
1 virtual NIC

```

## Recommended Requirements

```

CPU:
4 vCPU

Memory:
4 GB RAM

Storage:
40 GB

Network:
1 virtual NIC

```

---

# 7. Appliance Internal Layout

Filesystem:

```

/opt/equate/

├── agent/
│   └── equate-agent

├── collector/
│   └── docker-compose.yml

├── config/
│   └── collector.yaml

├── certificates/
│   ├── ca.crt
│   ├── collector.crt
│   └── collector.key

└── data/
└── collector.db

```

---

# 8. Service Architecture

The appliance uses systemd as the service manager.

Services:

```

equate-agent.service

equate-collector.service

docker.service

```

Startup sequence:

```

VM Boot

↓

systemd

↓

equate-agent

↓

collector container

↓

MQTT connection

↓

SNMP polling begins

```

---

# 9. Equate Agent

The Equate Agent is the native management daemon running on the appliance.

Responsibilities:

- Manage collector lifecycle.
- Handle registration.
- Manage certificates.
- Handle updates.
- Provide CLI backend.
- Report health information.

The agent should not require Docker commands from the user.

---

# 10. CLI Interface

Command:

```

equate

```

Main commands:

```

equate status

equate logs

equate reconfigure

equate restart

equate restart-all

equate update

equate version

```

---

# 11. Initial Setup Workflow

First boot:

```

================================
Equate Collector Setup
======================

Collector ID:

>

Client ID:

>

Site ID:

>

Registration Token:

>

MQTT Endpoint:

>

Starting registration...

```

The system then:

```

Generate private key

↓

Create certificate request

↓

Register with Equate Cloud

↓

Receive certificate

↓

Store credentials

↓

Start collector

```

---

# 12. Certificate Architecture

## Overview

All collector communication uses mutual TLS.

The collector has a unique identity.

Example:

```

Collector:
ogsd-hq-01

Certificate:

CN:
collector-ogsd-hq-01

SAN:
collector-ogsd-hq-01.ogsd.equate.cloud

```

---

# 13. Certificate Registration

Registration flow:

```

Collector

Generate Keypair

```
    |

    v
```

Registration API

```
    |

    v
```

Equate Certificate Authority

```
    |

    v
```

Signed Collector Certificate

```
    |

    v
```

Collector Activation

```

---

# 14. Certificate Database Mapping

Cloud database:

```

collectors

id
client_id
site_id
certificate_fingerprint
status
last_seen
created_at

```

Example:

```

Certificate Fingerprint

```
    |

    v
```

Collector Identity

```
    |

    v
```

Customer Site

```

---

# 15. MQTT Authentication

Collectors authenticate using certificates.

Example:

```

Collector Certificate:

collector-ogsd-hq-01

```

MQTT broker validates:

```

Certificate valid?
|
|
v

Is collector authorized?

```
    |
    |
    v
```

Allow publish

```

---

# 16. MQTT Topics

Example:

```

equate/{client}/{site}/telemetry

```

Example:

```

equate/ogsd/hq/telemetry

```

Collectors cannot publish outside their assigned namespace.

---

# 17. Update System

## Update Goals

The collector must:

- Update without SSH access.
- Validate software authenticity.
- Support rollback.
- Support release channels.

---

# 18. Release Channels

Supported channels:

```

stable

beta

edge

```

Example:

```

equate update --channel stable

```

---

# 19. Update Flow

```

Collector

```
    |
    |
    v
```

Update API

```
    |
    |
    v
```

Release Manifest

```
    |
    |
    v
```

Verify Signature

```
    |
    |
    v
```

Download Container

```
    |
    |
    v
```

Restart Service

```
    |
    |
    v
```

Health Check

```
    |
    |
    v
```

Commit Update

```

---

# 20. Release Artifact

Each release contains:

```

collector-version.tar.gz

manifest.json

signature.sig

````

Example manifest:

```json
{
  "version": "1.2.0",
  "sha256": "hash",
  "channel": "stable"
}
````

---

# 21. Container Update Strategy

The appliance OS remains static.

Only application containers update.

Example:

Before:

```
collector:v1.0
```

After:

```
collector:v1.1
```

Rollback:

```
collector:v1.1 failed

↓

collector:v1.0 restored
```

---

# 22. CLI Examples

## Status

```
equate status
```

Example:

```
Collector:
ogsd-hq-01

Version:
1.0.0

MQTT:
Connected

Devices:
42

Last heartbeat:
10 seconds ago
```

---

## Reconfigure

```
equate reconfigure
```

Allows:

* Collector ID changes.
* Client reassignment.
* Certificate renewal.
* MQTT configuration changes.

---

## Restart

```
equate restart
```

Restarts:

* Collector container.
* MQTT connection.
* Monitoring services.

---

## Restart All

```
equate restart-all
```

Restarts all Equate services.

---

# 23. Security Requirements

The appliance must:

* Require no inbound firewall rules.
* Use TLS for all cloud communication.
* Use mTLS for collector identity.
* Protect private keys.
* Support certificate revocation.
* Validate update signatures.
* Prevent unauthorized tenant access.

---

# 24. Development Milestones

## Milestone 1: Appliance Foundation

Deliver:

* Debian VM image.
* Docker runtime.
* systemd services.
* OVA export.

---

## Milestone 2: CLI Management

Deliver:

```
equate status

equate logs

equate restart
```

---

## Milestone 3: Cloud Registration

Deliver:

* Registration API.
* Certificate issuance.
* MQTT authentication.

---

## Milestone 4: Update System

Deliver:

* Release manifests.
* Artifact storage.
* Update client.
* Rollback support.

---

## Milestone 5: Production Release

Deliver:

* VMware OVA.
* Installation documentation.
* Deployment runbook.
* Recovery procedures.

---

# 25. Acceptance Criteria

A school district technician must be able to:

```
Deploy OVA

↓

Boot VM

↓

Complete Setup Wizard

↓

Register Collector

↓

Connect Network Devices

↓

View Telemetry
```

without:

* SSH access.
* Manual certificate generation.
* Linux administration.
* Developer assistance.

---

# 26. Future Enhancements

Potential future features:

* Web-based local management UI.
* Hardware appliance deployments.
* Automated network discovery.
* Remote diagnostics.
* Fleet management dashboard.
* Zero-touch provisioning.

```
```
