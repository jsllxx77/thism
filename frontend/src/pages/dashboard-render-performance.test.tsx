import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, render, screen } from "@testing-library/react"
import { Dashboard } from "./Dashboard"

const nodesMock = vi.fn()
const renderCounts = new Map<string, number>()

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: () => {},
    off: () => {},
  }),
}))

vi.mock("../components/NodeCard", () => ({
  NodeCard: ({ node }: { node: { id: string; name: string } }) => {
    renderCounts.set(node.id, (renderCounts.get(node.id) ?? 0) + 1)
    return <div data-testid={`node-card-${node.id}`}>{node.name}</div>
  },
}))

describe("dashboard render performance", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    renderCounts.clear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("does not rerender every node card each second when no dashboard data changes", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-03-06T00:01:00Z"))

    nodesMock.mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 1772755250 },
        { id: "n2", name: "beta", online: true, ip: "1.1.1.2", os: "linux", arch: "arm64", last_seen: 1772755250 },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} />)

    await act(async () => {
      await Promise.resolve()
      await Promise.resolve()
    })

    expect(screen.getByTestId("node-card-n1")).toBeInTheDocument()
    expect(screen.getByTestId("node-card-n2")).toBeInTheDocument()

    const initialCounts = {
      n1: renderCounts.get("n1") ?? 0,
      n2: renderCounts.get("n2") ?? 0,
    }

    await act(async () => {
      vi.advanceTimersByTime(5_000)
      await Promise.resolve()
    })

    expect(renderCounts.get("n1") ?? 0).toBe(initialCounts.n1)
    expect(renderCounts.get("n2") ?? 0).toBe(initialCounts.n2)
  })
})
