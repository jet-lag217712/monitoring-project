# SNMP Collector v2 Architecture

## Purpose

The collector is an operator-managed, local SNMPv2c service in the Customer OOB Monitoring Plane. It concurrently discovers (only when explicitly invoked) and polls a validated multi-device inventory, evaluates reachability-aware health, and reliably publishes v2 telemetry through outbound MQTT/TLS.

## Boundary and non-responsibilities

The collector owns SNMP polling-path evidence, local inventory/configuration, the SQLite outbox, and local operator controls. It does not own cloud persistence, dashboard/API requests, alert workflows, device configuration, console access, or an inbound management channel. SNMPv3, write operations, automatic enrollment, automatic dependency activation, and cloud-side credential editing are out of scope.

Its public operational surfaces are scrape-only `/metrics`, liveness `/healthz`, readiness `/readyz`, and a localhost/Unix-socket status/control service. The Bubble Tea TUI is a local client of that socket; it is not exposed through HTTP.

## Inventory and configuration

The collector merges a read-only static YAML deployment inventory with a separate TUI-managed inventory. Static entries are authoritative when the same device ID appears in both sources; unique managed entries are appended. Duplicate IDs within a source and duplicate host/IP identities in the active merged result are rejected. The collector atomically activates validated configuration snapshots. The TUI writes only the managed file using restrictive permissions, fsync, atomic rename, and reload.

Every device uses `version: 2c`, a `community_env` reference (never a plaintext community), optional per-device poll overrides, optional `temperature_warning_c`, interface-filter rules, and zero or more `upstream_device_ids`. Dependency IDs reference stable devices, never IPs. The merged graph must be an acyclic DAG: duplicate/self/missing references and cycles are rejected.

`collector validate -config …` validates YAML, environment-reference names, TLS/MQTT settings, file permissions, inventory, profile names, polling and discovery bounds, regular expressions, policy bounds, heartbeat interval, and the dependency DAG. SIGHUP and a later local TUI action reload transactionally: an invalid reload retains the active snapshot; existing polls finish on their captured snapshot. Site/collector identity, admin, publisher, MQTT, and buffer settings remain startup-only during Phase 1.

## Polling, profiles, and health

A bounded, context-aware worker pool independently polls every configured device. Core SNMPv2-MIB and IF-MIB collection always runs first. Cisco and Arista profiles are selected by authoritative `sysObjectID` (normalized `sysDescr` is a logged fallback) and add fixture-tested CPU, memory, temperature, and power mappings. Profile failures preserve core readings and are reported as profile-collection failures.

Interface filtering occurs after inventory read and before emission. Ordered rules can match index, name/alias regex, IF-MIB type, and admin/operational status. Defaults exclude virtual interfaces; explicit includes can restore one, and a final exclusion wins.

Health states are `healthy`, `warning`, `critical`, and `unknown`. A successful poll is Healthy or Warning according to the active temperature threshold (default 65°C). A failed root, or a failed device with a responding upstream, becomes Critical after the configured failure threshold (default two). A failed dependent is Unknown with reason `upstream_unreachable` only when every configured upstream is Critical or upstream-unreachable. When an upstream is still pending, the device retains its prior terminal state. The collector still polls every dependent; a responding dependent is Healthy/Warning regardless of failed upstreams.

## Discovery and local administration

`collector discover` and the TUI discovery workflow scan configured CIDR allowlists only. Discovery uses SNMPv2c to read `sysObjectID`, `sysName`, and `sysDescr`, does not use ICMP or device writes, and applies target, timeout, retry, concurrency, and token-bucket rate/burst bounds before every probe. Candidates are reviewable and may be rejected, exported as YAML, or explicitly accepted into managed inventory; discovery never silently activates a device or an LLDP-derived edge.

The TUI provides inventory, device/interface, discovery, thresholds, transport, and configuration views. Mutations require local OS access, explicit confirmation, durable audit entries without secrets, and a save/reload step. It cannot modify static YAML.

## Delivery and observability

The collector retains MQTT/TLS QoS 1, a durable SQLite outbox, and at-least-once delivery. Device, interface, health, and heartbeat events use the v2 contract in [`contracts.md`](contracts.md). The periodic heartbeat carries collector identity/build/runtime information and outbox depth without secrets.

`/readyz` requires a valid active configuration, available buffer, and usable publisher (connected MQTT in MQTT mode); polling and buffering continue during a broker outage. Logs are structured and redact communities, certificates, credentials, payloads, and secret-derived values. Metrics retain polling/buffer/MQTT coverage and add bounded-cardinality coverage for configuration reload, profiles, discovery, interface selection, health/dependency impact, heartbeat, and readiness.

## Phase 0 contract artifacts

The formal envelope and event schemas are in
[`docs/schemas/snmp-collector-v2/`](../schemas/snmp-collector-v2/). The
collector decision record is [`collector-1.md`](../../.ai/decisions/collector-1.md),
and Cisco/Arista evidence is tracked in
[`snmp-vendor-mappings.md`](snmp-vendor-mappings.md). These artifacts define
the producer boundary before runtime profile or health code is added.
