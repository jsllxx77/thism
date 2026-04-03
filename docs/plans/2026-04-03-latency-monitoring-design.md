# Latency Monitoring Design

**Date:** 2026-04-03

**Goal:** Add configurable ICMP, TCP, and HTTP latency monitoring items that run from selected nodes, with monitor management in settings and per-node latency charts in node detail.

## Context

The current product only stores and visualizes node resource metrics. It does not have a way to measure latency from a node to an external target such as an ISP endpoint, TCP socket, or HTTP URL.

The requested feature is not a node property. It is a reusable monitoring item:

- A monitor has a human-readable name such as `Guangdong Telecom IPv4`
- A monitor has a type: `icmp`, `tcp`, or `http`
- A monitor has a target value such as `gd-ct-v4.ip.zstaticcdn.com:80`
- A monitor has a custom interval
- A monitor applies to many nodes

Creation behavior was clarified:

- New monitors default to all current nodes selected
- The user may deselect nodes to scope a monitor to specific nodes
- New nodes registered later should automatically join existing monitors

Visualization behavior was also clarified:

- Node detail should show one unified latency line chart
- Every latency monitor assigned to the current node appears as one series
- Clicking a monitor name in the chart legend hides or shows that series

## Approved Direction

Implement latency monitoring as a global monitor catalog with explicit node bindings and node-local execution.

### Data Model

Add three persistence layers:

- `monitor_items`
  Stores monitor metadata: `id`, `name`, `type`, `target`, `interval_seconds`, `auto_assign_new_nodes`, timestamps
- `monitor_item_nodes`
  Stores monitor-to-node assignments: `monitor_id`, `node_id`
- `monitor_results`
  Stores each probe result: `monitor_id`, `node_id`, `ts`, `latency_ms`, `success`, `error_message`

New node registration should automatically bind the node to every monitor where `auto_assign_new_nodes = true`.

## Execution Model

Latency is measured by the agent running on each assigned node.

- `ICMP`
  Probe the target host or IP and record round-trip latency
- `TCP`
  Measure TCP connect latency to the configured `host:port`
- `HTTP`
  Measure request latency to the configured URL and record the time to first response

The server owns configuration and storage only:

- settings UI creates or updates monitors
- server persists monitors and node bindings
- server pushes the effective monitor list to online agents
- agents schedule local probes by interval
- agents send probe results back over WebSocket
- server stores results and broadcasts them to dashboard clients

Failed probes are stored with `success = false`, `latency_ms = null`, and a short error string. The frontend chart should render failures as gaps rather than fake zeroes.

## API Shape

Add admin APIs for monitor management:

- list monitors
- create monitor
- update monitor
- delete monitor

Add node-detail APIs for latency history:

- list monitor metadata assigned to a node
- query latency results for a node over a time range

The existing dashboard WebSocket should also broadcast new latency result messages so node detail updates live without polling.

## UI Notes

### Settings

Add a `Latency Monitors` card to the settings page.

The card should provide:

- monitor list with name, type, target, interval, and assigned node count
- create and edit dialog or inline form
- node multi-select with all current nodes checked by default on create
- delete action

The styling should reuse the current settings card language instead of introducing a new visual system.

### Node Detail

Add a latency chart section below the existing system metrics card group.

The chart should:

- overlay all assigned monitor series for the current node
- show a clickable legend for series visibility
- keep time range controls aligned with existing node-detail metric ranges
- show an empty state when the node has no assigned monitors

## Constraints

- Keep the first version focused on latency only
- Do not add retry policy, threshold alerting, custom HTTP headers, or status-code assertions in this pass
- Preserve current settings and node-detail layouts on mobile and desktop
- Keep guest access behavior unchanged unless a test proves otherwise

## Testing Notes

- Start with failing store tests for monitor persistence, assignment, auto-assignment, and result queries
- Add failing API tests for monitor CRUD, node-detail latency history, and live message handling
- Add failing collector tests for config replacement, per-type probing, scheduling, and result emission
- Add failing frontend tests for the settings card and node-detail multi-series visibility behavior
- Verify targeted Go and frontend tests, then run `make dev-restart TOKEN=thism2026 PORT=12026`
