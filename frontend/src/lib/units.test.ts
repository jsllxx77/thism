import { describe, expect, it } from "vitest"
import { deriveRateSeries, formatBytes, formatBytesPerSecond, formatCompactBytes, formatCompactBytesPerSecond } from "./units"

describe("formatBytes", () => {
  it("formats values with 1024-based units", () => {
    expect(formatBytes(0)).toBe("0 B")
    expect(formatBytes(1023)).toBe("1023.0 B")
    expect(formatBytes(1024)).toBe("1.0 KB")
    expect(formatBytes(1536)).toBe("1.5 KB")
    expect(formatBytes(1024 ** 2)).toBe("1.0 MB")
    expect(formatBytes(1024 ** 3)).toBe("1.0 GB")
    expect(formatBytes(1024 ** 4)).toBe("1.0 TB")
  })
})

describe("formatBytesPerSecond", () => {
  it("formats throughput values with a /s suffix", () => {
    expect(formatBytesPerSecond(0)).toBe("0 B/s")
    expect(formatBytesPerSecond(1536)).toBe("1.5 KB/s")
  })
})

describe("formatCompactBytes", () => {
  it("formats values with short axis-safe units", () => {
    expect(formatCompactBytes(0)).toBe("0B")
    expect(formatCompactBytes(512)).toBe("512B")
    expect(formatCompactBytes(1536)).toBe("1.5K")
    expect(formatCompactBytes(12 * 1024)).toBe("12K")
    expect(formatCompactBytes(1024 ** 2)).toBe("1M")
  })
})

describe("formatCompactBytesPerSecond", () => {
  it("keeps rate labels compact for chart axes", () => {
    expect(formatCompactBytesPerSecond(1536)).toBe("1.5K/s")
    expect(formatCompactBytesPerSecond(12 * 1024 ** 2)).toBe("12M/s")
  })
})

describe("deriveRateSeries", () => {
  it("derives per-second rates from cumulative counters", () => {
    expect(
      deriveRateSeries([
        { ts: 100, value: 1024 },
        { ts: 105, value: 2048 },
        { ts: 109, value: 4096 },
      ])
    ).toEqual([
      { ts: 105, value: 204.8 },
      { ts: 109, value: 512 },
    ])
  })

  it("skips samples with non-positive time deltas or counter resets", () => {
    expect(
      deriveRateSeries([
        { ts: 100, value: 100 },
        { ts: 100, value: 150 },
        { ts: 105, value: 120 },
        { ts: 110, value: 220 },
      ])
    ).toEqual([{ ts: 110, value: 20 }])
  })
})
