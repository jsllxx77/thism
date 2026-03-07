import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, render, screen, waitFor, within } from "@testing-library/react"
import { Dashboard } from "./Dashboard"

const nodesMock = vi.fn()
let wsHandler: ((msg: { type: string; payload?: unknown }) => void) | null = null

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: (handler: (msg: { type: string; payload?: unknown }) => void) => {
      wsHandler = handler
    },
    off: () => {
      wsHandler = null
    },
  }),
}))

describe("dashboard last seen updates", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    wsHandler = null
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("updates last seen when metrics messages include a newer timestamp", async () => {
    vi.spyOn(Date, "now").mockReturnValue(1772755260000)

    nodesMock.mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 1772755200 },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })
    expect(screen.getByText("Last seen 1m ago")).toBeInTheDocument()

    act(() => {
      wsHandler?.({
        type: "metrics",
        payload: {
          node_id: "n1",
          last_seen: 1772755260,
          data: { cpu: 20, mem: { used: 50, total: 100 } },
        },
      })
    })

    expect(screen.getByText("Last seen just now")).toBeInTheDocument()
  })

  it("keeps a node effectively online for a short grace window after disconnect", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-03-06T00:01:00Z"))

    nodesMock.mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 1772755260 },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} />)

    await act(async () => {
      await Promise.resolve()
    })

    expect(screen.getByText("alpha")).toBeInTheDocument()

    const nodeCard = screen.getByRole("button", { name: "alpha" })
    expect(within(nodeCard).getByText("Online")).toBeInTheDocument()

    act(() => {
      wsHandler?.({
        type: "node_status",
        payload: { node_id: "n1", online: false },
      })
    })

    expect(within(nodeCard).getByText("Online")).toBeInTheDocument()
    expect(screen.getByText("Last seen just now")).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(16_000)
    })

    expect(within(nodeCard).getByText("Offline")).toBeInTheDocument()
    expect(screen.getByText("Last seen 16s ago")).toBeInTheDocument()
  })
})
