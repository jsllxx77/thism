import { beforeEach, describe, expect, it, vi } from "vitest"
import { act, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { ComponentProps } from "react"
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

function renderSettings(path = "/settings?section=nodes", props?: ComponentProps<typeof Settings>) {
  window.history.replaceState({}, "", path)
  return render(<Settings {...props} />)
}

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
    vi.useRealTimers()
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
    metricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 7, options: [7, 30] })
    dashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: true })
    updateDashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: false })
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

  it("shows a loading state while settings data is pending", async () => {
    const request = deferred<{ nodes: [] }>()
    nodesMock.mockReturnValue(request.promise)

    renderSettings("/settings?section=nodes")
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

    const user = userEvent.setup()
    const { container } = renderSettings("/settings?section=nodes")

    await waitFor(() => {
      expect(screen.getByRole("heading", { name: "Nodes", level: 4 })).toBeInTheDocument()
    })

    const nodeRegistry = screen.getByRole("heading", { name: "Nodes", level: 4 }).closest("section") as HTMLElement | null

    expect(nodeRegistry?.className).toContain("enterprise-surface")

    const addNodeButton = screen.getByRole("button", { name: /add node/i })
    expect(addNodeButton.className).toContain("rounded-xl")

    window.scrollTo = vi.fn()
    await user.click(screen.getByRole("tab", { name: "Security" }))

    const securityCard = await screen.findByRole("heading", { name: "Password", level: 4 })
    expect(securityCard.closest("section")?.className).toContain("enterprise-surface")

    expect(container.textContent).toContain("Settings")
  })

  it("shows an error state when settings data fails to load", async () => {
    nodesMock.mockRejectedValue(new Error("timeout"))

    renderSettings("/settings?section=nodes")
    expect(await screen.findByText("We couldn't load settings data. Please try again.")).toBeInTheDocument()
  })

  it("refetches settings data when refreshNonce changes", async () => {
    nodesMock.mockResolvedValue({ nodes: [] })

    const { rerender } = renderSettings("/settings?section=nodes", { refreshNonce: 0 })
    let initialCalls = 0
    await waitFor(() => {
      initialCalls = nodesMock.mock.calls.length
      expect(initialCalls).toBeGreaterThanOrEqual(1)
    })

    rerender(<Settings refreshNonce={1} />)
    await waitFor(() => {
      expect(nodesMock.mock.calls.length).toBeGreaterThan(initialCalls)
    })
  })

  it("shows recently seen nodes as online during the same grace period used by the dashboard", async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date("2026-03-06T00:01:00Z"))

    nodesMock.mockResolvedValue({
      nodes: [
        {
          id: "node-1",
          name: "alpha",
          ip: "1.1.1.1",
          os: "linux",
          arch: "amd64",
          created_at: 0,
          last_seen: 1772755260,
          online: false,
        },
      ],
    })

    renderSettings("/settings?section=nodes")

    await act(async () => {
      await Promise.resolve()
    })

    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getAllByText("Online").length).toBeGreaterThanOrEqual(1)

    act(() => {
      vi.advanceTimersByTime(16_000)
    })

    expect(screen.getAllByText("Offline").length).toBeGreaterThanOrEqual(1)
    vi.useRealTimers()
  })

  it("loads and saves dashboard card IP visibility", async () => {
    const user = userEvent.setup()
    nodesMock.mockResolvedValue({ nodes: [] })

    renderSettings("/settings?section=monitoring")

    const checkbox = await screen.findByRole("checkbox", { name: "Show IP addresses on dashboard node cards" })
    expect(checkbox).toBeChecked()

    await user.click(checkbox)
    await user.click(screen.getByRole("button", { name: "Save dashboard visibility" }))

    await waitFor(() => {
      expect(updateDashboardSettingsMock).toHaveBeenCalledWith({ show_dashboard_card_ip: false })
    })
    expect(await screen.findByText("Dashboard card visibility updated.")).toBeInTheDocument()
  })
})
