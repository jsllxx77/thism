import { render } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

const metricsChartSpy = vi.fn()

vi.mock("../MetricsChart", () => ({
  MetricsChart: (props: {
    label: string
    xAxisTickFormatter?: (value: number) => string
    tooltipLabelFormatter?: (value: number) => string
  }) => {
    metricsChartSpy(props)
    return <div data-testid={`chart-${props.label}`} />
  },
}))

import { MetricTabs } from "./MetricTabs"

const points = [
  { ts: 1700000000, value: 30 },
  { ts: 1700000300, value: 44 },
]

describe("metric tabs date labels", () => {
  it("adds date information to chart time labels for 7d range", () => {
    render(
      <MetricTabs
        range={604800}
        cpuData={points}
        memData={points}
        netRxData={points}
        netTxData={points}
        netRxSpeedData={points}
        netTxSpeedData={points}
        diskData={points}
      />,
    )

    const cpuChartProps = metricsChartSpy.mock.calls.at(-1)?.[0]
    expect(cpuChartProps?.xAxisTickFormatter?.(1700000000)).toMatch(/\d{2}[-/]\d{2}/)
    expect(cpuChartProps?.tooltipLabelFormatter?.(1700000000)).toMatch(/\d{2}[-/]\d{2}/)
  })
})
