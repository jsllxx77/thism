import { describe, expect, it, vi, afterEach } from "vitest"
import { act, render, screen } from "@testing-library/react"
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
    last_seen: 1772755250,
    online: true,
    ...overrides,
  }
}

describe("node card live last seen", () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it("refreshes the relative last seen label over time", () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-03-06T00:01:00Z"))

    render(<NodeCard node={createNode()} cpu={0} memUsed={0} memTotal={1024} />)

    expect(screen.getByText("Last seen 10s ago")).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(65_000)
    })

    expect(screen.getByText("Last seen 1m ago")).toBeInTheDocument()
  })
})
