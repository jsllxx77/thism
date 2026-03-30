import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
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

describe("settings dispatcher diagnostics", () => {
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

  it("renders dispatcher runtime diagnostics", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 2,
      total_capacity: 512,
      queue_depth: 7,
      high_watermark: 12,
      enqueued: 120,
      processed: 119,
      dropped: 1,
    })

    render(<Settings />)

    expect(await screen.findByRole("heading", { name: "Dispatcher Diagnostics", level: 3 })).toBeInTheDocument()
    expect(screen.getByText("512")).toBeInTheDocument()
    expect(screen.getByText("12")).toBeInTheDocument()
    expect(screen.getByText("119")).toBeInTheDocument()
    expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
  })

  it("shows an error state when dispatcher diagnostics fail to load", async () => {
    dispatcherRuntimeStatsMock.mockRejectedValue(new Error("boom"))

    render(<Settings />)

    expect(await screen.findByText("Dispatcher diagnostics are currently unavailable.")).toBeInTheDocument()
  })

  it("refreshes dispatcher diagnostics on refreshNonce changes", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 0,
      high_watermark: 0,
      enqueued: 0,
      processed: 0,
      dropped: 0,
    })

    const { rerender } = render(<Settings refreshNonce={0} />)

    await waitFor(() => {
      expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
    })

    rerender(<Settings refreshNonce={1} />)

    await waitFor(() => {
      expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(2)
    })
  })
})
