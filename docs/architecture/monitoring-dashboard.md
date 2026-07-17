# Monitoring Dashboard Architecture — v2

The dashboard is a read-only UI/UX Cloud Plane frontend. It consumes Backend API responses only; it never accesses PostgreSQL, MQTT, SNMP, collector inventory, local TUI controls, or credentials.

V2 site summaries show healthy, warning, direct-critical, and dependency-impacted counts separately. Device rows and detail render an explicit Unknown treatment for `upstream_unreachable`, including the recorded unavailable-upstream and root-cause context. Unknown is not displayed as Critical.

Device detail presents current identity (vendor, model, serial, SNMP identity, profile/capabilities), uptime, CPU, memory, primary temperature and history, individual temperature/power components and status, health reason, and dependency evidence. Interface views present selected interface metadata, admin/oper status, counters/errors, speed, and traffic history. Existing site overview, detail, alert, live/demo indication, accessible dense layout, and five-second polling remain unless API freshness requirements change.

The dashboard visualizes collector-evaluated health; it does not independently apply CPU, memory, power, or topology rules. It must keep all display states, empty data, loading, and demo fallback visibly distinguishable.
