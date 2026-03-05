import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Process, ServiceCheck } from "../../lib/api"
import { ProcessTable } from "./ProcessTable"
import { ServiceStatusList } from "./ServiceStatusList"

const processes: Process[] = [
  { pid: 1234, name: "nginx", cpu: 12.4, mem: 52428800 },
]

const services: ServiceCheck[] = [
  { name: "nginx.service", status: "running", last_checked: 1733011200 },
  { name: "cron.service", status: "failed", last_checked: 1733011200 },
]

describe("node detail collapsible sections", () => {
  it("expands process panel to show rows", async () => {
    const user = userEvent.setup()
    render(<ProcessTable processes={processes} />)

    expect(screen.queryByText("nginx")).not.toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: /Top Processes/i }))
    expect(screen.getByText("nginx")).toBeInTheDocument()
  })

  it("shows process rows by default when configured open", () => {
    render(<ProcessTable processes={processes} defaultOpen />)
    expect(screen.getByText("nginx")).toBeInTheDocument()
  })

  it("expands services panel and renders status semantics", async () => {
    const user = userEvent.setup()
    render(<ServiceStatusList services={services} />)

    await user.click(screen.getByRole("button", { name: /Services/i }))
    expect(screen.getByText("nginx.service")).toBeInTheDocument()
    expect(screen.getByText("running")).toHaveAttribute("data-status", "running")
    expect(screen.getByText("failed")).toHaveAttribute("data-status", "failed")
  })

  it("shows services by default when configured open", () => {
    render(<ServiceStatusList services={services} defaultOpen />)
    expect(screen.getByText("nginx.service")).toBeInTheDocument()
  })
})
