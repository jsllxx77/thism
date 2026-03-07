import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Settings } from "./Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
  },
}))

describe("settings change password", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    agentReleaseMock.mockImplementation((_os: string, arch: string) => Promise.resolve({ target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd", download_url: `https://example.com/${arch}`, sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64", check_interval_seconds: 1800 }))
    nodesMock.mockResolvedValue({ nodes: [] })
  })

  it("validates password confirmation before submitting", async () => {
    const user = userEvent.setup()
    render(<Settings />)

    await user.type(screen.getByLabelText(/current password/i), "old-pass")
    await user.type(screen.getByLabelText(/^new password$/i), "new-pass")
    await user.type(screen.getByLabelText(/confirm new password/i), "different-pass")
    await user.click(screen.getByRole("button", { name: /update password/i }))

    expect(screen.getByRole("alert")).toHaveTextContent("New password and confirmation do not match.")
    expect(changePasswordMock).not.toHaveBeenCalled()
  })

  it("submits change password request and shows success state", async () => {
    const user = userEvent.setup()
    changePasswordMock.mockResolvedValue({ ok: true })
    render(<Settings />)

    await user.type(screen.getByLabelText(/current password/i), "old-pass")
    await user.type(screen.getByLabelText(/^new password$/i), "new-pass-123")
    await user.type(screen.getByLabelText(/confirm new password/i), "new-pass-123")
    await user.click(screen.getByRole("button", { name: /update password/i }))

    await waitFor(() => {
      expect(changePasswordMock).toHaveBeenCalledWith("old-pass", "new-pass-123")
    })
    expect(await screen.findByText("Password updated successfully.")).toBeInTheDocument()

    expect(screen.getByLabelText(/current password/i).className).toContain("rounded-xl")
    expect(screen.getByRole("button", { name: /update password/i }).className).toContain("rounded-xl")
  })

  it("shows backend error when password change fails", async () => {
    const user = userEvent.setup()
    changePasswordMock.mockRejectedValue(new Error("invalid current password"))
    render(<Settings />)

    await user.type(screen.getByLabelText(/current password/i), "bad-old-pass")
    await user.type(screen.getByLabelText(/^new password$/i), "new-pass-123")
    await user.type(screen.getByLabelText(/confirm new password/i), "new-pass-123")
    await user.click(screen.getByRole("button", { name: /update password/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent("invalid current password")
  })
})
