import { act, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

const nodeMock = vi.fn()
const metricsMock = vi.fn()
const processesMock = vi.fn()
const servicesMock = vi.fn()
const dockerMock = vi.fn()
const metricsRetentionMock = vi.fn()
let wsHandler: ((msg: { type: string; payload?: unknown }) => void) | null = null
let latestDiskValue: number | null | undefined

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
  MetricTabs: (props: { diskData: Array<{ ts: number; value: number | null }> }) => {
    latestDiskValue = props.diskData[props.diskData.length - 1]?.value
    return <div data-testid="latest-disk">{latestDiskValue == null ? "null" : String(latestDiskValue)}</div>
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

describe("node detail live disk usage", () => {
  beforeEach(() => {
    nodeMock.mockReset()
    metricsMock.mockReset()
    processesMock.mockReset()
    servicesMock.mockReset()
    dockerMock.mockReset()
    metricsRetentionMock.mockReset()
    wsHandler = null
    latestDiskValue = undefined
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
    metricsMock.mockResolvedValue({
      metrics: [
        {
          ts: 100,
          cpu: 10,
          mem_used: 100,
          mem_total: 200,
          disk_used: 300,
          disk_total: 600,
          net_rx: 1000,
          net_tx: 2000,
        },
      ],
    })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })
  })

  it("keeps disk usage from raw live disk partitions instead of dropping to zero", async () => {
    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })
    expect(screen.getByTestId("latest-disk")).toHaveTextContent("50")

    act(() => {
      wsHandler?.({
        type: "metrics",
        payload: {
          node_id: "node-1",
          data: {
            ts: 110,
            cpu: 12,
            mem: { used: 110, total: 200 },
            disk: [
              { mount: "/", used: 320, total: 640 },
              { mount: "/data", used: 40, total: 80 },
            ],
            net: { rx_bytes: 1200, tx_bytes: 2200 },
          },
        },
      })
    })

    await waitFor(() => {
      expect(screen.getByTestId("latest-disk")).toHaveTextContent("50")
    })
  })
})
