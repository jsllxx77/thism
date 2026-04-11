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

function renderSettings(path = "/settings") {
  window.history.replaceState({}, "", path)
  return render(<Settings />)
}

describe("settings section tabs", () => {
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
    agentReleaseMock.mockImplementation((_os: string, arch: string) =>
      Promise.resolve({
        target_version: arch === "amd64" ? "aaaa1111bbbb" : "cccc2222dddd",
        download_url: `https://example.com/${arch}`,
        sha256: arch === "amd64" ? "sha-amd64" : "sha-arm64",
        check_interval_seconds: 1800,
      }),
    )
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
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 0,
      high_watermark: 0,
      enqueued: 0,
      processed: 0,
      dropped: 0,
    })
    window.scrollTo = vi.fn()
  })

  it("normalizes missing section to nodes", async () => {
    renderSettings("/settings")

    await waitFor(() => {
      expect(window.location.search).toBe("?section=nodes")
    })

    expect(await screen.findByRole("button", { name: /add node/i })).toBeInTheDocument()
  })

  it("normalizes invalid section to nodes", async () => {
    renderSettings("/settings?section=unknown")

    await waitFor(() => {
      expect(window.location.search).toBe("?section=nodes")
    })

    expect(await screen.findByRole("button", { name: /add node/i })).toBeInTheDocument()
  })

  it("defers section-specific data requests until the tab is visited", async () => {
    const user = userEvent.setup()
    renderSettings("/settings?section=nodes")

    await screen.findByRole("button", { name: /add node/i })

    expect(agentReleaseMock).not.toHaveBeenCalled()
    expect(metricsRetentionMock).not.toHaveBeenCalled()
    expect(dashboardSettingsMock).not.toHaveBeenCalled()
    expect(notificationSettingsMock).not.toHaveBeenCalled()
    expect(dispatcherRuntimeStatsMock).not.toHaveBeenCalled()
    expect(versionMetaMock).not.toHaveBeenCalled()

    await user.click(screen.getByRole("tab", { name: "Agent" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=agent")
    })
    expect(await screen.findByRole("heading", { name: "Automatic Updates", level: 3 })).toBeInTheDocument()
    await waitFor(() => {
      expect(agentReleaseMock).toHaveBeenCalledTimes(2)
    })

    await user.click(screen.getByRole("tab", { name: "Monitoring" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=monitoring")
    })
    expect(await screen.findByRole("heading", { name: "Metrics Retention", level: 3 })).toBeInTheDocument()
    await waitFor(() => {
      expect(metricsRetentionMock).toHaveBeenCalledTimes(1)
      expect(dashboardSettingsMock).toHaveBeenCalledTimes(1)
    })

    await user.click(screen.getByRole("tab", { name: "Alerts" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=alerts")
    })
    expect(await screen.findByRole("heading", { name: "Notifications", level: 3 })).toBeInTheDocument()
    await waitFor(() => {
      expect(notificationSettingsMock).toHaveBeenCalledTimes(1)
      expect(dispatcherRuntimeStatsMock).toHaveBeenCalledTimes(1)
    })

    await user.click(screen.getByRole("tab", { name: "Security" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=security")
    })
    expect(await screen.findByRole("heading", { name: "Version", level: 3 })).toBeInTheDocument()
    await waitFor(() => {
      expect(versionMetaMock).toHaveBeenCalledTimes(1)
    })
  })

  it("switches sections and updates the query string", async () => {
    const user = userEvent.setup()
    renderSettings("/settings?section=nodes")

    await screen.findByRole("button", { name: /add node/i })
    await user.click(screen.getByRole("tab", { name: "Agent" }))

    await waitFor(() => {
      expect(window.location.search).toBe("?section=agent")
    })

    expect(await screen.findByRole("heading", { name: "Automatic Updates", level: 3 })).toBeInTheDocument()
    expect(screen.getByRole("heading", { name: "Latency Monitors", level: 3 })).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /add node/i })).not.toBeInTheDocument()
  })

  it("preserves unsaved notification form input across section switches", async () => {
    const user = userEvent.setup()
    renderSettings("/settings?section=alerts")

    const botTokenInput = await screen.findByLabelText("Telegram bot token")
    await user.type(botTokenInput, "123456:ABC")

    await user.click(screen.getByRole("tab", { name: "Nodes" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=nodes")
    })

    await user.click(screen.getByRole("tab", { name: "Alerts" }))
    await waitFor(() => {
      expect(window.location.search).toBe("?section=alerts")
    })

    expect(await screen.findByLabelText("Telegram bot token")).toHaveValue("123456:ABC")
  })
})
