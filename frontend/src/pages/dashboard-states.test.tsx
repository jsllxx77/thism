import { beforeEach, describe, expect, it, vi } from "vitest"
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

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (error?: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe("dashboard page states", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    wsHandler = null
  })

  it("shows a loading state while node inventory is pending", async () => {
    const request = deferred<{ nodes: [] }>()
    nodesMock.mockReturnValue(request.promise)

    render(<Dashboard onSelectNode={() => {}} />)
    expect(screen.getByText("Loading node inventory...")).toBeInTheDocument()

    request.resolve({ nodes: [] })
    await waitFor(() => {
      expect(screen.queryByText("Loading node inventory...")).not.toBeInTheDocument()
    })
  })

  it("shows an error state when node loading fails", async () => {
    nodesMock.mockRejectedValue(new Error("network down"))

    render(<Dashboard onSelectNode={() => {}} />)

    expect(await screen.findByRole("alert")).toHaveTextContent("We couldn't load nodes. Please try again.")
  })

  it("refetches node inventory when refreshNonce changes", async () => {
    nodesMock.mockResolvedValue({ nodes: [] })

    const { rerender } = render(<Dashboard onSelectNode={() => {}} refreshNonce={0} />)
    await waitFor(() => {
      expect(nodesMock).toHaveBeenCalledTimes(1)
    })

    rerender(<Dashboard onSelectNode={() => {}} refreshNonce={1} />)
    await waitFor(() => {
      expect(nodesMock).toHaveBeenCalledTimes(2)
    })
  })

  it("computes average memory from valid samples only", async () => {
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 0 },
        { id: "n2", name: "beta", online: true, ip: "1.1.1.2", os: "linux", arch: "amd64", last_seen: 0 },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    act(() => {
      wsHandler?.({
        type: "metrics",
        payload: { node_id: "n1", data: { cpu: 20, mem: { used: 50, total: 100 } } },
      })
      wsHandler?.({
        type: "metrics",
        payload: { node_id: "n2", data: { cpu: 40, mem: { used: 0, total: 0 } } },
      })
    })

    await waitFor(() => {
      const avgMemoryLabel = screen.getByText("Avg Memory")
      const statCardContent = avgMemoryLabel.parentElement
      expect(statCardContent).not.toBeNull()
      expect(within(statCardContent as HTMLElement).getByText("50.0%")).toBeInTheDocument()
    })
  })

  it("hydrates initial dashboard metrics from the nodes response", async () => {
    nodesMock.mockResolvedValue({
      nodes: [
        {
          id: "n1",
          name: "alpha",
          online: true,
          ip: "1.1.1.1",
          os: "linux",
          arch: "amd64",
          last_seen: 0,
          latest_metrics: { ts: 1733011200, cpu: 37.5, mem_used: 50, mem_total: 100 },
        },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    expect(screen.getAllByText("37.5%").length).toBeGreaterThan(0)
    expect(screen.getAllByText("50.0%").length).toBeGreaterThan(0)
  })

  it("locks guest users to cards view only", async () => {
    nodesMock.mockResolvedValue({
      nodes: [
        {
          id: "n1",
          name: "alpha",
          online: true,
          ip: "1.1.1.1",
          os: "linux",
          arch: "amd64",
          last_seen: 0,
        },
      ],
    })

    render(<Dashboard onSelectNode={() => {}} accessMode="guest" />)

    expect(await screen.findByText("alpha")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Table View" })).not.toBeInTheDocument()
    expect(screen.queryByText("1.1.1.1")).not.toBeInTheDocument()
    expect(screen.getByText("linux/amd64")).toBeInTheDocument()
  })
})
