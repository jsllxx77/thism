import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../../lib/api"
import { NodesTable } from "./NodesTable"

function node(overrides: Partial<Node>): Node {
  return {
    id: "id",
    name: "name",
    ip: "127.0.0.1",
    os: "linux",
    arch: "amd64",
    created_at: 1733011200,
    last_seen: 1733011200,
    online: true,
    ...overrides,
  }
}

describe("settings nodes table", () => {
  it("renders, filters, sorts, and shows action buttons", async () => {
    const user = userEvent.setup()

    render(
      <NodesTable
        nodes={[
          node({ id: "n1", name: "zeta", online: false }),
          node({ id: "n2", name: "alpha", online: true }),
        ]}
      />
    )

    expect(screen.getAllByRole("button", { name: "View" }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole("button", { name: "Copy" }).length).toBeGreaterThan(0)
    expect(screen.getAllByRole("button", { name: "Remove" }).length).toBeGreaterThan(0)

    await user.selectOptions(screen.getByLabelText("Settings status filter"), "online")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.queryByText("zeta")).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText("Settings status filter"), "all")
    await user.click(screen.getByRole("button", { name: "Name" }))
    const rows = screen.getAllByRole("row").slice(1)
    expect(rows[0]).toHaveTextContent("alpha")
  })
})
