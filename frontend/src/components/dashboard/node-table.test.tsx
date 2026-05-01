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

  it("shows agent versions with a fallback when unavailable", () => {
    render(
      <NodeTable
        nodes={[
          { ...node({ id: "n1", name: "alpha" }), agent_version: "cda21ec8f20b" } as Node,
          node({ id: "n2", name: "beta" }),
        ]}
        onSelectNode={vi.fn()}
      />
    )

    expect(screen.getByRole("columnheader", { name: "Agent" })).toBeInTheDocument()
    expect(screen.getByText("cda21ec8f20b")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
  })

  it("includes dark-safe classes for buttons and row text", () => {
    render(<NodeTable nodes={[node({ id: "n1", name: "alpha", online: true })]} onSelectNode={vi.fn()} />)

    const sortByName = screen.getByRole("button", { name: "Node Name" })
    const rowNameCell = screen.getByText("alpha")
    const rowNameButton = screen.getByRole("button", { name: "Open node alpha" })
    const statusCell = screen.getByText("Online")

    expect(sortByName.className).toContain("dark:hover:text-slate-200")
    expect(rowNameButton.className).toContain("dark:text-slate-100")
    expect(rowNameCell.tagName).toBe("SPAN")
    expect(statusCell.className).toContain("dark:text-slate-300")
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

    const rowButton = screen.getByRole("button", { name: "Open node alpha" })
    rowButton.focus()
    await user.keyboard("{Enter}")

    expect(onSelect).toHaveBeenCalledWith("n1")
  })

  it("shows a country flag before the node name in table view when country code is available", () => {
    render(<NodeTable nodes={[node({ id: "n1", name: "alpha", country_code: "HK", online: true })]} onSelectNode={vi.fn()} />)

    expect(screen.getByText("🇭🇰")).toBeInTheDocument()
    expect(screen.getByText("alpha")).toBeInTheDocument()
  })
})
