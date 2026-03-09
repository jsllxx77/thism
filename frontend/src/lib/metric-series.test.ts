import { describe, expect, it } from "vitest"
import { buildMetricChartSeries, buildMetricRateChartSeries } from "./metric-series"
import type { MetricsRow } from "./api"

function metric(overrides: Partial<MetricsRow>): MetricsRow {
  return {
    ts: 0,
    cpu: 0,
    mem_used: 0,
    mem_total: 1,
    disk_used: 0,
    disk_total: 1,
    net_rx: 0,
    net_tx: 0,
    uptime_seconds: 0,
    ...overrides,
  }
}

describe("metric-series", () => {
  it("breaks chart lines when there is a large collection gap", () => {
    const metrics = [
      metric({ ts: 100, cpu: 10, uptime_seconds: 1000 }),
      metric({ ts: 105, cpu: 20, uptime_seconds: 1005 }),
      metric({ ts: 110, cpu: 30, uptime_seconds: 1010 }),
      metric({ ts: 500, cpu: 40, uptime_seconds: 1400 }),
    ]

    expect(buildMetricChartSeries(metrics, 604800, (row) => row.cpu, "average")).toEqual([
      { ts: 100, value: 10 },
      { ts: 105, value: 20 },
      { ts: 110, value: 30 },
      { ts: 499, value: null },
      { ts: 500, value: 40 },
    ])
  })

  it("breaks rate lines when a node reboots instead of connecting across the restart", () => {
    const metrics = [
      metric({ ts: 100, net_rx: 1000, uptime_seconds: 1000 }),
      metric({ ts: 105, net_rx: 1500, uptime_seconds: 1005 }),
      metric({ ts: 110, net_rx: 200, uptime_seconds: 5 }),
      metric({ ts: 115, net_rx: 700, uptime_seconds: 10 }),
    ]

    expect(buildMetricRateChartSeries(metrics, 604800, (row) => row.net_rx)).toEqual([
      { ts: 105, value: 100 },
      { ts: 114, value: null },
      { ts: 115, value: 100 },
    ])
  })
})
