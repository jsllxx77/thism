# Latency Monitoring Summary and Unified Range Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add packet loss and jitter summaries to latency monitoring and move node-detail time ranges into one shared controller that updates both system metrics and latency charts.

**Architecture:** Keep one summary row per monitor cycle and calculate `latency_ms`, `loss_percent`, and `jitter_ms` inside the agent from 5 probe attempts. Expose those fields through store and API layers, then update node detail to use one shared range control and compact two-line monitor buttons showing the latest loss / jitter summary.

**Tech Stack:** Go, SQLite, Chi, Gorilla WebSocket, React 19, TypeScript, Recharts, Vitest, Testing Library

---

### Task 1: Lock in summary persistence with failing store tests

**Files:**
- Modify: `internal/store/store_test.go`
- Modify: `internal/models/types.go` if tests need shared fields first

**Step 1: Write the failing test**

- Add store tests that:
  - insert a latency result with `loss_percent` and `jitter_ms`
  - query node latency history and verify both fields round-trip
  - verify delete cascades still remove summary results

**Step 2: Run test to verify it fails**

Run: `go test ./internal/store -run 'Test(LatencyResultSummaryRoundTrip|DeleteLatencyMonitorCascades)'`

Expected: FAIL because summary fields do not exist in the schema or model yet.

### Task 2: Implement summary result storage

**Files:**
- Modify: `internal/models/types.go`
- Modify: `internal/store/store.go`

**Step 1: Write minimal implementation**

- Add `loss_percent` and `jitter_ms` to latency result models
- Migrate `monitor_results` with nullable summary columns
- Update insert/query logic to persist and scan the new fields

**Step 2: Run test to verify it passes**

Run: `go test ./internal/store -run 'Test(LatencyResultSummaryRoundTrip|DeleteLatencyMonitorCascades)'`

Expected: PASS

### Task 3: Lock in API exposure with failing tests

**Files:**
- Modify: `internal/api/api_test.go`
- Modify: `internal/api/agent_ws_broadcast_test.go` if live payload coverage fits there

**Step 1: Write the failing test**

- Add API tests that expect:
  - node latency history to include `loss_percent` and `jitter_ms`
  - live latency result broadcasts to include the summary fields

**Step 2: Run test to verify it fails**

Run: `go test ./internal/api -run 'Test(LatencyMonitorNodeHistory|DashboardWebSocketBroadcastsLatencyResults)'`

Expected: FAIL because the response payloads do not include the summary fields yet.

### Task 4: Implement API summary exposure

**Files:**
- Modify: `internal/api/api.go`
- Modify: `internal/models/types.go`
- Modify: `internal/store/store.go`

**Step 1: Write minimal implementation**

- Pass summary fields through node history responses
- Pass summary fields through live `latency_result` broadcasts unchanged

**Step 2: Run test to verify it passes**

Run: `go test ./internal/api -run 'Test(LatencyMonitorNodeHistory|DashboardWebSocketBroadcastsLatencyResults)'`

Expected: PASS

### Task 5: Lock in collector aggregation with failing tests

**Files:**
- Modify: `internal/collector/latency_monitor_test.go`

**Step 1: Write the failing test**

- Add collector tests that expect one monitor cycle to:
  - run 5 probe attempts
  - compute average latency from successful probes
  - compute packet loss percentage from failed probes
  - compute jitter as standard deviation when 2+ successes exist
  - return `jitter_ms = nil` when too few successes exist

**Step 2: Run test to verify it fails**

Run: `go test ./internal/collector -run 'Test(CollectorLatencyProbeAggregation|CollectorLatencyProbeResultPayloads)'`

Expected: FAIL because the collector still reports one probe attempt per cycle with no summary math.

### Task 6: Implement collector 5-probe aggregation

**Files:**
- Modify: `internal/collector/collector.go`
- Modify: `internal/collector/latency_monitor.go`
- Modify: `internal/models/types.go`

**Step 1: Write minimal implementation**

- Run 5 probe attempts per due monitor cycle
- Compute `latency_ms`, `loss_percent`, and `jitter_ms`
- Emit only one summary `latency_result` message per cycle

**Step 2: Run test to verify it passes**

Run: `go test ./internal/collector -run 'Test(CollectorLatencyProbeAggregation|CollectorLatencyProbeResultPayloads)'`

Expected: PASS

### Task 7: Lock in node-detail UI behavior with failing frontend tests

**Files:**
- Modify: `frontend/src/components/node-detail/latency-monitor-chart.test.tsx`
- Modify: `frontend/src/pages/node-detail-states.test.tsx`
- Modify: `frontend/src/components/node-detail/node-detail-metrics.test.tsx` if shared range expectations fit there

**Step 1: Write the failing test**

- Add tests that expect:
  - a shared time range control above charts
  - changing the shared range refetches both metrics and latency history
  - monitor buttons render a second line with latest loss and jitter
  - tooltip failure copy is normalized

**Step 2: Run test to verify it fails**

Run: `cd frontend && npm test -- src/components/node-detail/latency-monitor-chart.test.tsx src/pages/node-detail-states.test.tsx src/components/node-detail/node-detail-metrics.test.tsx`

Expected: FAIL because range controls are still owned by `MetricTabs` and monitor buttons have no summary subtitle.

### Task 8: Implement shared range control and button summaries

**Files:**
- Modify: `frontend/src/pages/NodeDetail.tsx`
- Modify: `frontend/src/components/node-detail/MetricTabs.tsx`
- Modify: `frontend/src/components/node-detail/LatencyMonitorChart.tsx`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/i18n/messages.en.ts`
- Modify: `frontend/src/i18n/messages.zh-CN.ts`
- Modify: `frontend/src/i18n/messages.ts` if legacy key mapping is needed

**Step 1: Write minimal implementation**

- Move range buttons to one shared controller in node detail
- Keep `MetricTabs` focused on tab switching only
- Surface latest `loss_percent` and `jitter_ms` in each monitor button
- Normalize tooltip failure text into short labels

**Step 2: Run test to verify it passes**

Run: `cd frontend && npm test -- src/components/node-detail/latency-monitor-chart.test.tsx src/pages/node-detail-states.test.tsx src/components/node-detail/node-detail-metrics.test.tsx`

Expected: PASS

### Task 9: Run focused regression verification

**Files:**
- No source changes expected

**Step 1: Run backend verification**

Run: `go test ./internal/store ./internal/api ./internal/collector`

Expected: PASS

**Step 2: Run frontend verification**

Run: `cd frontend && npm test -- src/components/node-detail/latency-monitor-chart.test.tsx src/pages/node-detail-states.test.tsx src/components/node-detail/node-detail-metrics.test.tsx src/pages/settings-states.test.tsx src/components/settings/latency-monitors-card.test.tsx`

Expected: PASS

### Task 10: Refresh the local deployment

**Files:**
- No source changes expected

**Step 1: Rebuild and restart**

Run: `make dev-restart TOKEN=thism2026 PORT=12026`

Expected: frontend assets rebuilt, server binary rebuilt, runtime restarted on port `12026`
