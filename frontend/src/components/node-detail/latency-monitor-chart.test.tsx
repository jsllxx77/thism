import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { buildLatencyMonitorSeries, LatencyMonitorChart } from "./LatencyMonitorChart"

describe("latency monitor chart", () => {
  it("shows an empty state when the node has no assigned latency monitors", () => {
    render(<LatencyMonitorChart monitors={[]} results={[]} range={3600} />)

    expect(screen.getByText("No latency monitors are assigned to this node yet.")).toBeInTheDocument()
  })

  it("toggles monitor visibility from the legend buttons", async () => {
    const user = userEvent.setup()
    render(
      <LatencyMonitorChart
        range={3600}
        monitors={[
          { id: "monitor-1", name: "Guangdong Telecom IPv4", type: "tcp", target: "gd-ct-v4.ip.zstaticcdn.com:80", interval_seconds: 60, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
          { id: "monitor-2", name: "Beijing HTTP", type: "http", target: "https://example.com/healthz", interval_seconds: 90, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
        ]}
        results={[
          { monitor_id: "monitor-1", node_id: "node-1", ts: 100, latency_ms: 23.5, success: true },
          { monitor_id: "monitor-2", node_id: "node-1", ts: 100, latency_ms: 48.1, success: true },
        ]}
      />
    )

    const firstToggle = screen.getByRole("button", { name: /Guangdong Telecom IPv4/i })
    const secondToggle = screen.getByRole("button", { name: /Beijing HTTP/i })

    expect(firstToggle).toHaveAttribute("aria-pressed", "true")
    expect(secondToggle).toHaveAttribute("aria-pressed", "true")

    await user.click(firstToggle)

    expect(firstToggle).toHaveAttribute("aria-pressed", "false")
    expect(secondToggle).toHaveAttribute("aria-pressed", "true")
  })

  it("renders loss and jitter summary under the monitor name", () => {
    render(
      <LatencyMonitorChart
        range={3600}
        monitors={[
          { id: "monitor-1", name: "Guangdong Telecom IPv4", type: "tcp", target: "gd-ct-v4.ip.zstaticcdn.com:80", interval_seconds: 60, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
        ]}
        results={[
          { monitor_id: "monitor-1", node_id: "node-1", ts: 100, latency_ms: 23.5, loss_percent: 20, jitter_ms: 4.2, success: true },
        ]}
      />
    )

    const toggle = screen.getByRole("button", { name: /Guangdong Telecom IPv4/i })
    expect(toggle).toHaveTextContent("Loss: 20%")
    expect(toggle).toHaveTextContent("Jitter: 4.2 ms")
  })

  it("keeps each monitor as an independent series when timestamps are staggered", () => {
    const { chartData, seriesByMonitorID } = buildLatencyMonitorSeries(
      [
        { id: "monitor-1", name: "Guangdong Telecom IPv4", type: "tcp", target: "gd-ct-v4.ip.zstaticcdn.com:80", interval_seconds: 60, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
        { id: "monitor-2", name: "Beijing HTTP", type: "http", target: "https://example.com/healthz", interval_seconds: 90, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
      ],
      [
        { monitor_id: "monitor-1", node_id: "node-1", ts: 100, latency_ms: 23.5, success: true },
        { monitor_id: "monitor-2", node_id: "node-1", ts: 130, latency_ms: 48.1, success: true },
        { monitor_id: "monitor-1", node_id: "node-1", ts: 160, latency_ms: 24.2, success: true },
      ]
    )

    expect(seriesByMonitorID["monitor-1"]).toEqual([
      { ts: 100, value: 23.5, success: true, errorMessage: "" },
      { ts: 160, value: 24.2, success: true, errorMessage: "" },
    ])
    expect(seriesByMonitorID["monitor-2"]).toEqual([
      { ts: 130, value: 48.1, success: true, errorMessage: "" },
    ])
    expect(chartData).toEqual([
      { ts: 100, "monitor-1": 23.5 },
      { ts: 130, "monitor-2": 48.1 },
      { ts: 160, "monitor-1": 24.2 },
    ])
  })

  it("clusters nearby timestamps so tooltip rows can show multiple monitors together", () => {
    const { chartData, resultsByTimestamp } = buildLatencyMonitorSeries(
      [
        { id: "monitor-1", name: "Guangdong Telecom IPv4", type: "tcp", target: "gd-ct-v4.ip.zstaticcdn.com:80", interval_seconds: 60, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
        { id: "monitor-2", name: "Guangdong Mobile IPv4", type: "tcp", target: "gd-cm-v4.ip.zstaticcdn.com:80", interval_seconds: 60, auto_assign_new_nodes: true, assigned_node_count: 1, assigned_node_ids: ["node-1"], created_at: 1, updated_at: 1 },
      ],
      [
        { monitor_id: "monitor-1", node_id: "node-1", ts: 100, latency_ms: 23.5, success: true },
        { monitor_id: "monitor-2", node_id: "node-1", ts: 107, latency_ms: 48.1, success: true },
      ]
    )

    expect(chartData).toEqual([
      { ts: 100, "monitor-1": 23.5, "monitor-2": 48.1 },
    ])
    expect(resultsByTimestamp[100]).toEqual([
      { monitor_id: "monitor-1", node_id: "node-1", ts: 100, latency_ms: 23.5, success: true },
      { monitor_id: "monitor-2", node_id: "node-1", ts: 107, latency_ms: 48.1, success: true },
    ])
  })
})
