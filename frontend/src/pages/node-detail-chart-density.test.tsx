import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

const nodeMock = vi.fn()
const metricsMock = vi.fn()
const processesMock = vi.fn()
const servicesMock = vi.fn()
const dockerMock = vi.fn()
const metricsRetentionMock = vi.fn()
let wsHandler: ((msg: { type: string; payload?: unknown }) => void) | null = null
let latestCPUData: Array<{ ts: number; value: number }> = []

vi.mock("../lib/api", () => ({
  api: {
    node: (...args: unknown[]) => nodeMock(...args),
    metrics: (...args: unknown[]) => metricsMock(...args),
    processes: (...args: unknown[]) => processesMock(...args),
    services: (...args: unknown[]) => servicesMock(...args),
    docker: (...args: unknown[]) => dockerMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: (handler: (msg: { type: string; payload?: unknown }) => void) => {
      wsHandler = handler
    },
    off: () => {
      wsHandler = null
    },
  }),
}))

vi.mock("../components/node-detail/MetricTabs", () => ({
  MetricTabs: (props: {
    range: number
    cpuData: Array<{ ts: number; value: number }>
  }) => {
    latestCPUData = props.cpuData
    const span = props.cpuData.length > 1 ? props.cpuData[props.cpuData.length - 1].ts - props.cpuData[0].ts : 0

    return (
      <div>
        <div data-testid="metric-range">{props.range}</div>
        <div data-testid="cpu-points">{props.cpuData.length}</div>
        <div data-testid="cpu-span">{span}</div>
      </div>
    )
  },
}))

import { NodeDetail } from "./NodeDetail"

function mockMatchMedia() {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: () => ({
      matches: true,
      media: "(min-width: 768px)",
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

function buildMetrics(from: number, to: number) {
  const rows = []
  for (let ts = from + 5; ts <= to; ts += 5) {
    rows.push({
      ts,
      cpu: ts % 100,
      mem_used: 512,
      mem_total: 1024,
      disk_used: 2048,
      disk_total: 4096,
      net_rx: ts * 10,
      net_tx: ts * 20,
      uptime_seconds: 1000 + ts,
    })
  }
  return rows
}

describe("node detail chart density", () => {
  beforeEach(() => {
    nodeMock.mockReset()
    metricsMock.mockReset()
    processesMock.mockReset()
    servicesMock.mockReset()
    dockerMock.mockReset()
    metricsRetentionMock.mockReset()
    wsHandler = null
    latestCPUData = []
    mockMatchMedia()

    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    nodeMock.mockResolvedValue({
      node: {
          id: "node-1",
          name: "alpha",
          ip: "1.1.1.1",
          os: "linux",
          arch: "amd64",
          created_at: 0,
          last_seen: 0,
          online: true,
        },
    })
    metricsMock.mockImplementation((_nodeId: string, from?: number, to?: number) =>
      Promise.resolve({ metrics: buildMetrics(from ?? 0, to ?? 0) })
    )
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })
  })

  it("keeps the selected 6h window after live updates while reducing chart density", async () => {
    const user = userEvent.setup()

    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    await waitFor(() => {
      expect(Number(screen.getByTestId("cpu-points").textContent)).toBeLessThanOrEqual(120)
    })
    expect(Number(screen.getByTestId("cpu-span").textContent)).toBeGreaterThan(3000)

    await user.click(screen.getByRole("button", { name: "6h" }))

    await waitFor(() => {
      expect(screen.getByTestId("metric-range")).toHaveTextContent("21600")
    })
    await waitFor(() => {
      expect(Number(screen.getByTestId("cpu-points").textContent)).toBeLessThanOrEqual(180)
    })
    expect(Number(screen.getByTestId("cpu-span").textContent)).toBeGreaterThan(18000)

    const latestPoint = latestCPUData[latestCPUData.length - 1]
    expect(latestPoint).toBeDefined()

    act(() => {
      wsHandler?.({
        type: "metrics",
        payload: {
          node_id: "node-1",
          data: {
            ts: latestPoint.ts + 5,
            cpu: 42,
            mem: { used: 768, total: 1024 },
            disk_used: 2048,
            disk_total: 4096,
            net: { rx_bytes: (latestPoint.ts + 5) * 10, tx_bytes: (latestPoint.ts + 5) * 20 },
            uptime_seconds: 1000 + latestPoint.ts + 5,
          },
        },
      })
    })

    await waitFor(() => {
      expect(Number(screen.getByTestId("cpu-points").textContent)).toBeLessThanOrEqual(180)
    })
    expect(Number(screen.getByTestId("cpu-span").textContent)).toBeGreaterThan(18000)
  })
})
