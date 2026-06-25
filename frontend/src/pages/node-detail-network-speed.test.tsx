import { beforeEach, describe, expect, it, vi } from "vitest"
import { act, render, screen, waitFor, within } from "@testing-library/react"
import { NodeDetail } from "./NodeDetail"

const nodeMock = vi.fn()
const metricsMock = vi.fn()
const processesMock = vi.fn()
const servicesMock = vi.fn()
const dockerMock = vi.fn()
const metricsRetentionMock = vi.fn()
let wsHandler: ((msg: { type: string; payload?: unknown }) => void) | null = null

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

describe("node detail network speed", () => {
  beforeEach(() => {
    nodeMock.mockReset()
    metricsMock.mockReset()
    processesMock.mockReset()
    servicesMock.mockReset()
    dockerMock.mockReset()
    metricsRetentionMock.mockReset()
    wsHandler = null
    mockMatchMedia()
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })

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
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })
  })

  it("shows a network summary plus traffic and speed charts", async () => {
    metricsMock.mockResolvedValue({
      metrics: [
        {
          ts: 100,
          cpu: 10,
          mem_used: 100,
          mem_total: 200,
          disk_used: 300,
          disk_total: 600,
          net_rx: 1024,
          net_tx: 2048,
        },
        {
          ts: 110,
          cpu: 15,
          mem_used: 110,
          mem_total: 200,
          disk_used: 310,
          disk_total: 600,
          net_rx: 1024 + 5 * 1024 * 1024,
          net_tx: 2048 + 2 * 1024 * 1024,
        },
      ],
    })

    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    expect(screen.getByRole("heading", { name: "Throughput / Traffic" })).toBeInTheDocument()
    expect(screen.getByRole("region", { name: "Network Traffic" })).toBeInTheDocument()
    const summary = screen.getByRole("region", { name: "Network summary" })
    expect(within(summary).getByText("Inbound Total")).toBeInTheDocument()
    expect(within(summary).getByText("Outbound Total")).toBeInTheDocument()
    expect(within(summary).getByText("Inbound Speed")).toBeInTheDocument()
    expect(within(summary).getByText("Outbound Speed")).toBeInTheDocument()
    expect(within(summary).getByText("5.0 MB")).toBeInTheDocument()
    expect(within(summary).getByText("2.0 MB")).toBeInTheDocument()
    expect(within(summary).getByText("512.0 KB/s")).toBeInTheDocument()
    expect(within(summary).getByText("204.8 KB/s")).toBeInTheDocument()

    expect(screen.getByText("Inbound Traffic")).toBeInTheDocument()
    expect(screen.getByText("Outbound Traffic")).toBeInTheDocument()
    expect(screen.getAllByText("Inbound Speed").length).toBeGreaterThanOrEqual(2)
    expect(screen.getAllByText("Outbound Speed").length).toBeGreaterThanOrEqual(2)
  })

  it("shows totals but keeps speed as placeholder until a second sample exists", async () => {
    metricsMock.mockResolvedValue({
      metrics: [
        {
          ts: 100,
          cpu: 10,
          mem_used: 100,
          mem_total: 200,
          disk_used: 300,
          disk_total: 600,
          net_rx: 1024,
          net_tx: 2048,
        },
      ],
    })

    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    const summary = screen.getByRole("region", { name: "Network summary" })
    expect(within(summary).getByText("Inbound Total")).toBeInTheDocument()
    expect(within(summary).getByText("Outbound Total")).toBeInTheDocument()
    expect(within(summary).getByText("1.0 KB")).toBeInTheDocument()
    expect(within(summary).getByText("2.0 KB")).toBeInTheDocument()
    expect(within(summary).getAllByText("—")).toHaveLength(2)
  })

  it("updates the network summary when a live websocket metric arrives", async () => {
    metricsMock.mockResolvedValue({
      metrics: [
        {
          ts: 100,
          cpu: 10,
          mem_used: 100,
          mem_total: 200,
          disk_used: 300,
          disk_total: 600,
          net_rx: 1024,
          net_tx: 2048,
        },
        {
          ts: 110,
          cpu: 15,
          mem_used: 110,
          mem_total: 200,
          disk_used: 310,
          disk_total: 600,
          net_rx: 1024 + 5 * 1024 * 1024,
          net_tx: 2048 + 2 * 1024 * 1024,
        },
      ],
    })

    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    act(() => {
      wsHandler?.({
        type: "metrics",
        payload: {
          node_id: "node-1",
          data: {
            ts: 120,
            cpu: 12,
            mem: { used: 120, total: 200 },
            net: { rx_bytes: 1024 + 7 * 1024 * 1024, tx_bytes: 2048 + 3 * 1024 * 1024 },
          },
        },
      })
    })

    const summary = screen.getByRole("region", { name: "Network summary" })
    expect(within(summary).getByText("7.0 MB")).toBeInTheDocument()
    expect(within(summary).getByText("3.0 MB")).toBeInTheDocument()
    expect(within(summary).getByText("204.8 KB/s")).toBeInTheDocument()
    expect(within(summary).getByText("102.4 KB/s")).toBeInTheDocument()
  })

  it("does not reuse stale speeds after a reboot until the new segment has two samples", async () => {
    metricsMock.mockResolvedValue({
      metrics: [
        {
          ts: 100,
          cpu: 10,
          mem_used: 100,
          mem_total: 200,
          disk_used: 300,
          disk_total: 600,
          net_rx: 1024,
          net_tx: 2048,
          uptime_seconds: 1000,
        },
        {
          ts: 110,
          cpu: 15,
          mem_used: 110,
          mem_total: 200,
          disk_used: 310,
          disk_total: 600,
          net_rx: 1024 + 5 * 1024 * 1024,
          net_tx: 2048 + 2 * 1024 * 1024,
          uptime_seconds: 1010,
        },
        {
          ts: 120,
          cpu: 12,
          mem_used: 90,
          mem_total: 200,
          disk_used: 250,
          disk_total: 600,
          net_rx: 512,
          net_tx: 1024,
          uptime_seconds: 5,
        },
      ],
    })

    render(<NodeDetail nodeId="node-1" />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    const summary = screen.getByRole("region", { name: "Network summary" })
    expect(within(summary).getByText("512.0 B")).toBeInTheDocument()
    expect(within(summary).getByText("1.0 KB")).toBeInTheDocument()
    expect(within(summary).getAllByText("—")).toHaveLength(2)
    expect(within(summary).queryByText("512.0 KB/s")).not.toBeInTheDocument()
    expect(within(summary).queryByText("204.8 KB/s")).not.toBeInTheDocument()
  })
})
