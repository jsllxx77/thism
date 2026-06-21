import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import type { Node } from "../../lib/api"
import { NodeHero } from "./NodeHero"
import { MetricTabs } from "./MetricTabs"

const node: Node = {
  id: "node-1",
  name: "alpha",
  country_code: "HK",
  ip: "10.0.0.7",
  os: "linux",
  arch: "amd64",
  created_at: 0,
  last_seen: 1733011200,
  online: true,
  latest_metrics: {
    ts: 1733011200,
    cpu: 37.5,
    mem_used: 2048,
    mem_total: 4096,
    disk_used: 8192,
    disk_total: 16384,
    net_rx: 1234,
    net_tx: 5678,
    uptime_seconds: 93784,
  },
  hardware: {
    cpu_model: "AMD EPYC 7B13",
    cpu_cores: 8,
    cpu_threads: 16,
    memory_total: 34359738368,
    disk_total: 322122547200,
    virtualization_system: "kvm",
    virtualization_role: "guest",
  },
}

describe("node detail metrics", () => {
  it("renders hero with node identity", () => {
    const { container } = render(<NodeHero node={node} />)

    expect(screen.getByText("Node")).toBeInTheDocument()
    expect(screen.getByRole("img", { name: "HK" })).toHaveClass("country-flag")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getByText("Online")).toBeInTheDocument()
    expect(screen.getByText(/10.0.0.7/)).toBeInTheDocument()
    expect(screen.getByText(/Uptime 1d 2h 3m/)).toBeInTheDocument()
    expect(screen.queryByText(/Last seen/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Node ID/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Heartbeat/i)).not.toBeInTheDocument()
    expect(screen.queryByText("CPU Model")).not.toBeInTheDocument()
    expect(screen.queryByText("Cores / Threads")).not.toBeInTheDocument()

    const heroCard = container.firstElementChild as HTMLElement | null
    expect(heroCard?.className).toContain("enterprise-hero")
  })

  it("renders grouped metric charts", () => {
    const points = [
      { ts: 1700000000, value: 30 },
      { ts: 1700000300, value: 44 },
    ]

    const { container } = render(
      <MetricTabs
        range={3600}
        cpuData={points}
        memData={points}
        netRxData={points}
        netTxData={points}
        netRxSpeedData={points}
        netTxSpeedData={points}
        diskData={points}
        diskReadSpeedData={points}
        diskWriteSpeedData={points}
      />
    )

    expect(screen.getByRole("heading", { name: "Resource Usage" })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Throughput / Traffic" })).toBeInTheDocument()
    expect(screen.queryByRole("tab")).not.toBeInTheDocument()
    expect(screen.getByText("CPU Usage")).toBeInTheDocument()
    expect(screen.getByText("Memory Usage")).toBeInTheDocument()
    expect(screen.getByText("Disk Usage")).toBeInTheDocument()
    expect(screen.getByText("Inbound Traffic")).toBeInTheDocument()
    expect(screen.getByText("Inbound Speed")).toBeInTheDocument()
    expect(screen.getByLabelText("Disk IO")).toBeInTheDocument()
    expect(screen.getByText("Disk Read Speed")).toBeInTheDocument()
    expect(screen.getByText("Disk Write Speed")).toBeInTheDocument()
    const resourceSection = container.querySelector("section[aria-labelledby='resource-usage-heading'] .grid") as HTMLElement | null
    expect(resourceSection?.className).toContain("lg:grid-cols-3")
  })
})
