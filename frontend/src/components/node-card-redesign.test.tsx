import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../lib/api"
import { NodeCard } from "./NodeCard"

function createNode(overrides: Partial<Node> = {}): Node {
  return {
    id: "n1",
    name: "alpha",
    ip: "10.0.0.1",
    os: "linux",
    arch: "amd64",
    created_at: 0,
    last_seen: 1733011200,
    online: true,
    ...overrides,
  }
}

describe("node card redesign", () => {
  it("renders status and metrics and supports click", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()

    render(<NodeCard node={createNode()} cpu={42.3} memUsed={512} memTotal={1024} onClick={onClick} />)

    expect(screen.getByText("Online")).toBeInTheDocument()
    expect(screen.getByText("42.3%")).toBeInTheDocument()
    expect(screen.getByText("50.0%")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /alpha/i }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })

  it("shows offline state and last seen text", () => {
    render(<NodeCard node={createNode({ online: false })} cpu={0} memUsed={0} memTotal={1024} />)

    expect(screen.getByText("Offline")).toBeInTheDocument()
    expect(screen.getByText(/Last seen/i)).toBeInTheDocument()
  })
})
