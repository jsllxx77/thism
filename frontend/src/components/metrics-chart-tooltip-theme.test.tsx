import { beforeEach, describe, expect, it, vi } from "vitest"
import { render } from "@testing-library/react"
import { MetricsChart } from "./MetricsChart"

type RechartsProps = {
  children?: React.ReactNode
}

type TooltipProps = {
  contentStyle?: {
    background?: string
    border?: string
    borderRadius?: number
    color?: string
    fontSize?: number
  }
  itemStyle?: {
    color?: string
  }
  labelStyle?: {
    color?: string
  }
  formatter?: (value: number) => [string, string]
}

type YAxisProps = {
  tickFormatter?: (value: number) => string
}

const tooltipSpy = vi.fn<(props: TooltipProps) => void>()
const yAxisSpy = vi.fn<(props: YAxisProps) => void>()

vi.mock("recharts", () => {
  return {
    ResponsiveContainer: ({ children }: RechartsProps) => <div data-testid="responsive-container">{children}</div>,
    AreaChart: ({ children }: RechartsProps) => <svg data-testid="area-chart">{children}</svg>,
    Area: () => <div data-testid="area-series" />,
    XAxis: () => <div data-testid="x-axis" />,
    YAxis: (props: YAxisProps) => {
      yAxisSpy(props)
      return <div data-testid="y-axis" />
    },
    Tooltip: (props: TooltipProps) => {
      tooltipSpy(props)
      return <div data-testid="tooltip-proxy" />
    },
  }
})

describe("MetricsChart tooltip theme", () => {
  beforeEach(() => {
    tooltipSpy.mockClear()
    yAxisSpy.mockClear()
  })

  it("uses theme-aware tooltip styles instead of hardcoded light background", () => {
    render(
      <MetricsChart
        label="CPU Usage"
        color="#2563eb"
        data={[
          { ts: 1700000000, value: 10 },
          { ts: 1700000300, value: 25 },
        ]}
      />,
    )

    expect(tooltipSpy).toHaveBeenCalled()
    const tooltipProps = tooltipSpy.mock.calls.at(-1)?.[0]

    expect(tooltipProps?.contentStyle?.background).toBe("hsl(var(--popover))")
    expect(tooltipProps?.contentStyle?.border).toBe("1px solid hsl(var(--border))")
    expect(tooltipProps?.contentStyle?.color).toBe("hsl(var(--popover-foreground))")
    expect(tooltipProps?.labelStyle?.color).toBe("hsl(var(--popover-foreground))")
    expect(tooltipProps?.itemStyle?.color).toBe("hsl(var(--popover-foreground))")
  })

  it("uses a custom value formatter for tooltip values without changing the default y-axis path", () => {
    const valueFormatter = vi.fn((value: number) => `${(value / 1024).toFixed(1)} MB/s`)

    render(
      <MetricsChart
        label="Inbound Traffic"
        color="#2563eb"
        unit=" KB"
        domain={[0, "auto"]}
        valueFormatter={valueFormatter}
        data={[
          { ts: 1700000000, value: 1024 },
          { ts: 1700000300, value: 2048 },
        ]}
      />,
    )

    const tooltipProps = tooltipSpy.mock.calls.at(-1)?.[0]
    const yAxisProps = yAxisSpy.mock.calls.at(-1)?.[0]

    expect(tooltipProps?.formatter?.(2048)).toEqual(["2.0 MB/s", "Inbound Traffic"])
    expect(yAxisProps?.tickFormatter).toBeUndefined()
    expect(valueFormatter).toHaveBeenCalledWith(2048)
  })

  it("lets y-axis ticks use a separate formatter when provided", () => {
    const valueFormatter = vi.fn((value: number) => `${(value / 1024).toFixed(1)} MB/s`)
    const axisTickFormatter = vi.fn((value: number) => `${Math.round(value / 1024)} MB/s`)

    render(
      <MetricsChart
        label="Outbound Traffic"
        color="#16a34a"
        unit=" KB"
        domain={[0, "auto"]}
        valueFormatter={valueFormatter}
        axisTickFormatter={axisTickFormatter}
        data={[
          { ts: 1700000000, value: 1024 },
          { ts: 1700000300, value: 2048 },
        ]}
      />,
    )

    const tooltipProps = tooltipSpy.mock.calls.at(-1)?.[0]
    const yAxisProps = yAxisSpy.mock.calls.at(-1)?.[0]

    expect(tooltipProps?.formatter?.(2048)).toEqual(["2.0 MB/s", "Outbound Traffic"])
    expect(yAxisProps?.tickFormatter?.(3072)).toBe("3 MB/s")
    expect(valueFormatter).toHaveBeenCalledWith(2048)
    expect(axisTickFormatter).toHaveBeenCalledWith(3072)
  })
})
