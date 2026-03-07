import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
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

describe("batch upgrade controls", () => {
  it("keeps batch upgrade controls hidden from the default settings table UI", () => {
    render(
      <NodesTable
        nodes={[
          node({ id: "n1", name: "alpha", online: true }),
          node({ id: "n2", name: "beta", online: false }),
        ]}
      />,
    )

    expect(screen.queryByRole("button", { name: /batch upgrade/i })).not.toBeInTheDocument()
    expect(screen.queryByRole("checkbox", { name: /select node/i })).not.toBeInTheDocument()
  })
})
