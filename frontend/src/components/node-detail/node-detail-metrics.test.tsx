import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import type { Node } from "../../lib/api"
import { NodeHero } from "./NodeHero"
import { MetricTabs } from "./MetricTabs"
import { DiskHealthPanel } from "./DiskHealthPanel"

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
  disk_health: [
    {
      name: "nvme0n1",
      path: "/dev/nvme0n1",
      type: "nvme",
      model: "Fast NVMe",
      serial: "NVME123",
      firmware: "1.2.3",
      size_bytes: 1024000,
      status: "ok",
      temperature_c: 41,
      life_used_percent: 6,
      available_spare_percent: 92,
      power_on_hours: 1234,
      unsafe_shutdowns: 5,
      media_errors: 0,
    },
    {
      name: "sda",
      path: "/dev/sda",
      type: "ata",
      model: "Bulk SATA",
      size_bytes: 2048000,
      status: "critical",
      temperature_c: 43,
      power_on_hours: 2222,
      media_errors: 3,
      reallocated_sectors: 2,
      pending_sectors: 1,
      interface_crc_errors: 3,
      message: "ATA SMART reports unstable or uncorrectable sectors",
    },
  ],
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
        load1Data={points}
        cpuIOWaitData={points}
        cpuStealData={points}
        pressureCPUSomeData={points}
        pressureMemorySomeData={points}
        pressureMemoryFullData={points}
        pressureIOSomeData={points}
        pressureIOFullData={points}
        swapData={points}
        swapInSpeedData={points}
        swapOutSpeedData={points}
        oomKillsData={points}
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
    expect(screen.getByRole("heading", { name: "System Pressure" })).toBeInTheDocument()
    expect(screen.getByText("Load Average (1m)")).toBeInTheDocument()
    expect(screen.getByText("CPU I/O Wait")).toBeInTheDocument()
    expect(screen.getByText("CPU Steal")).toBeInTheDocument()
    expect(screen.getByText("CPU Pressure")).toBeInTheDocument()
    expect(screen.getByText("IO Pressure")).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Memory Pressure" })).toBeInTheDocument()
    expect(screen.getByText("Swap Usage")).toBeInTheDocument()
    expect(screen.getByText("Swap In Speed")).toBeInTheDocument()
    expect(screen.getByText("Swap Out Speed")).toBeInTheDocument()
    expect(screen.getByText("OOM Kills")).toBeInTheDocument()
    expect(screen.getByText("Memory Pressure")).toBeInTheDocument()
    const resourceSection = container.querySelector("section[aria-labelledby='resource-usage-heading'] .grid") as HTMLElement | null
    expect(resourceSection?.className).toContain("lg:grid-cols-3")
  })

  it("renders disk health details", () => {
    render(<DiskHealthPanel disks={node.disk_health ?? []} />)

    expect(screen.getByRole("heading", { name: "Disk Health" })).toBeInTheDocument()
    expect(screen.getByText("nvme0n1")).toBeInTheDocument()
    expect(screen.getByText("Fast NVMe")).toBeInTheDocument()
    expect(screen.getByText("OK")).toBeInTheDocument()
    expect(screen.getByText("41.0°C")).toBeInTheDocument()
    expect(screen.getByText("6.0%")).toBeInTheDocument()
    expect(screen.getByText("92.0%")).toBeInTheDocument()
    expect(screen.getByText("sda")).toBeInTheDocument()
    expect(screen.getByText("Critical")).toBeInTheDocument()
    expect(screen.getByText("Reallocated 2")).toBeInTheDocument()
    expect(screen.getByText("Pending 1")).toBeInTheDocument()
    expect(screen.getByText("CRC 3")).toBeInTheDocument()
    expect(screen.getByText(/ATA SMART reports unstable/)).toBeInTheDocument()
  })
})
