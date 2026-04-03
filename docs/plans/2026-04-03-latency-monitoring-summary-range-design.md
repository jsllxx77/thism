# Latency Monitoring Summary and Unified Range Design

**Date:** 2026-04-03

**Goal:** Extend latency monitoring with packet loss and jitter summaries, and move node-detail time ranges into one shared controller for both system metrics and latency charts.

## Context

The current latency monitoring feature stores one latency result per monitor cycle and renders a shared latency chart in node detail. Users now want two additions:

- each monitor button in node detail should show the latest `loss` and `jitter` summary under the monitor name
- node detail should use one shared time range selector so `CPU`, `memory`, `network`, `disk`, and `latency` all switch together

The user explicitly does **not** want a large card redesign. The existing clickable monitor name buttons should stay, with only a second text row added:

- first line: monitor name
- second line: `丢包: xx  波动: xx`

## Approved Direction

Keep one stored summary result per monitor cycle and compute packet loss / jitter inside the agent.

### Probe Model

For every scheduled monitor cycle, the agent runs `5` probe attempts in sequence.

For that cycle, the agent computes:

- `latency_ms`: average of successful samples
- `loss_percent`: failed probes / 5 * 100
- `jitter_ms`: standard deviation of successful latency samples

Summary rules:

- if all 5 probes fail:
  - `latency_ms = null`
  - `loss_percent = 100`
  - `jitter_ms = null`
- if fewer than 2 probes succeed:
  - `jitter_ms = null`

This keeps the storage footprint stable while matching the user-visible mental model.

## Data Model Changes

Extend latency monitor result storage and API payloads to include:

- `loss_percent`
- `jitter_ms`

Existing per-cycle latency rows stay the canonical history source. The latency chart continues to plot `latency_ms`, while the latest row for each monitor powers the button subtitle.

## Node Detail UI

### Shared Range Control

Move the node-detail time range buttons out of `MetricTabs` and into a single shared controller above both chart sections.

Shared ranges remain:

- `1h`
- `6h`
- `24h`
- `7d`
- `30d` when retention allows it

Changing this shared range updates:

- system metric tabs
- latency chart data query window
- live append retention window

### Monitor Buttons

Keep the current clickable button group and hide/show interaction.

Change button content to two rows:

- row 1: monitor name
- row 2: latest `loss` and `jitter`

Display rules:

- `丢包: 0%`
- `丢包: 40%`
- `丢包: 100%`
- `波动: 12.4 ms`
- `波动: —` when no valid jitter can be computed

### Tooltip

Failure states in the latency tooltip should use short normalized labels instead of raw socket errors:

- `超时`
- `连接被拒绝`
- `解析失败`
- `网络不可达`
- fallback: `探测失败`

## Testing Notes

- Add failing backend tests for summary result persistence and API exposure of `loss_percent` / `jitter_ms`
- Add failing collector tests for 5-probe aggregation and summary math
- Add failing frontend tests for shared range control and two-line monitor buttons
- Verify relevant Go and frontend tests, then run `make dev-restart TOKEN=thism2026 PORT=12026`
