# Latency Monitoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ICMP, TCP, and HTTP latency monitors that can be configured in settings, automatically assigned to new nodes, and visualized as a multi-series latency chart in node detail.

**Architecture:** Persist monitors as global configuration objects with explicit node bindings and auto-assignment for future nodes. Push effective monitor configuration from the server to agents, let agents execute probes locally on each node, store every result in SQLite, and render assigned monitor histories as a single multi-series chart on the node detail page.

**Tech Stack:** Go, SQLite, Chi, Gorilla WebSocket, React 19, TypeScript, Recharts, Vitest, Testing Library

---

### Task 1: Lock in store behavior with failing tests

**Files:**
- Modify: `internal/store/store_test.go`
- Modify: `internal/models/types.go` if shared types are needed by tests first

**Step 1: Write the failing test**

- Add tests for:
  - creating a monitor with selected node bindings
  - auto-assigning a newly registered node to existing auto-assigned monitors
  - inserting latency results and querying them by node and time range
  - deleting a monitor and removing its bindings and results

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run 'Test(StoreLatencyMonitors|NewNodeAutoAssignedLatencyMonitors|LatencyResultsQuery|DeleteLatencyMonitorCascades)'`

Expected: FAIL because the schema and store methods do not exist yet.

### Task 2: Implement latency monitor storage

**Files:**
- Modify: `internal/store/store.go`
- Modify: `internal/models/types.go`

**Step 1: Write minimal implementation**

- Add SQLite tables and indexes for `monitor_items`, `monitor_item_nodes`, and `monitor_results`
- Add model types for latency monitor metadata and result rows
- Add store methods for monitor CRUD, node binding replacement, auto-assignment for new nodes, result insert, and result query
- Extend node registration flow support by exposing a store helper that attaches new nodes to auto-assigned monitors

**Step 2: Run test to verify it passes**

Run: `go test ./internal/store -run 'Test(StoreLatencyMonitors|NewNodeAutoAssignedLatencyMonitors|LatencyResultsQuery|DeleteLatencyMonitorCascades)'`

Expected: PASS

### Task 3: Lock in API behavior with failing tests

**Files:**
- Modify: `internal/api/api_test.go`
- Modify: `internal/api/guest_access_regression_test.go` if access behavior needs coverage

**Step 1: Write the failing test**

- Add API tests for:
  - listing monitors
  - creating, updating, and deleting a monitor
  - returning node-scoped latency history
  - registering a node and verifying auto-assignment
  - broadcasting latency result messages to dashboard subscribers

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run 'Test(LatencyMonitorCRUD|LatencyMonitorNodeHistory|RegisterNodeAutoAssignsLatencyMonitors|DashboardWebSocketBroadcastsLatencyResults)'`

Expected: FAIL because the endpoints and message handling do not exist yet.

### Task 4: Implement latency monitor APIs and WebSocket handling

**Files:**
- Modify: `internal/api/api.go`
- Modify: `internal/hub/hub.go` if helper behavior needs adjustment
- Modify: `internal/models/types.go`
- Modify: `internal/store/store.go`

**Step 1: Write minimal implementation**

- Add admin routes for latency monitor CRUD
- Add admin node-detail route for latency history
- Update node registration to auto-assign the new node before returning success
- Add agent WebSocket payload handling for latency result submissions
- Broadcast live latency result messages to dashboard subscribers
- Send effective monitor configuration to agents when they connect and when monitor assignments change

**Step 2: Run test to verify it passes**

Run: `go test ./internal/api -run 'Test(LatencyMonitorCRUD|LatencyMonitorNodeHistory|RegisterNodeAutoAssignsLatencyMonitors|DashboardWebSocketBroadcastsLatencyResults)'`

Expected: PASS

### Task 5: Lock in collector behavior with failing tests

**Files:**
- Create: `internal/collector/latency_monitor_test.go`
- Modify: `internal/collector/collector_test.go` if shared helpers fit better there

**Step 1: Write the failing test**

- Add tests for:
  - replacing the active monitor config from a server message
  - running ICMP, TCP, and HTTP probes through injectable probe functions
  - respecting monitor intervals
  - emitting latency result messages with success and failure payloads

**Step 2: Run test to verify it fails**

Run: `go test ./internal/collector -run 'Test(CollectorReplacesLatencyMonitorConfig|CollectorLatencyProbeScheduling|CollectorLatencyProbeResultPayloads)'`

Expected: FAIL because the collector has no latency monitor scheduler or payload handling.

### Task 6: Implement collector latency scheduling and probes

**Files:**
- Modify: `internal/collector/collector.go`
- Create: `internal/collector/latency_monitor.go` if extraction keeps the scheduler readable
- Modify: `internal/models/types.go`

**Step 1: Write minimal implementation**

- Add WebSocket message handling for server-sent latency monitor configuration
- Maintain an in-memory monitor set keyed by monitor ID
- Schedule probes per monitor interval without blocking the main metrics loop
- Implement injectable ICMP, TCP, and HTTP probe functions
- Emit typed latency result messages back to the server

**Step 2: Run test to verify it passes**

Run: `go test ./internal/collector -run 'Test(CollectorReplacesLatencyMonitorConfig|CollectorLatencyProbeScheduling|CollectorLatencyProbeResultPayloads)'`

Expected: PASS

### Task 7: Lock in settings UI behavior with failing tests

**Files:**
- Create: `frontend/src/components/settings/latency-monitors-card.test.tsx`
- Modify: `frontend/src/pages/settings-states.test.tsx` if settings load state coverage needs updates

**Step 1: Write the failing test**

- Add tests that expect:
  - the latency monitors card to render monitor rows
  - the create flow to default all nodes to selected
  - node deselection to affect the submitted payload
  - success and error states to render correctly

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- src/components/settings/latency-monitors-card.test.tsx src/pages/settings-states.test.tsx`

Expected: FAIL because the settings UI and API bindings do not exist yet.

### Task 8: Implement settings latency monitor management

**Files:**
- Create: `frontend/src/components/settings/LatencyMonitorsCard.tsx`
- Modify: `frontend/src/pages/Settings.tsx`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/i18n/messages.ts`
- Modify: `frontend/src/i18n/messages.en.ts`
- Modify: `frontend/src/i18n/messages.zh-CN.ts`

**Step 1: Write minimal implementation**

- Add typed frontend API helpers for monitor CRUD and node-scoped latency history
- Implement the settings card and form
- Default current nodes to selected on create
- Pass the settings page node list into the latency monitors card
- Keep the existing settings visual language and mobile behavior

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm test -- src/components/settings/latency-monitors-card.test.tsx src/pages/settings-states.test.tsx`

Expected: PASS

### Task 9: Lock in node-detail visualization with failing tests

**Files:**
- Create: `frontend/src/components/node-detail/latency-monitor-chart.test.tsx`
- Modify: `frontend/src/pages/node-detail-states.test.tsx`

**Step 1: Write the failing test**

- Add tests that expect:
  - node detail to request latency history
  - one unified chart to render multiple monitor series
  - clicking a legend item to hide one series
  - an empty state when the node has no assigned monitors
  - live WebSocket latency updates to append to the correct series

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- src/components/node-detail/latency-monitor-chart.test.tsx src/pages/node-detail-states.test.tsx`

Expected: FAIL because node detail does not load or render latency monitor data.

### Task 10: Implement node-detail latency chart

**Files:**
- Create: `frontend/src/components/node-detail/LatencyMonitorChart.tsx`
- Modify: `frontend/src/pages/NodeDetail.tsx`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/components/MetricsChart.tsx` only if a shared multi-series primitive is cleaner than a dedicated chart
- Modify: `frontend/src/i18n/messages.ts`
- Modify: `frontend/src/i18n/messages.en.ts`
- Modify: `frontend/src/i18n/messages.zh-CN.ts`

**Step 1: Write minimal implementation**

- Load node-scoped latency history alongside existing detail data
- Render a dedicated multi-series latency chart section
- Implement clickable legend toggles for series visibility
- Append live latency result messages from the dashboard WebSocket
- Show failures as gaps and show a clear empty state when no monitor exists

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm test -- src/components/node-detail/latency-monitor-chart.test.tsx src/pages/node-detail-states.test.tsx`

Expected: PASS

### Task 11: Run focused regression coverage

**Files:**
- No source changes expected

**Step 1: Run backend verification**

Run: `go test ./internal/store ./internal/api ./internal/collector`

Expected: PASS

**Step 2: Run frontend verification**

Run: `cd frontend && npm test -- src/components/settings/latency-monitors-card.test.tsx src/components/node-detail/latency-monitor-chart.test.tsx src/pages/settings-states.test.tsx src/pages/node-detail-states.test.tsx`

Expected: PASS

### Task 12: Refresh the local deployment

**Files:**
- No source changes expected

**Step 1: Rebuild and restart**

Run: `make dev-restart TOKEN=thism2026 PORT=12026`

Expected: frontend assets rebuilt, server binary rebuilt, runtime restarted on port `12026`
