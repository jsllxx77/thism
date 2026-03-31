import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Settings } from "../../pages/Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const createAgentUpdateJobMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
    createAgentUpdateJob: (...args: unknown[]) => createAgentUpdateJobMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    updateMetricsRetention: (...args: unknown[]) => updateMetricsRetentionMock(...args),
  },
}))

describe("agent auto update status card", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    createAgentUpdateJobMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    nodesMock.mockResolvedValue({ nodes: [] })
    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    createAgentUpdateJobMock.mockResolvedValue({
      job: { id: "job-1", kind: "self_update", target_version: "v-next", download_url: "https://example.com", sha256: "sha", created_at: 0, updated_at: 0, created_by: "admin", status: "pending" },
      targets: [],
    })
    agentReleaseMock.mockImplementation((_os: string, arch: string) =>
      Promise.resolve({
        target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd",
        download_url: `https://example.com/${arch}`,
        sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64",
        check_interval_seconds: 1800,
      }),
    )
  })

  it("shows auto update status and hides batch upgrade controls", async () => {
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "node-1", name: "alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 0, last_seen: 0, online: true },
      ],
    })

    render(<Settings />)

    expect(await screen.findByText("Automatic Updates")).toBeInTheDocument()
    expect(screen.getByText("On")).toBeInTheDocument()
    expect(screen.getByText("Checks every 30 minutes")).toBeInTheDocument()
    expect(screen.getByText("linux/amd64 · aaaa1111bbbb")).toBeInTheDocument()
    expect(screen.getByText("linux/arm64 · cccc2222dddd")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Update now" })).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /batch upgrade/i })).not.toBeInTheDocument()
    })
  })

  it("dispatches immediate updates for eligible online nodes grouped by arch", async () => {
    const user = userEvent.setup()
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "node-1", name: "alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 0, last_seen: 0, online: true },
        { id: "node-2", name: "beta", ip: "2.2.2.2", os: "linux", arch: "arm64", created_at: 0, last_seen: 0, online: true },
        { id: "node-3", name: "gamma", ip: "3.3.3.3", os: "linux", arch: "amd64", created_at: 0, last_seen: 0, online: false },
      ],
    })

    render(<Settings />)

    await screen.findByRole("button", { name: "Update now" })
    await user.click(screen.getByRole("button", { name: "Update now" }))

    await waitFor(() => {
      expect(createAgentUpdateJobMock).toHaveBeenCalledTimes(2)
    })

    expect(createAgentUpdateJobMock).toHaveBeenNthCalledWith(1, ["node-1"], "aaaa1111bbbb", "https://example.com/amd64", "sha-amd64")
    expect(createAgentUpdateJobMock).toHaveBeenNthCalledWith(2, ["node-2"], "cccc2222dddd", "https://example.com/arm64", "sha-arm64")
    expect(screen.getByText("Queued updates for 2 nodes.")).toBeInTheDocument()
  })

  it("disables immediate updates when all online agents already run the latest version", async () => {
    const user = userEvent.setup()
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "node-1", name: "alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", agent_version: "aaaa1111bbbb", created_at: 0, last_seen: 0, online: true },
        { id: "node-2", name: "beta", ip: "2.2.2.2", os: "linux", arch: "arm64", agent_version: "cccc2222dddd", created_at: 0, last_seen: 0, online: true },
      ],
    })

    render(<Settings />)

    const button = await screen.findByRole("button", { name: "Update now" })
    expect(button).toBeDisabled()
    expect(screen.getByText("0 online agents can update now")).toBeInTheDocument()

    await user.click(button)

    expect(createAgentUpdateJobMock).not.toHaveBeenCalled()
  })
})
