import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { Settings } from "./Settings"

function renderAlertsSettings() {
  window.history.replaceState({}, "", "/settings?section=alerts")
  return render(<Settings />)
}

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
const dispatcherRuntimeStatsMock = vi.fn()

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
    dispatcherRuntimeStats: (...args: unknown[]) => dispatcherRuntimeStatsMock(...args),
  },
}))

describe("settings dispatcher health", () => {
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
    dispatcherRuntimeStatsMock.mockReset()

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

  it("renders dispatcher health inside notifications without a standalone diagnostics card", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 2,
      total_capacity: 512,
      queue_depth: 7,
      high_watermark: 12,
      enqueued: 120,
      processed: 119,
      dropped: 1,
    })

    renderAlertsSettings()

    expect(await screen.findByText("Alert delivery status")).toBeInTheDocument()
    expect(screen.getByText("Drops detected")).toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Dispatcher Diagnostics", level: 3 })).not.toBeInTheDocument()
    expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
  })

  it("does not render dispatcher health when diagnostics fail to load", async () => {
    dispatcherRuntimeStatsMock.mockRejectedValue(new Error("boom"))

    renderAlertsSettings()

    await waitFor(() => {
      expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
    })
    expect(screen.queryByText("Alert delivery status")).not.toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Dispatcher Diagnostics", level: 3 })).not.toBeInTheDocument()
  })

  it("loads dispatcher health once through the notifications card", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 0,
      high_watermark: 0,
      enqueued: 0,
      processed: 0,
      dropped: 0,
    })

    renderAlertsSettings()

    await waitFor(() => {
      expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
    })
  })
})
