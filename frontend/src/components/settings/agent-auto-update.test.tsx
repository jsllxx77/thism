import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { Settings } from "../../pages/Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
  },
}))

describe("agent auto update status card", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    nodesMock.mockResolvedValue({ nodes: [] })
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
    render(<Settings />)

    expect(await screen.findByText("Automatic Updates")).toBeInTheDocument()
    expect(screen.getByText("On")).toBeInTheDocument()
    expect(screen.getByText("Checks every 30 minutes")).toBeInTheDocument()
    expect(screen.getByText("linux/amd64 · aaaa1111bbbb")).toBeInTheDocument()
    expect(screen.getByText("linux/arm64 · cccc2222dddd")).toBeInTheDocument()

    await waitFor(() => {
      expect(screen.queryByRole("button", { name: /batch upgrade/i })).not.toBeInTheDocument()
    })
  })
})
