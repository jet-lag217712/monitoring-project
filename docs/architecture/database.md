# Database Architecture

## Purpose

PostgreSQL is the system of record for the OGSD monitoring platform.

It stores normalized site inventory, monitored device inventory, interface information, collected metric samples, current monitoring state, and generated alerts. The database provides a queryable historical view of network state for consumption by the Backend API and UI/UX Cloud Plane.

## Plane Ownership

Plane: UI/UX Cloud Plane.

PostgreSQL is not deployed inside the Customer OOB Monitoring Plane as part of the product architecture.

## Responsibilities

The database is responsible for:

- Site inventory storage.
- Device inventory storage.
- Interface inventory storage.
- Historical metric retention.
- Current monitoring state.
- Alert storage and lifecycle tracking.
- Providing queryable state to the Backend API.

The database is not responsible for:

- SNMP polling.
- Telemetry transport processing.
- Alert generation logic.
- User interface rendering.
- Device configuration or console access.

## Data Model

The system is organized around the following hierarchy:

```text
Site
└── Device
    ├── Interface
    ├── Metric Samples
    └── Alerts
```

### Sites

Represents a physical location or customer site.

Examples:

- Hub Site.
- Remote Site A.
- Remote Site B.

### Devices

Represents a monitored network device.

Examples:

- Router.
- Switch.
- Firewall.

Each device belongs to exactly one site.

### Interfaces

Represents a network interface discovered through IF-MIB.

Examples:

- `GigabitEthernet0/0`
- `GigabitEthernet0/1`
- `Serial0/0`

Each interface belongs to exactly one device.

### Metric Types

Defines a metric category.

Examples:

- `cpu_utilization`
- `memory_utilization`
- `uptime_seconds`

Metric types are metadata and rarely change.

### Metric Samples

Stores time-series measurements collected from monitored devices.

Examples:

- CPU utilization.
- Memory utilization.
- Device uptime.

### Alerts

Stores alert lifecycle information.

Examples:

- Device down.
- High CPU.
- Interface down.

Alerts may reference a device, an interface, or both.

## Data Flow

```text
Cloud Ingestion
    ↓
PostgreSQL Database
    ↓
Backend API
    ↓
UI/UX Cloud Plane
```

The database only accepts monitoring writes from the Ingestion Service.

The Backend API is the only service expected to perform application read operations for frontend consumers.

## Retention Strategy

Metric samples are expected to grow significantly faster than inventory tables.

High-volume tables:

- `metric_samples`
- `interface_samples`

Low-volume tables:

- `sites`
- `devices`
- `interfaces`
- `alerts`
- `metric_types`

Future versions should implement partitioning and archival policies for time-series data.

## Availability Requirements

The database is the authoritative source of monitoring data.

Temporary database outages will prevent:

- Metric ingestion.
- Alert persistence.
- Dashboard updates.

The database should be backed up regularly and monitored for storage, replication, query health, and migration state.

## Future Scaling

Potential future enhancements:

- Monthly partitioning.
- Read replicas.
- TimescaleDB evaluation.
- Data retention policies.
- Alert correlation tables.
- User management tables.
- Dashboard preferences.
