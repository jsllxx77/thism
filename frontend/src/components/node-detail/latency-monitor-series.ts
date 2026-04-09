import type { LatencyMonitor, LatencyMonitorResult } from "../../lib/api"

export type SeriesPoint = {
  ts: number
  value: number | null
  success: boolean
  errorMessage: string
}

export type ChartState = {
  chartData: Array<Record<string, number | null | undefined>>
  seriesByMonitorID: Record<string, SeriesPoint[]>
  resultsByTimestamp: Record<number, LatencyMonitorResult[]>
}

export function buildLatencyMonitorSeries(monitors: LatencyMonitor[], results: LatencyMonitorResult[]): ChartState {
  const knownMonitorIDs = new Set(monitors.map((monitor) => monitor.id))
  const rowsByTimestamp = new Map<number, Record<string, number | null | undefined>>()
  const seriesByMonitorID: Record<string, SeriesPoint[]> = {}
  const resultsByTimestamp: Record<number, LatencyMonitorResult[]> = {}
  const clusterThresholdSeconds = getLatencyClusterThresholdSeconds(monitors)

  for (const monitor of monitors) {
    seriesByMonitorID[monitor.id] = []
  }

  const relevantResults = results
    .filter((result) => knownMonitorIDs.has(result.monitor_id))
    .sort((left, right) => left.ts - right.ts)

  const clusters: Array<{ ts: number; results: LatencyMonitorResult[] }> = []
  for (const result of relevantResults) {
    const previousCluster = clusters.at(-1)
    if (!previousCluster || result.ts - previousCluster.ts > clusterThresholdSeconds) {
      clusters.push({ ts: result.ts, results: [result] })
      continue
    }
    previousCluster.results.push(result)
  }

  for (const cluster of clusters) {
    const row = rowsByTimestamp.get(cluster.ts) ?? { ts: cluster.ts }
    const latestByMonitorID = new Map<string, LatencyMonitorResult>()

    for (const result of cluster.results) {
      const current = latestByMonitorID.get(result.monitor_id)
      if (!current || result.ts >= current.ts) {
        latestByMonitorID.set(result.monitor_id, result)
      }
    }

    const tooltipResults = Array.from(latestByMonitorID.values()).sort((left, right) => {
      if (left.monitor_id === right.monitor_id) {
        return left.ts - right.ts
      }
      return left.monitor_id.localeCompare(right.monitor_id)
    })

    for (const result of tooltipResults) {
      row[result.monitor_id] = result.success ? result.latency_ms : null
      seriesByMonitorID[result.monitor_id].push({
        ts: cluster.ts,
        value: result.success ? result.latency_ms : null,
        success: result.success,
        errorMessage: result.error_message ?? "",
      })
    }

    rowsByTimestamp.set(cluster.ts, row)
    resultsByTimestamp[cluster.ts] = tooltipResults
  }

  for (const monitorID of Object.keys(seriesByMonitorID)) {
    seriesByMonitorID[monitorID].sort((left, right) => left.ts - right.ts)
  }

  return {
    chartData: Array.from(rowsByTimestamp.values()).sort((left, right) => Number(left.ts) - Number(right.ts)),
    seriesByMonitorID,
    resultsByTimestamp,
  }
}

function getLatencyClusterThresholdSeconds(monitors: LatencyMonitor[]): number {
  const intervals = monitors
    .map((monitor) => monitor.interval_seconds)
    .filter((value): value is number => typeof value === "number" && Number.isFinite(value) && value > 0)

  if (intervals.length === 0) {
    return 5
  }

  const minInterval = Math.min(...intervals)
  return Math.max(2, Math.min(15, Math.floor(minInterval / 4)))
}
