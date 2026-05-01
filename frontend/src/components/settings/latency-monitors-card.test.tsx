import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { LatencyMonitorsCard } from "./LatencyMonitorsCard"

const latencyMonitorsMock = vi.fn()
const createLatencyMonitorMock = vi.fn()
const updateLatencyMonitorMock = vi.fn()
const deleteLatencyMonitorMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    latencyMonitors: (...args: unknown[]) => latencyMonitorsMock(...args),
    createLatencyMonitor: (...args: unknown[]) => createLatencyMonitorMock(...args),
    updateLatencyMonitor: (...args: unknown[]) => updateLatencyMonitorMock(...args),
    deleteLatencyMonitor: (...args: unknown[]) => deleteLatencyMonitorMock(...args),
  },
}))

describe("latency monitors card", () => {
  beforeEach(() => {
    latencyMonitorsMock.mockReset()
    createLatencyMonitorMock.mockReset()
    updateLatencyMonitorMock.mockReset()
    deleteLatencyMonitorMock.mockReset()

    latencyMonitorsMock.mockResolvedValue({
      monitors: [
        {
          id: "monitor-1",
          name: "Guangdong Telecom IPv4",
          type: "tcp",
          target: "gd-ct-v4.ip.zstaticcdn.com:80",
          interval_seconds: 60,
          auto_assign_new_nodes: true,
          assigned_node_count: 2,
          assigned_node_ids: ["node-1", "node-2"],
          created_at: 1,
          updated_at: 1,
        },
      ],
    })
    createLatencyMonitorMock.mockResolvedValue({
      id: "monitor-2",
      name: "Beijing HTTP",
      type: "http",
      target: "https://example.com/healthz",
      interval_seconds: 90,
      auto_assign_new_nodes: true,
      assigned_node_count: 1,
      assigned_node_ids: ["node-1"],
      created_at: 2,
      updated_at: 2,
    })
  })

  it("shows a country flag in latency monitor node assignment when available", async () => {
    const user = userEvent.setup()
    render(
      <LatencyMonitorsCard
        nodes={[
          { id: "node-1", name: "Alpha", country_code: "HK", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
        ]}
      />
    )

    expect(await screen.findByText("Guangdong Telecom IPv4")).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "New monitor" }))
    const dialog = await screen.findByRole("dialog")
    expect(within(dialog).getByText("🇭🇰")).toBeInTheDocument()
    expect(within(dialog).getByText("Alpha")).toBeInTheDocument()
  })

  it("renders existing monitors and creates a new monitor with all nodes selected by default", async () => {
    const user = userEvent.setup()
    render(
      <LatencyMonitorsCard
        nodes={[
          { id: "node-1", name: "Alpha", country_code: "HK", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
          { id: "node-2", name: "Beta", ip: "2.2.2.2", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
        ]}
      />
    )

    expect(await screen.findByText("Guangdong Telecom IPv4")).toBeInTheDocument()
    expect(screen.getByText("gd-ct-v4.ip.zstaticcdn.com:80")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "New monitor" }))

    const dialog = await screen.findByRole("dialog")
    expect(within(dialog).getByLabelText("Assign Alpha")).toBeChecked()
    expect(within(dialog).getByLabelText("Assign Beta")).toBeChecked()

    await user.clear(within(dialog).getByLabelText("Monitor name"))
    await user.type(within(dialog).getByLabelText("Monitor name"), "Beijing HTTP")
    await user.selectOptions(within(dialog).getByLabelText("Monitor type"), "http")
    await user.clear(within(dialog).getByLabelText("Target"))
    await user.type(within(dialog).getByLabelText("Target"), "https://example.com/healthz")
    await user.clear(within(dialog).getByLabelText("Interval (seconds)"))
    await user.type(within(dialog).getByLabelText("Interval (seconds)"), "90")
    await user.click(within(dialog).getByLabelText("Assign Beta"))
    await user.click(within(dialog).getByRole("button", { name: "Save monitor" }))

    await waitFor(() => expect(createLatencyMonitorMock).toHaveBeenCalled())
    expect(createLatencyMonitorMock).toHaveBeenCalledWith({
      name: "Beijing HTTP",
      type: "http",
      target: "https://example.com/healthz",
      interval_seconds: 90,
      auto_assign_new_nodes: true,
      node_ids: ["node-1"],
    })
    expect(await screen.findByText("Latency monitor saved.")).toBeInTheDocument()
  })

  it("shows an error state when monitor loading fails", async () => {
    latencyMonitorsMock.mockRejectedValueOnce(new Error("boom"))

    render(
      <LatencyMonitorsCard
        nodes={[
          { id: "node-1", name: "Alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
        ]}
      />
    )

    expect(await screen.findByRole("alert")).toHaveTextContent("Failed to load latency monitors.")
  })
})
