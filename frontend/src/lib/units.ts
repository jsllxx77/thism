const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB"] as const

export type SeriesPoint = {
  ts: number
  value: number
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) {
    return "0 B"
  }

  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), BYTE_UNITS.length - 1)
  const value = bytes / 1024 ** unitIndex

  return `${value.toFixed(1)} ${BYTE_UNITS[unitIndex]}`
}

export function formatBytesPerSecond(bytesPerSecond: number): string {
  return `${formatBytes(bytesPerSecond)}/s`
}

export function deriveRateSeries(points: ReadonlyArray<SeriesPoint>): SeriesPoint[] {
  const rates: SeriesPoint[] = []

  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1]
    const current = points[index]
    const deltaTs = current.ts - previous.ts
    const deltaValue = current.value - previous.value

    if (deltaTs <= 0 || deltaValue < 0) {
      continue
    }

    rates.push({
      ts: current.ts,
      value: deltaValue / deltaTs,
    })
  }

  return rates
}
