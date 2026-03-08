import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { Settings } from "./Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    updateMetricsRetention: (...args: unknown[]) => updateMetricsRetentionMock(...args),
  },
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

describe("settings page states", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    agentReleaseMock.mockImplementation((_os: string, arch: string) => Promise.resolve({ target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd", download_url: `https://example.com/${arch}`, sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64", check_interval_seconds: 1800 }))
    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
  })

  it("shows a loading state while settings data is pending", async () => {
    const request = deferred<{ nodes: [] }>()
    nodesMock.mockReturnValue(request.promise)

    render(<Settings />)
    expect(screen.getByText("Loading nodes...")).toBeInTheDocument()

    request.resolve({ nodes: [] })
    await waitFor(() => {
      expect(screen.queryByText("Loading nodes...")).not.toBeInTheDocument()
    })
  })

  it("uses the engineering card shell for settings sections", async () => {
    nodesMock.mockResolvedValue({
      nodes: [
        {
          id: "node-1",
          name: "alpha",
          ip: "1.1.1.1",
          os: "linux",
          arch: "amd64",
          created_at: 0,
          last_seen: 0,
          online: true,
        },
      ],
    })

    const { container } = render(<Settings />)

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Nodes", level: 4 })).toBeInTheDocument()
    })

    const nodeRegistry = screen.getByRole("heading", { name: "Nodes", level: 4 }).closest("section") as HTMLElement | null
    const securityCard = screen.getByRole("heading", { name: "Password", level: 4 }).closest("section") as HTMLElement | null

    expect(nodeRegistry?.className).toContain("enterprise-surface")
    expect(securityCard?.className).toContain("enterprise-surface")

    const addNodeButton = screen.getByRole("button", { name: /add node/i })
    expect(addNodeButton.className).toContain("rounded-xl")

    expect(container.textContent).toContain("Settings")
  })

  it("shows an error state when settings data fails to load", async () => {
    nodesMock.mockRejectedValue(new Error("timeout"))

    render(<Settings />)
    expect(await screen.findByRole("alert")).toHaveTextContent("We couldn't load settings data. Please try again.")
  })

  it("refetches settings data when refreshNonce changes", async () => {
    nodesMock.mockResolvedValue({ nodes: [] })

    const { rerender } = render(<Settings refreshNonce={0} />)
    await waitFor(() => {
      expect(nodesMock).toHaveBeenCalledTimes(1)
    })

    rerender(<Settings refreshNonce={1} />)
    await waitFor(() => {
      expect(nodesMock).toHaveBeenCalledTimes(2)
    })
  })
})
