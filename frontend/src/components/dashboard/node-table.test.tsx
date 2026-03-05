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

    render(<NodeTable nodes={nodes} onSelectNode={onSelect} />)

    const rows = screen.getAllByRole("row")
    expect(rows.length).toBeGreaterThan(2)

    await user.click(screen.getByRole("button", { name: "Name" }))
    const bodyRowsAfterNameSort = screen.getAllByRole("row").slice(1)
    expect(within(bodyRowsAfterNameSort[0]).getByText("alpha")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Status" }))
    const bodyRowsAfterStatusSort = screen.getAllByRole("row").slice(1)
    expect(within(bodyRowsAfterStatusSort[0]).getByText("Online")).toBeInTheDocument()

    await user.click(within(bodyRowsAfterStatusSort[0]).getByText("alpha"))
    expect(onSelect).toHaveBeenCalledWith("n2")
  })
})
