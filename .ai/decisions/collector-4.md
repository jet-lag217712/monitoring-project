## collector - 4

### Primary Service

SNMP Collector

### Choice Made

Phase 3 evaluates health only after a completed due-device `pollAll` batch.
Outcomes are buffered during the batch and committed only when every due device
finishes. Cancellation before completion discards the buffered outcomes and
leaves the health ledger and health gauges unchanged.

`PollOutcome.ObservedAt` is captured immediately after the device poll pipeline
completes, not at schedule/start time.

The tracker re-evaluates the full active inventory after each committed batch,
using retained outcomes for devices that were not due. Upstream IDs are
deduplicated before DAG traversal. Events from `ApplyBatch` are sorted by
device ID; unavailable and root-cause ID lists are sorted unique.

Health transitions emitted in Phase 3 are only `initial`, `entered`, and
`recovered`. `reasserted` is never emitted. Events are produced for initial
assignment, terminal state changes, reason changes, and recovery. Pending
failures below the consecutive-failure threshold retain the prior terminal
state and do not emit. Unchanged state+reason pairs do not emit.

Primary temperature is the maximum valid vendor temperature component. Nil,
NaN, and negative values are ignored. When no valid temperature exists, a
successful poll is Healthy with reason `reachable`.

On configuration reload, retained devices keep failure count, terminal state,
last successful observation timestamp, and health evidence. Removed devices are
pruned. New devices start unobserved.

Phase 3 owns the health state machine, failure ledger, dependency correlation,
local health events, health/dependency metrics, and `/readyz`. Phase 4 owns
MQTT v2 health-event serialization, envelope metadata, health publishing, and
heartbeat publishing.

### Status

Accepted — Phase 3 implementation decision.
