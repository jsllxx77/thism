import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import type { NodeHardware } from "../../lib/api"
import { HardwarePassport } from "./HardwarePassport"

function createHardware(overrides: Partial<NodeHardware> = {}): NodeHardware {
  return {
    cpu_model: "AMD EPYC 7B13",
    cpu_cores: 8,
    cpu_threads: 16,
    memory_total: 34359738368,
    disk_total: 322122547200,
    virtualization_system: "kvm",
    virtualization_role: "guest",
    ...overrides,
  }
}

describe("hardware passport", () => {
  it("renders hardware identity fields", () => {
    render(<HardwarePassport hardware={createHardware()} os="linux" arch="amd64" />)

    expect(screen.getByText("Hardware")).toBeInTheDocument()
    expect(screen.getByText("AMD EPYC 7B13")).toBeInTheDocument()
    expect(screen.getByText("8 / 16")).toBeInTheDocument()
    expect(screen.getByText("32.0 GB")).toBeInTheDocument()
    expect(screen.getByText("300.0 GB")).toBeInTheDocument()
    expect(screen.getByText("KVM · guest")).toBeInTheDocument()
    expect(screen.getByText("linux / amd64")).toBeInTheDocument()
  })

  it("renders placeholders for missing values", () => {
    render(<HardwarePassport hardware={createHardware({ cpu_model: "", memory_total: 0, disk_total: 0, virtualization_system: "", virtualization_role: "" })} os="" arch="" />)

    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(4)
  })

  it("uses a dedicated light-mode passport surface instead of a dark-only shell", () => {
    const { container } = render(<HardwarePassport hardware={createHardware()} os="linux" arch="amd64" />)

    const section = container.querySelector("section")
    expect(section).toBeTruthy()
    expect(section?.className).toContain("enterprise-hero")

    const firstValueCard = screen.getByText("8 / 16").closest("div.rounded-2xl")
    expect(firstValueCard?.className).toContain("enterprise-inner-surface")
  })
})
