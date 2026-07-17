# Monitoring Requirements — SNMP Collector v2

## Purpose

The dashboard is a read-only presentation layer for PostgreSQL-backed monitoring state delivered by the Backend API. It does not own collector configuration, health rules, or any source-of-truth state.

## Required views and data

The all-sites overview must show monitored-device totals and separate healthy, warning, direct-critical, and dependency-impacted counts, with live/demo indication and last-update state. Site detail must retain searchable/dense device visibility and include current health state/reason.

Device detail must render identity (vendor, model, serial, SNMP identity, detected profile/capabilities), uptime, CPU, memory, primary temperature and history, individual temperature and power components/status, and health/dependency evidence. Interface views must render identity/metadata, admin/oper status, counters/errors, speed, and traffic history. The API supplies these values; the frontend does not query collectors or recompute them.

## Health presentation

The API preserves numeric device status compatibility: `0` Unknown, `1` Healthy, `2` Warning, `3` Critical. It also supplies explicit `status_reason`, `upstream_device_ids`, `unavailable_upstream_device_ids`, and `root_cause_device_ids` when relevant.

Unknown is an explicit `upstream_unreachable` evidence state, visually distinct from Critical and excluded from direct-critical counts. Warning represents a reachable device at/above its collector temperature threshold. CPU, memory, and power are live display metrics in v2, not dashboard-side health thresholds.

## Behavior, accessibility, and security

Five-second polling remains the default. Refresh updates overview, selected site/device details, and data mode state. If live data is unavailable, demo fallback must remain visibly labeled and never masquerade as live telemetry. The UI needs accessible state labels/color treatment, readable dense tables, explicit loading/empty/error states, and keyboard-accessible navigation.

The dashboard cannot edit inventory, thresholds, dependencies, discovery candidates, or credentials; those are authenticated local collector-TUI operations. It must not expose raw credentials, TLS material, local paths, or direct infrastructure access.
