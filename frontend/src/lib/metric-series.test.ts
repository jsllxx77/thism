import { describe, expect, it } from "vitest"
import {
  buildMetricChartSeries,
  buildMetricDeltaChartSeries,
  buildMetricRateChartSeries,
  buildNodeDetailMetricSeries,
  getLatestMetricRate,
  getMetricRangeDelta,
} from "./metric-series"
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
    load1: 0,
    load5: 0,
    load15: 0,
    cpu_iowait_percent: 0,
    cpu_steal_percent: 0,
    pressure_cpu_some: 0,
    pressure_memory_some: 0,
    pressure_memory_full: 0,
    pressure_io_some: 0,
    pressure_io_full: 0,
    swap_used: 0,
    swap_total: 1,
    swap_in: 0,
    swap_out: 0,
    oom_kills: 0,
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
    expect(bundled.netRxData).toEqual(buildMetricDeltaChartSeries(metrics, 604800, (row) => row.net_rx))
    expect(bundled.netTxData).toEqual(buildMetricDeltaChartSeries(metrics, 604800, (row) => row.net_tx))
    expect(bundled.netRxSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.net_rx))
    expect(bundled.netTxSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.net_tx))
    expect(bundled.diskData).toEqual(buildMetricChartSeries(metrics, 604800, (row) => (row.disk_total > 0 ? (row.disk_used / row.disk_total) * 100 : 0), "average"))
    expect(bundled.diskReadSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.disk_read_bytes ?? 0))
    expect(bundled.diskWriteSpeedData).toEqual(buildMetricRateChartSeries(metrics, 604800, (row) => row.disk_write_bytes ?? 0))
  })

  it("builds node pressure and memory pressure series", () => {
    const metrics = [
      metric({
        ts: 100,
        load1: 1,
        load5: 1.5,
        load15: 2,
        cpu_iowait_percent: 3,
        cpu_steal_percent: 1,
        pressure_cpu_some: 0.5,
        pressure_memory_some: 0.7,
        pressure_memory_full: 0.1,
        pressure_io_some: 0.9,
        pressure_io_full: 0.2,
        swap_used: 100,
        swap_total: 1000,
        swap_in: 10,
        swap_out: 20,
        oom_kills: 0,
        uptime_seconds: 1000,
      }),
      metric({
        ts: 110,
        load1: 2,
        load5: 2.5,
        load15: 3,
        cpu_iowait_percent: 6,
        cpu_steal_percent: 2,
        pressure_cpu_some: 1.5,
        pressure_memory_some: 1.7,
        pressure_memory_full: 0.3,
        pressure_io_some: 1.9,
        pressure_io_full: 0.4,
        swap_used: 200,
        swap_total: 1000,
        swap_in: 30,
        swap_out: 50,
        oom_kills: 1,
        uptime_seconds: 1010,
      }),
    ]

    const bundled = buildNodeDetailMetricSeries(metrics, 3600)

    expect(bundled.load1Data).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.load1 ?? 0, "average"))
    expect(bundled.cpuIOWaitData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.cpu_iowait_percent ?? 0, "average"))
    expect(bundled.cpuStealData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.cpu_steal_percent ?? 0, "average"))
    expect(bundled.pressureCPUSomeData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.pressure_cpu_some ?? 0, "average"))
    expect(bundled.pressureMemorySomeData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.pressure_memory_some ?? 0, "average"))
    expect(bundled.pressureMemoryFullData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.pressure_memory_full ?? 0, "average"))
    expect(bundled.pressureIOSomeData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.pressure_io_some ?? 0, "average"))
    expect(bundled.pressureIOFullData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.pressure_io_full ?? 0, "average"))
    expect(bundled.swapData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => ((row.swap_total ?? 0) > 0 ? ((row.swap_used ?? 0) / (row.swap_total ?? 1)) * 100 : 0), "average"))
    expect(bundled.swapInSpeedData).toEqual(buildMetricRateChartSeries(metrics, 3600, (row) => row.swap_in ?? 0))
    expect(bundled.swapOutSpeedData).toEqual(buildMetricRateChartSeries(metrics, 3600, (row) => row.swap_out ?? 0))
    expect(bundled.oomKillsData).toEqual(buildMetricChartSeries(metrics, 3600, (row) => row.oom_kills ?? 0, "last"))
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

  it("builds range traffic charts from cumulative counter deltas", () => {
    const metrics = [
      metric({ ts: 100, net_rx: 1000, uptime_seconds: 1000 }),
      metric({ ts: 105, net_rx: 1500, uptime_seconds: 1005 }),
      metric({ ts: 110, net_rx: 200, uptime_seconds: 5 }),
      metric({ ts: 115, net_rx: 700, uptime_seconds: 10 }),
    ]

    expect(buildMetricDeltaChartSeries(metrics, 604800, (row) => row.net_rx)).toEqual([
      { ts: 100, value: 0 },
      { ts: 105, value: 500 },
      { ts: 109, value: null },
      { ts: 110, value: 500 },
      { ts: 115, value: 1000 },
    ])
  })
})
