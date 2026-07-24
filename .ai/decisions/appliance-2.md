# Appliance decision 2: Google sessions and SNMP-only setup

## Status

Accepted for appliance release `v2.0.0`.

## Decision

The appliance authenticates dashboard users with Google Identity Services and
server-issued secure sessions. Google ID tokens are exchanged only at the
appliance API and are never persisted by the dashboard. A token must have the
Equate-managed client ID as its audience, a verified email, and a hosted-domain
(`hd`) claim matching the exact domain allow-list entered through the local
SNMP setup TUI. No-`hd` and consumer identities are rejected.

All operator-entered appliance configuration is limited to that TUI: Workspace
domains, default and discovery SNMP communities, discovery CIDRs/rate limits,
temperature threshold, and TLS import. TLS media is read-only and labeled
`EQUATE_TLS`; it supplies `tls.crt` and `tls.key`. The certificate must have a
matching key and exactly one concrete `*.equatecloud.tech` SAN. The appliance
uses that SAN as its hostname. SNMP communities and the TLS private key are
encrypted at rest in `/etc/equate/secrets` and rendered only under
`/run/equate` during boot.

The appliance owns Ethernet DHCP through `systemd-networkd` and starts its
five-service Compose release after `network-online`. UI is published only on
80 and 443. Until a leaf certificate is imported, a transient boot-only
self-signed certificate keeps the local UI reachable.

## Consequences

Customers must provide internal DNS and client-CA trust for the assigned
appliance hostname. Equate operations must register the exact HTTPS origin in
the managed Google client. A real Google client secret, redirect callback, and
external admin portal are not required for this GIS ID-token flow.
