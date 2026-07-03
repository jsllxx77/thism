import { describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../../lib/api"
import { NodeTable } from "./NodeTable"

function node(overrides: Partial<Node>): Node {
  return {
    id: "id",
    name: "name",
    ip: "1.1.1.1",
    os: "linux",
    arch: "amd64",
    created_at: 0,
    last_seen: 0,
    online: true,
    latest_metrics: {
      ts: 100,
      cpu: 24.4,
      mem_used: 3 * 1024 ** 3,
      mem_total: 8 * 1024 ** 3,
      disk_used: 40 * 1024 ** 3,
      disk_total: 100 * 1024 ** 3,
      disk_read_bytes: 0,
      disk_write_bytes: 0,
      net_rx: 12 * 1024 ** 3,
      net_tx: 2 * 1024 ** 3,
    },
    ...overrides,
  }
}

describe("node table", () => {
  it("renders rows, supports sorting, and handles row click", async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()
    const nodes: Node[] = [
      node({ id: "n1", name: "zeta", online: false }),
      node({ id: "n2", name: "alpha", online: true }),
    ]

    const { container } = render(<NodeTable nodes={nodes} onSelectNode={onSelect} />)
    expect(screen.getByText("Node Inventory")).toBeInTheDocument()

    const tableShell = container.firstElementChild as HTMLElement | null
    expect(tableShell?.className).toContain("enterprise-surface")

    const rows = screen.getAllByRole("row")
    expect(rows.length).toBeGreaterThan(2)

    await user.click(screen.getByRole("button", { name: "Node Name" }))
    const bodyRowsAfterNameSort = screen.getAllByRole("row").slice(1)
    expect(within(bodyRowsAfterNameSort[0]).getByText("alpha")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Status" }))
    const bodyRowsAfterStatusSort = screen.getAllByRole("row").slice(1)
    expect(within(bodyRowsAfterStatusSort[0]).getByText("Online")).toBeInTheDocument()

    await user.click(within(bodyRowsAfterStatusSort[0]).getByText("alpha"))
    expect(onSelect).toHaveBeenCalledWith("n2")
  })

  it("shows operational metric columns with formatted resource values", () => {
    const { container } = render(
      <NodeTable
        nodes={[
          node({ id: "n1", name: "alpha", os: "linux", arch: "amd64" }),
          node({ id: "n2", name: "beta", latest_metrics: null }),
        ]}
        onSelectNode={vi.fn()}
      />
    )
    const table = container.querySelector("table") as HTMLElement

    expect(screen.getByRole("columnheader", { name: "Operating System" })).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "CPU" })).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "Memory" })).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "Disk" })).toBeInTheDocument()
    expect(screen.getByRole("columnheader", { name: "Network" })).toBeInTheDocument()
    expect(within(table).getAllByText("linux/amd64").length).toBeGreaterThan(0)
    expect(within(table).getByText("24.4%")).toBeInTheDocument()
    expect(within(table).getByText("24.4%").parentElement?.parentElement?.className).toContain("grid-rows-[1.25rem_0.375rem_1rem]")
    expect(within(table).getByText("37.5%")).toBeInTheDocument()
    expect(within(table).getByText("40.0%")).toBeInTheDocument()
    expect(within(table).getByText("↓ 12G")).toBeInTheDocument()
    expect(within(table).getByText("↑ 2G")).toBeInTheDocument()
  })

  it("includes dark-safe classes for buttons and row text", () => {
    const { container } = render(<NodeTable nodes={[node({ id: "n1", name: "alpha", online: true })]} onSelectNode={vi.fn()} />)
    const table = container.querySelector("table") as HTMLElement

    const sortByName = screen.getByRole("button", { name: "Node Name" })
    const rowNameCell = within(table).getByText("alpha")
    const rowNameButton = within(table).getByRole("button", { name: "Open node alpha" })
    const statusCell = within(table).getByText("Online")
    const osCell = within(table).getByText("linux/amd64")

    expect(sortByName.className).toContain("dark:hover:text-slate-200")
    expect(rowNameButton.className).toContain("dark:text-slate-100")
    expect(rowNameCell.tagName).toBe("SPAN")
    expect(osCell.className).toContain("dark:text-slate-300")
    expect(statusCell.className).toContain("dark:text-emerald-300")
  })

  it("supports keyboard activation on rows", async () => {
    const user = userEvent.setup()
    const onSelect = vi.fn()

    render(
      <NodeTable
        nodes={[node({ id: "n1", name: "alpha", online: true })]}
        onSelectNode={onSelect}
      />
    )

    const rowButton = screen.getAllByRole("button", { name: "Open node alpha" })[0]
    rowButton.focus()
    await user.keyboard("{Enter}")

    expect(onSelect).toHaveBeenCalledWith("n1")
  })

  it("shows a country flag before the node name in table view when country code is available", () => {
    const { container } = render(<NodeTable nodes={[node({ id: "n1", name: "alpha", country_code: "HK", online: true })]} onSelectNode={vi.fn()} />)
    const table = container.querySelector("table") as HTMLElement

    const flag = screen.getAllByRole("img", { name: "HK" })[0]
    expect(flag).toHaveClass("country-flag")
    expect(flag.querySelector("img")).toHaveAttribute("src", "/assets/flags/HK.svg")
    expect(within(table).getByText("alpha")).toBeInTheDocument()
  })

  it("uses live metrics when available", () => {
    const { container } = render(
      <NodeTable
        nodes={[node({ id: "n1", name: "alpha" })]}
        liveMetrics={{
          n1: {
            cpu: 55.5,
            memUsed: 6 * 1024 ** 3,
            memTotal: 8 * 1024 ** 3,
            netRxSpeed: 128 * 1024,
            netTxSpeed: 64 * 1024,
          },
        }}
        onSelectNode={vi.fn()}
      />
    )
    const table = container.querySelector("table") as HTMLElement

    expect(within(table).getByText("55.5%")).toBeInTheDocument()
    expect(within(table).getByText("75.0%")).toBeInTheDocument()
    expect(within(table).getByText("↓ 128K/s")).toBeInTheDocument()
    expect(within(table).getByText("↑ 64K/s")).toBeInTheDocument()
  })
})
