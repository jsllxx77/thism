import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NodeDetail } from "./NodeDetail"

const nodeMock = vi.fn()
const metricsMock = vi.fn()
const processesMock = vi.fn()
const servicesMock = vi.fn()
const dockerMock = vi.fn()
const metricsRetentionMock = vi.fn()
const latencyResultsMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    node: (...args: unknown[]) => nodeMock(...args),
    metrics: (...args: unknown[]) => metricsMock(...args),
    processes: (...args: unknown[]) => processesMock(...args),
    services: (...args: unknown[]) => servicesMock(...args),
    docker: (...args: unknown[]) => dockerMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    latencyResults: (...args: unknown[]) => latencyResultsMock(...args),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: () => {},
    off: () => {},
  }),
}))

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe("node detail page states", () => {
  beforeEach(() => {
    nodeMock.mockReset()
    metricsMock.mockReset()
    processesMock.mockReset()
    servicesMock.mockReset()
    dockerMock.mockReset()
    metricsRetentionMock.mockReset()
    latencyResultsMock.mockReset()

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
    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
  })

  it("shows a loading state while node detail data is pending", async () => {
    const nodeRequest = deferred<{
      node: {
        id: string
        name: string
        ip: string
        os: string
        arch: string
        created_at: number
        last_seen: number
        online: boolean
      } | null
    }>()

    nodeMock.mockReturnValue(nodeRequest.promise)
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)
    expect(screen.getByText("Loading node details...")).toBeInTheDocument()

    nodeRequest.resolve({
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

    await waitFor(() => {
      expect(screen.queryByText("Loading node details...")).not.toBeInTheDocument()
    })
  })

  it("shows an error state when node detail data fails", async () => {
    nodeMock.mockRejectedValue(new Error("timeout"))
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)

    expect(await screen.findByRole("alert")).toHaveTextContent("We couldn't load node details. Please try again.")
  })

  it("shows docker details when docker is available", async () => {
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
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({
      docker_available: true,
      containers: [
        { id: "0123456789ab", name: "web", image: "nginx:alpine", state: "running", status: "Up 2 hours" },
      ],
    })

    render(<NodeDetail nodeId="node-1" />)

    expect(await screen.findByRole("button", { name: /Docker/i })).toBeInTheDocument()
  })

  it("uses a single detail column when only one operational section is available", async () => {
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
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([{ pid: 10, name: "sshd", cpu: 0.1, mem: 1024 }])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)

    const processSnapshot = await screen.findByRole("button", { name: /Top Processes/i })
    const sectionsGrid = processSnapshot.closest("div.grid")

    expect(sectionsGrid).toBeInTheDocument()
    expect(sectionsGrid).not.toHaveClass("xl:grid-cols-2")
  })

  it("refetches node detail data when refreshNonce changes", async () => {
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
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    const { rerender } = render(<NodeDetail nodeId="node-1" refreshNonce={0} />)
    await waitFor(() => {
      expect(nodeMock).toHaveBeenCalledTimes(1)
      expect(metricsMock).toHaveBeenCalledTimes(1)
      expect(latencyResultsMock).toHaveBeenCalledTimes(1)
      expect(processesMock).toHaveBeenCalledTimes(1)
      expect(servicesMock).toHaveBeenCalledTimes(1)
      expect(dockerMock).toHaveBeenCalledTimes(1)
    })

    rerender(<NodeDetail nodeId="node-1" refreshNonce={1} />)
    await waitFor(() => {
      expect(nodeMock).toHaveBeenCalledTimes(2)
      expect(metricsMock).toHaveBeenCalledTimes(2)
      expect(latencyResultsMock).toHaveBeenCalledTimes(2)
      expect(processesMock).toHaveBeenCalledTimes(2)
      expect(servicesMock).toHaveBeenCalledTimes(2)
      expect(dockerMock).toHaveBeenCalledTimes(2)
    })
  })

  it("uses a shared range control for metrics and latency history", async () => {
    const user = userEvent.setup()
    const dateNowSpy = vi.spyOn(Date, "now").mockReturnValue(1700003600 * 1000)

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
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({
      monitors: [
        {
          id: "monitor-1",
          name: "Guangdong Telecom IPv4",
          type: "tcp",
          target: "gd-ct-v4.ip.zstaticcdn.com:80",
          interval_seconds: 60,
          auto_assign_new_nodes: true,
          assigned_node_count: 1,
          assigned_node_ids: ["node-1"],
          created_at: 1,
          updated_at: 1,
        },
      ],
      results: [
        { monitor_id: "monitor-1", node_id: "node-1", ts: 1700003000, latency_ms: 23.5, loss_percent: 20, jitter_ms: 4.2, success: true },
      ],
    })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)

    expect(await screen.findByRole("button", { name: "1h" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Guangdong Telecom IPv4/i })).toHaveTextContent("Loss: 20%")
    expect(metricsMock).toHaveBeenCalledWith("node-1", 1700000000, 1700003600)
    expect(latencyResultsMock).toHaveBeenCalledWith("node-1", 1700000000, 1700003600)

    await user.click(screen.getByRole("button", { name: "6h" }))

    await waitFor(() => {
      expect(metricsMock).toHaveBeenLastCalledWith("node-1", 1699982000, 1700003600)
      expect(latencyResultsMock).toHaveBeenLastCalledWith("node-1", 1699982000, 1700003600)
    })

    dateNowSpy.mockRestore()
  })

  it("renders hardware passport when node metadata includes hardware", async () => {
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
          hardware: {
            cpu_model: "AMD EPYC 7B13",
            cpu_cores: 8,
            cpu_threads: 16,
            memory_total: 34359738368,
            disk_total: 322122547200,
            virtualization_system: "kvm",
            virtualization_role: "guest",
          },
      },
    })
    metricsMock.mockResolvedValue({ metrics: [] })
    latencyResultsMock.mockResolvedValue({ monitors: [], results: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)

    expect(await screen.findByText("Hardware")).toBeInTheDocument()
    expect(screen.getAllByText("AMD EPYC 7B13").length).toBeGreaterThan(0)
  })


  it("shows a 30d metric range when retention is set to 30 days", async () => {
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [7, 30] })
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
    metricsMock.mockResolvedValue({ metrics: [] })
    processesMock.mockResolvedValue([])
    servicesMock.mockResolvedValue({ services: [] })
    dockerMock.mockResolvedValue({ docker_available: false, containers: [] })

    render(<NodeDetail nodeId="node-1" />)

    expect(await screen.findByRole("button", { name: "30d" })).toBeInTheDocument()
  })

  it("limits guest users to basic node information", async () => {
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
          hardware: {
            cpu_model: "AMD EPYC 7B13",
            cpu_cores: 8,
            cpu_threads: 16,
            memory_total: 34359738368,
            disk_total: 322122547200,
            virtualization_system: "kvm",
            virtualization_role: "guest",
          },
        },
    })
    metricsMock.mockResolvedValue({ metrics: [] })
    processesMock.mockResolvedValue([{ pid: 10, name: "sshd", cpu: 0.1, mem: 1024 }])
    servicesMock.mockResolvedValue({ services: [{ name: "nginx", status: "ok", last_checked: 0 }] })
    dockerMock.mockResolvedValue({
      docker_available: true,
      containers: [{ id: "0123456789ab", name: "web", image: "nginx:alpine", state: "running", status: "Up 2 hours" }],
    })

    render(<NodeDetail nodeId="node-1" accessMode="guest" />)

    expect(await screen.findByText("alpha")).toBeInTheDocument()
    expect(screen.queryByText("1.1.1.1")).not.toBeInTheDocument()
    expect(screen.getByText("Hardware")).toBeInTheDocument()
    expect(screen.queryByText("CPU Usage")).not.toBeInTheDocument()
    expect(screen.queryByText("Top Processes")).not.toBeInTheDocument()
    expect(nodeMock).toHaveBeenCalledWith("node-1")
    expect(metricsMock).not.toHaveBeenCalled()
    expect(processesMock).not.toHaveBeenCalled()
    expect(servicesMock).not.toHaveBeenCalled()
    expect(dockerMock).not.toHaveBeenCalled()
  })
})
