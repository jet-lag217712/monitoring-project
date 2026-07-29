# Appliance decision 3: isolated NoAuth ISO edition

## Status

Accepted for the `v2.0.0` NoAuth appliance artifact.

## Decision

Equate ships a separate AMD64 installer named
`Equate-Appliance-NoAuth-v2.0.0-amd64.iso` for strictly air-gapped
environments. Its internal release version is `2.0.0-noauth`. The edition
sets API authentication to disabled, serves the dashboard without a sign-in
screen, and makes no Google Identity Services request. The local console omits
Workspace-domain configuration but retains TLS, encrypted SNMP community, and
discovery setup.

## Consequences

Anyone able to reach the appliance HTTPS endpoint can read dashboard/API data.
The deployment must therefore be isolated and access-controlled at the network
boundary. The NoAuth edition is independently built and must not be confused
with, installed over, or updated from the standard Google-authenticated
appliance release line.
