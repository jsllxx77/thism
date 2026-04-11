import type { MetricsRow } from "./api"
import { deriveRateSeries, type SeriesPoint } from "./units"

type DownsampleStrategy = "average" | "last"
export type ChartPoint = { ts: number; value: number | null }

export type NodeDetailMetricSeries = {
  cpuData: ChartPoint[]
  memData: ChartPoint[]
  netRxData: ChartPoint[]
  netTxData: ChartPoint[]
  netRxSpeedData: ChartPoint[]
  netTxSpeedData: ChartPoint[]
  diskData: ChartPoint[]
}

type MetricsValueSelector = (row: MetricsRow) => number

const MIN_GAP_BREAK_SECONDS = 30
const GAP_BREAK_MULTIPLIER = 6

export function getChartPointBudget(rangeSeconds: number): number {
  if (rangeSeconds <= 3600) {
    return 120
  }
  if (rangeSeconds <= 21600) {
    return 180
  }
  if (rangeSeconds <= 86400) {
    return 240
  }
  if (rangeSeconds <= 604800) {
    return 280
  }
  return 360
}

export function appendLiveMetricPoint(
  previous: ReadonlyArray<MetricsRow>,
  point: MetricsRow,
  rangeSeconds: number,
): MetricsRow[] {
  const next = previous.length > 0 && previous[previous.length - 1]?.ts === point.ts
    ? [...previous.slice(0, -1), point]
    : [...previous, point]

  if (rangeSeconds <= 0) {
    return next
  }

  const cutoff = point.ts - rangeSeconds
  const firstVisibleIndex = next.findIndex((sample) => sample.ts >= cutoff)

  if (firstVisibleIndex <= 0) {
    return next
  }

  return next.slice(firstVisibleIndex)
}

export function downsampleSeriesForRange(
  points: ReadonlyArray<SeriesPoint>,
  rangeSeconds: number,
  strategy: DownsampleStrategy,
): SeriesPoint[] {
  return downsampleSeries(points, getChartPointBudget(rangeSeconds), strategy)
}

export function buildMetricChartSeries(
  metrics: ReadonlyArray<MetricsRow>,
  rangeSeconds: number,
  selectValue: MetricsValueSelector,
  strategy: DownsampleStrategy,
): ChartPoint[] {
  const segments = splitMetricSegments(metrics).map((segment) =>
    segment.map((row) => ({ ts: row.ts, value: selectValue(row) })),
  )
  return mergeSegmentedSeries(segments, getChartPointBudget(rangeSeconds), strategy)
}

export function buildMetricRateChartSeries(
  metrics: ReadonlyArray<MetricsRow>,
  rangeSeconds: number,
  selectValue: MetricsValueSelector,
): ChartPoint[] {
  const segments = splitMetricSegments(metrics).map((segment) =>
    deriveRateSeries(segment.map((row) => ({ ts: row.ts, value: selectValue(row) }))),
  )
  return mergeSegmentedSeries(segments, getChartPointBudget(rangeSeconds), "average")
}

export function buildNodeDetailMetricSeries(
  metrics: ReadonlyArray<MetricsRow>,
  rangeSeconds: number,
): NodeDetailMetricSeries {
  const targetPoints = getChartPointBudget(rangeSeconds)
  const segments = splitMetricSegments(metrics).map((segment) => {
    const cpu: SeriesPoint[] = []
    const memory: SeriesPoint[] = []
    const netRx: SeriesPoint[] = []
    const netTx: SeriesPoint[] = []
    const netRxSpeed: SeriesPoint[] = []
    const netTxSpeed: SeriesPoint[] = []
    const disk: SeriesPoint[] = []
    let previous: MetricsRow | null = null

    segment.forEach((row) => {
      cpu.push({ ts: row.ts, value: row.cpu })
      memory.push({ ts: row.ts, value: row.mem_total > 0 ? (row.mem_used / row.mem_total) * 100 : 0 })
      netRx.push({ ts: row.ts, value: row.net_rx })
      netTx.push({ ts: row.ts, value: row.net_tx })
      disk.push({ ts: row.ts, value: row.disk_total > 0 ? (row.disk_used / row.disk_total) * 100 : 0 })

      if (previous) {
        const deltaTs = row.ts - previous.ts
        const rxDelta = row.net_rx - previous.net_rx
        const txDelta = row.net_tx - previous.net_tx

        if (deltaTs > 0 && rxDelta >= 0) {
          netRxSpeed.push({ ts: row.ts, value: rxDelta / deltaTs })
        }
        if (deltaTs > 0 && txDelta >= 0) {
          netTxSpeed.push({ ts: row.ts, value: txDelta / deltaTs })
        }
      }

      previous = row
    })

    return { cpu, memory, netRx, netTx, netRxSpeed, netTxSpeed, disk }
  })

  return {
    cpuData: mergeSegmentedSeries(segments.map((segment) => segment.cpu), targetPoints, "average"),
    memData: mergeSegmentedSeries(segments.map((segment) => segment.memory), targetPoints, "average"),
    netRxData: mergeSegmentedSeries(segments.map((segment) => segment.netRx), targetPoints, "last"),
    netTxData: mergeSegmentedSeries(segments.map((segment) => segment.netTx), targetPoints, "last"),
    netRxSpeedData: mergeSegmentedSeries(segments.map((segment) => segment.netRxSpeed), targetPoints, "average"),
    netTxSpeedData: mergeSegmentedSeries(segments.map((segment) => segment.netTxSpeed), targetPoints, "average"),
    diskData: mergeSegmentedSeries(segments.map((segment) => segment.disk), targetPoints, "average"),
  }
}

function splitMetricSegments(metrics: ReadonlyArray<MetricsRow>): MetricsRow[][] {
  if (metrics.length === 0) {
    return []
  }

  const gapThreshold = getGapBreakThreshold(metrics)
  const segments: MetricsRow[][] = []
  let currentSegment: MetricsRow[] = [metrics[0]]

  for (let index = 1; index < metrics.length; index += 1) {
    const previous = metrics[index - 1]
    const current = metrics[index]

    if (shouldBreakMetricSegment(previous, current, gapThreshold)) {
      segments.push(currentSegment)
      currentSegment = [current]
      continue
    }

    currentSegment.push(current)
  }

  if (currentSegment.length > 0) {
    segments.push(currentSegment)
  }

  return segments
}

function getGapBreakThreshold(metrics: ReadonlyArray<MetricsRow>): number {
  const deltas: number[] = []

  for (let index = 1; index < metrics.length; index += 1) {
    const delta = metrics[index].ts - metrics[index - 1].ts
    if (delta > 0) {
      deltas.push(delta)
    }
  }

  if (deltas.length === 0) {
    return MIN_GAP_BREAK_SECONDS
  }

  deltas.sort((left, right) => left - right)
  const median = deltas[Math.floor(deltas.length / 2)]
  return Math.max(MIN_GAP_BREAK_SECONDS, median * GAP_BREAK_MULTIPLIER)
}

function shouldBreakMetricSegment(previous: MetricsRow, current: MetricsRow, gapThreshold: number): boolean {
  const delta = current.ts - previous.ts
  if (delta <= 0) {
    return true
  }
  if (delta > gapThreshold) {
    return true
  }

  const previousUptime = previous.uptime_seconds
  const currentUptime = current.uptime_seconds
  if (
    typeof previousUptime === "number" &&
    typeof currentUptime === "number" &&
    previousUptime > 0 &&
    currentUptime > 0 &&
    currentUptime < previousUptime
  ) {
    return true
  }

  return false
}

function mergeSegmentedSeries(
  segments: ReadonlyArray<ReadonlyArray<SeriesPoint>>,
  targetPoints: number,
  strategy: DownsampleStrategy,
): ChartPoint[] {
  const nonEmptySegments = segments.filter((segment) => segment.length > 0)
  if (nonEmptySegments.length === 0) {
    return []
  }

  const budgets = allocateSegmentBudgets(nonEmptySegments.map((segment) => segment.length), targetPoints)
  const merged: ChartPoint[] = []

  nonEmptySegments.forEach((segment, index) => {
    const reduced = downsampleSeries(segment, budgets[index], strategy)
    if (reduced.length === 0) {
      return
    }

    if (merged.length > 0) {
      merged.push({ ts: Math.max(reduced[0].ts - 1, merged[merged.length - 1].ts + 1), value: null })
    }

    merged.push(...reduced)
  })

  return merged
}

function allocateSegmentBudgets(lengths: ReadonlyArray<number>, targetPoints: number): number[] {
  if (lengths.length === 0) {
    return []
  }

  const totalLength = lengths.reduce((sum, length) => sum + length, 0)
  const safeTarget = Math.max(targetPoints, lengths.length)

  const budgets = lengths.map((length) => Math.max(1, Math.floor((safeTarget * length) / totalLength)))
  let assigned = budgets.reduce((sum, budget) => sum + budget, 0)

  if (assigned < safeTarget) {
    const remainders = lengths
      .map((length, index) => ({
        index,
        remainder: (safeTarget * length) / totalLength - budgets[index],
      }))
      .sort((left, right) => right.remainder - left.remainder)

    let cursor = 0
    while (assigned < safeTarget && remainders.length > 0) {
      const target = remainders[cursor % remainders.length]
      budgets[target.index] += 1
      assigned += 1
      cursor += 1
    }
  }

  return budgets
}

function downsampleSeries(
  points: ReadonlyArray<SeriesPoint>,
  targetPoints: number,
  strategy: DownsampleStrategy,
): SeriesPoint[] {
  if (targetPoints <= 0 || points.length <= targetPoints) {
    return [...points]
  }

  const bucketSize = Math.max(1, Math.ceil(points.length / targetPoints))
  const reduced: SeriesPoint[] = []

  for (let index = 0; index < points.length; index += bucketSize) {
    const bucket = points.slice(index, index + bucketSize)
    const lastPoint = bucket[bucket.length - 1]

    if (!lastPoint) {
      continue
    }

    if (strategy === "last") {
      reduced.push({ ts: lastPoint.ts, value: lastPoint.value })
      continue
    }

    const total = bucket.reduce((sum, point) => sum + point.value, 0)
    reduced.push({
      ts: lastPoint.ts,
      value: total / bucket.length,
    })
  }

  return reduced
}
