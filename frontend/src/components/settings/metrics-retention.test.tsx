import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Settings } from "../../pages/Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()
const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    updateMetricsRetention: (...args: unknown[]) => updateMetricsRetentionMock(...args),
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    versionMeta: () => Promise.resolve({ version: "1.0.0", commit: "abc", build_time: "2026-03-19T00:00:00Z" }),
  },
}))

describe("metrics retention settings", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()

    nodesMock.mockResolvedValue({ nodes: [] })
    agentReleaseMock.mockImplementation((_os: string, arch: string) =>
      Promise.resolve({
        target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd",
        download_url: `https://example.com/${arch}`,
        sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64",
        check_interval_seconds: 1800,
      }),
    )
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [7, 30] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    notificationSettingsMock.mockResolvedValue({
      enabled: false,
      channel: "telegram",
      telegram_bot_token_set: false,
      telegram_targets: [],
      cpu_warning_percent: 85,
      cpu_critical_percent: 95,
      mem_warning_percent: 85,
      mem_critical_percent: 95,
      disk_warning_percent: 85,
      disk_critical_percent: 95,
      cooldown_minutes: 30,
    })
    updateNotificationSettingsMock.mockResolvedValue({
      enabled: false,
      channel: "telegram",
      telegram_bot_token_set: false,
      telegram_targets: [],
      cpu_warning_percent: 85,
      cpu_critical_percent: 95,
      mem_warning_percent: 85,
      mem_critical_percent: 95,
      disk_warning_percent: 85,
      disk_critical_percent: 95,
      cooldown_minutes: 30,
    })
  })

  it("loads the current retention and saves a new retention period", async () => {
    const user = userEvent.setup()

    render(<Settings />)

    await user.click(await screen.findByRole("tab", { name: "Monitoring" }))
    expect(await screen.findByRole("heading", { name: "Metrics Retention", level: 3 })).toBeInTheDocument()
    expect(screen.getByRole("radio", { name: "30 days" })).toBeChecked()

    await user.click(screen.getByRole("radio", { name: "7 days" }))
    await user.click(screen.getByRole("button", { name: "Save retention" }))

    await waitFor(() => {
      expect(updateMetricsRetentionMock).toHaveBeenCalledWith(7)
    })
    expect(await screen.findByText("Metrics retention updated.")).toBeInTheDocument()
  })
})
