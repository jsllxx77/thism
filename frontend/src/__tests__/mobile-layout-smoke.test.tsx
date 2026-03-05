import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import type { Node } from "../lib/api"
import { NodeTable } from "../components/dashboard/NodeTable"
import { NodesTable } from "../components/settings/NodesTable"

const nodes: Node[] = [
  {
    id: "n1",
    name: "alpha",
    ip: "1.1.1.1",
    os: "linux",
    arch: "amd64",
    created_at: 1733011200,
    last_seen: 1733011200,
    online: true,
  },
]

describe("mobile layout smoke", () => {
  it("keeps dense table sections horizontally scrollable", () => {
    const { container } = render(<NodeTable nodes={nodes} onSelectNode={() => {}} />)
    expect(container.querySelector(".overflow-x-auto")).toBeInTheDocument()

    render(<NodesTable nodes={nodes} />)
    expect(screen.getAllByText("alpha").length).toBeGreaterThan(0)
  })
})
