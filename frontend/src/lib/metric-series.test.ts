import { describe, expect, it } from "vitest"
import { buildMetricChartSeries, buildMetricRateChartSeries, buildNodeDetailMetricSeries, getLatestMetricRate, getMetricRangeDelta } from "./metric-series"
import type { MetricsRow } from "./api"

function metric(overrides: Partial<MetricsRow>): MetricsRow {
  return {
    ts: 0,
    cpu: 0,
    mem_used: 0,
    mem_total: 1,
    disk_used: 0,
    disk_total: 1,
    disk_read_bytes: 0,
    disk_write_bytes: 0,
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

  it("builds node detail series with the same chart outputs as the individual builders", () => {
    const metrics = [
      metric({ ts: 100, cpu: 10, mem_used: 512, mem_total: 1024, disk_used: 1024, disk_total: 4096, disk_read_bytes: 4000, disk_write_bytes: 8000, net_rx: 1000, net_tx: 2000, uptime_seconds: 1000 }),
      metric({ ts: 105, cpu: 20, mem_used: 768, mem_total: 1024, disk_used: 1536, disk_total: 4096, disk_read_bytes: 5000, disk_write_bytes: 9500, net_rx: 1500, net_tx: 2600, uptime_seconds: 1005 }),
      metric({ ts: 110, cpu: 30, mem_used: 896, mem_total: 1024, disk_used: 2048, disk_total: 4096, disk_read_bytes: 6500, disk_write_bytes: 11000, net_rx: 2200, net_tx: 3400, uptime_seconds: 1010 }),
      metric({ ts: 500, cpu: 40, mem_used: 256, mem_total: 1024, disk_used: 3072, disk_total: 4096, disk_read_bytes: 500, disk_write_bytes: 900, net_rx: 300, net_tx: 600, uptime_seconds: 5 }),
      metric({ ts: 505, cpu: 45, mem_used: 384, mem_total: 1024, disk_used: 3200, disk_total: 4096, disk_read_bytes: 2000, disk_write_bytes: 1900, net_rx: 900, net_tx: 900, uptime_seconds: 10 }),
    ]

    const bundled = buildNodeDetailMetricSeries(metrics, 604800)

    expect(bundled.cpuData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => row.cpu, "average"))
    expect(bundled.memData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => (row.mem_total > 0 ? (row.mem_used / row.mem_total) * 100 : 0), "average"))
    expect(bundled.netRxData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => row.net_rx, "last"))
    expect(bundled.netTxData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => row.net_tx, "last"))
    expect(bundled.netRxSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.net_rx))
    expect(bundled.netTxSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.net_tx))
    expect(bundled.diskData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => (row.disk_total > 0 ? (row.disk_used / row.disk_total) * 100 : 0), "average"))
    expect(bundled.diskReadSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.disk_read_bytes ?? 0))
    expect(bundled.diskWriteSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.disk_write_bytes ?? 0))
  })

  it("only returns a latest rate when the newest samples are in the same segment", () => {
    const metrics = [
      metric({ ts: 100, net_rx: 1000, uptime_seconds: 1000 }),
      metric({ ts: 105, net_rx: 1500, uptime_seconds: 1005 }),
      metric({ ts: 110, net_rx: 200, uptime_seconds: 5 }),
    ]

    expect(getLatestMetricRate(metrics, (row) => row.net_rx)).toBeUndefined()

    const recoveredMetrics = [
      ...metrics,
      metric({ ts: 115, net_rx: 700, uptime_seconds: 10 }),
    ]

    expect(getLatestMetricRate(recoveredMetrics, (row) => row.net_rx)).toBe(100)
  })

  it("sums range deltas without crossing reboot segments", () => {
    const metrics = [
      metric({ ts: 100, net_rx: 1000, uptime_seconds: 1000 }),
      metric({ ts: 105, net_rx: 1500, uptime_seconds: 1005 }),
      metric({ ts: 110, net_rx: 200, uptime_seconds: 5 }),
      metric({ ts: 115, net_rx: 700, uptime_seconds: 10 }),
    ]

    expect(getMetricRangeDelta(metrics, (row) => row.net_rx)).toBe(1000)
    expect(getMetricRangeDelta([metrics[0]], (row) => row.net_rx)).toBeUndefined()
  })
})
