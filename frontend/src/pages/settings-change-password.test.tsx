import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Settings } from "./Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()
const dashboardSettingsMock = vi.fn()
const updateDashboardSettingsMock = vi.fn()
const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()
const versionMetaMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    updateMetricsRetention: (...args: unknown[]) => updateMetricsRetentionMock(...args),
    dashboardSettings: (...args: unknown[]) => dashboardSettingsMock(...args),
    updateDashboardSettings: (...args: unknown[]) => updateDashboardSettingsMock(...args),
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    versionMeta: (...args: unknown[]) => versionMetaMock(...args),
  },
}))

function renderSettings(path = "/settings?section=security") {
  window.history.replaceState({}, "", path)
  return render(<Settings />)
}

describe("settings change password", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    dashboardSettingsMock.mockReset()
    updateDashboardSettingsMock.mockReset()
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    versionMetaMock.mockReset()
    agentReleaseMock.mockImplementation((_os: string, arch: string) => Promise.resolve({ target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd", download_url: `https://example.com/${arch}`, sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64", check_interval_seconds: 1800 }))
    nodesMock.mockResolvedValue({ nodes: [] })
    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    dashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: true })
    updateDashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: true })
    notificationSettingsMock.mockResolvedValue({
      enabled: false,
      channel: "telegram",
      telegram_bot_token_set: false,
      telegram_targets: [],
      enabled_node_ids: [],
      node_scope_mode: "all",
      node_scope_node_ids: [],
      cpu_warning_percent: 85,
      cpu_critical_percent: 95,
      mem_warning_percent: 85,
      mem_critical_percent: 95,
      disk_warning_percent: 85,
      disk_critical_percent: 95,
      cooldown_minutes: 30,
      notify_node_offline: true,
      notify_node_online: false,
      node_offline_grace_minutes: 2,
    })
    updateNotificationSettingsMock.mockResolvedValue({
      enabled: false,
      channel: "telegram",
      telegram_bot_token_set: false,
      telegram_targets: [],
      enabled_node_ids: [],
      node_scope_mode: "all",
      node_scope_node_ids: [],
      cpu_warning_percent: 85,
      cpu_critical_percent: 95,
      mem_warning_percent: 85,
      mem_critical_percent: 95,
      disk_warning_percent: 85,
      disk_critical_percent: 95,
      cooldown_minutes: 30,
      notify_node_offline: true,
      notify_node_online: false,
      node_offline_grace_minutes: 2,
    })
    versionMetaMock.mockResolvedValue({ version: "1.0.0", commit: "abc", build_time: "2026-03-19T00:00:00Z" })
  })

  it("keeps update password disabled until the form is valid", async () => {
    const user = userEvent.setup()
    renderSettings()

    const updateButton = await screen.findByRole("button", { name: /update password/i })
    expect(updateButton).toBeDisabled()

    await user.type(screen.getByLabelText(/current password/i), "old-pass")
    expect(updateButton).toBeDisabled()

    await user.type(screen.getByLabelText(/^new password$/i), "old-pass")
    expect(updateButton).toBeDisabled()

    await user.type(screen.getByLabelText(/confirm new password/i), "different-pass")
    expect(updateButton).toBeDisabled()

    await user.clear(screen.getByLabelText(/^new password$/i))
    await user.type(screen.getByLabelText(/^new password$/i), "new-pass")
    expect(updateButton).toBeDisabled()

    await user.clear(screen.getByLabelText(/confirm new password/i))
    await user.type(screen.getByLabelText(/confirm new password/i), "new-pass")
    expect(updateButton).not.toBeDisabled()
    expect(changePasswordMock).not.toHaveBeenCalled()
  })

  it("submits change password request and shows success state", async () => {
    const user = userEvent.setup()
    changePasswordMock.mockResolvedValue({ ok: true })
    renderSettings()

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
    renderSettings()

    await user.type(screen.getByLabelText(/current password/i), "bad-old-pass")
    await user.type(screen.getByLabelText(/^new password$/i), "new-pass-123")
    await user.type(screen.getByLabelText(/confirm new password/i), "new-pass-123")
    await user.click(screen.getByRole("button", { name: /update password/i }))

    expect(await screen.findByRole("alert")).toHaveTextContent("invalid current password")
  })
})
