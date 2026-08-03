import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"

import { AppThemeProvider } from "../theme/theme"
import { ThemeModeProvider } from "../theme/mode"
import { Settings } from "./Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()
const geoIPSettingsMock = vi.fn()
const updateGeoIPSettingsMock = vi.fn()
const updateGeoIPDatabaseMock = vi.fn()
const dashboardSettingsMock = vi.fn()
const updateDashboardSettingsMock = vi.fn()
const themeSettingsMock = vi.fn()
const updateThemeSettingsMock = vi.fn()
const importThemeArchiveMock = vi.fn()
const importThemeFromGitHubMock = vi.fn()
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
    geoIPSettings: (...args: unknown[]) => geoIPSettingsMock(...args),
    updateGeoIPSettings: (...args: unknown[]) => updateGeoIPSettingsMock(...args),
    updateGeoIPDatabase: (...args: unknown[]) => updateGeoIPDatabaseMock(...args),
    dashboardSettings: (...args: unknown[]) => dashboardSettingsMock(...args),
    updateDashboardSettings: (...args: unknown[]) => updateDashboardSettingsMock(...args),
    themeSettings: (...args: unknown[]) => themeSettingsMock(...args),
    updateThemeSettings: (...args: unknown[]) => updateThemeSettingsMock(...args),
    importThemeArchive: (...args: unknown[]) => importThemeArchiveMock(...args),
    importThemeFromGitHub: (...args: unknown[]) => importThemeFromGitHubMock(...args),
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    versionMeta: (...args: unknown[]) => versionMetaMock(...args),
  },
}))

const auroraThemePackage = {
  type: "thism-theme",
  version: 1,
  id: "aurora-command",
  name: "Aurora Command",
  description: "A high-contrast operations theme.",
  accent: "#8b5cf6",
  tokens: {
    light: {
      background: "248 80% 98%",
      foreground: "246 38% 12%",
      card: "0 0% 100%",
      "card-foreground": "246 38% 12%",
      primary: "265 83% 58%",
      "primary-foreground": "0 0% 100%",
      border: "252 30% 84%",
      input: "252 30% 82%",
      ring: "265 83% 58%",
    },
    dark: {
      background: "246 42% 8%",
      foreground: "248 80% 96%",
      card: "246 35% 12%",
      "card-foreground": "248 80% 96%",
      primary: "265 91% 70%",
      "primary-foreground": "246 42% 8%",
      border: "248 22% 24%",
      input: "248 22% 22%",
      ring: "265 91% 70%",
    },
  },
  appearance: {
    radius: "1.25rem",
    cardRadius: "1.5rem",
    panelRadius: "0.75rem",
    controlRadius: "0.375rem",
    density: "compact",
    surface: "command",
    background: "grid",
    navigation: "solid",
    cardPadding: "0.875rem",
    panelPadding: "1rem",
    fontFamily: "\"Fira Sans\", \"Segoe UI\", sans-serif",
    monoFontFamily: "\"Fira Code\", \"SFMono-Regular\", monospace",
    shadow: "0 18px 46px rgba(32, 18, 96, 0.22)",
  },
}

function renderSettings(path = "/settings?section=appearance") {
  window.history.replaceState({}, "", path)
  return render(
    <ThemeModeProvider>
      <AppThemeProvider>
        <Settings />
      </AppThemeProvider>
    </ThemeModeProvider>,
  )
}

describe("settings theme system", () => {
  beforeEach(() => {
    window.localStorage.clear()
    document.documentElement.className = ""
    document.documentElement.removeAttribute("data-theme")
    document.documentElement.removeAttribute("data-theme-source")
    document.documentElement.removeAttribute("data-theme-surface")
    document.documentElement.removeAttribute("data-theme-background")
    document.documentElement.removeAttribute("style")

    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    geoIPSettingsMock.mockReset()
    updateGeoIPSettingsMock.mockReset()
    updateGeoIPDatabaseMock.mockReset()
    dashboardSettingsMock.mockReset()
    updateDashboardSettingsMock.mockReset()
    themeSettingsMock.mockReset()
    updateThemeSettingsMock.mockReset()
    importThemeArchiveMock.mockReset()
    importThemeFromGitHubMock.mockReset()
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    versionMetaMock.mockReset()

    nodesMock.mockResolvedValue({ nodes: [] })
    agentReleaseMock.mockResolvedValue({ target_version: "abc", download_url: "https://example.com/agent", sha256: "sha", check_interval_seconds: 1800 })
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
    geoIPSettingsMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    updateGeoIPSettingsMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    updateGeoIPDatabaseMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    dashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: true, show_system_pressure: true, show_memory_pressure: true })
    updateDashboardSettingsMock.mockResolvedValue({ show_dashboard_card_ip: true, show_system_pressure: true, show_memory_pressure: true })
    themeSettingsMock.mockResolvedValue({ theme: "classic", custom_themes: [], configured: false })
    updateThemeSettingsMock.mockResolvedValue({ theme: "classic", custom_themes: [], configured: true })
    importThemeArchiveMock.mockResolvedValue({ theme: auroraThemePackage })
    importThemeFromGitHubMock.mockResolvedValue({ filename: "aurora-command.thism-theme.zip", data: "AQID" })
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
    updateNotificationSettingsMock.mockResolvedValue({})
    versionMetaMock.mockResolvedValue({ version: "1.0.0", commit: "abc", build_time: "2026-03-19T00:00:00Z" })
    window.scrollTo = vi.fn()
  })

  it("keeps appearance settings focused on theme state and import sources", async () => {
    renderSettings()

    expect(await screen.findByRole("tab", { name: "Appearance" })).toBeInTheDocument()
    expect(await screen.findByRole("heading", { name: "Theme System", level: 3 })).toBeInTheDocument()
    expect(await screen.findByText("Current theme")).toBeInTheDocument()
    expect(screen.getByRole("list", { name: "Theme list" })).toBeInTheDocument()
    expect(screen.getByLabelText("GitHub theme repository")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Import release archive" })).toBeInTheDocument()
    expect(screen.getByLabelText("Upload theme zip")).toHaveAttribute("accept", expect.stringContaining(".zip"))
    expect(screen.queryByText("Frontend Skins")).not.toBeInTheDocument()
    expect(screen.queryByText("Theme Plugin management")).not.toBeInTheDocument()
    expect(screen.queryByRole("button", { name: "Use theme" })).not.toBeInTheDocument()
  })

  it("imports and applies an uploaded theme zip", async () => {
    const user = userEvent.setup()
    renderSettings()

    const file = new File(["theme archive"], "aurora-command.thism-theme.zip", { type: "application/zip" })
    await user.upload(await screen.findByLabelText("Upload theme zip"), file)

    await waitFor(() => expect(importThemeArchiveMock).toHaveBeenCalledWith("aurora-command.thism-theme.zip", expect.any(String)))
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("custom:aurora-command"))
    expect(screen.getAllByText("Aurora Command").length).toBeGreaterThanOrEqual(1)
    expect(await screen.findByText("Imported and applied Aurora Command.")).toBeInTheDocument()
  })

  it("imports the latest GitHub release zip through the server proxy", async () => {
    const user = userEvent.setup()
    renderSettings()
    await user.type(await screen.findByLabelText("GitHub theme repository"), "https://github.com/acme/thism-themes")
    await user.click(screen.getByRole("button", { name: "Import release archive" }))

    await waitFor(() => expect(importThemeFromGitHubMock).toHaveBeenCalledWith("https://github.com/acme/thism-themes"))
    await waitFor(() => expect(document.documentElement.dataset.theme).toBe("custom:aurora-command"))
    expect(await screen.findByText("Imported and applied Aurora Command.")).toBeInTheDocument()
  })

  it("reports invalid theme archives without changing the current theme", async () => {
    const user = userEvent.setup()
    importThemeArchiveMock.mockRejectedValueOnce(new Error("Invalid theme archive"))
    renderSettings()

    const file = new File(["invalid archive"], "aurora-command.thism-theme.zip", { type: "application/zip" })
    await user.upload(await screen.findByLabelText("Upload theme zip"), file)

    expect(await screen.findByRole("alert")).toHaveTextContent("Invalid theme archive")
    expect(importThemeArchiveMock).toHaveBeenCalled()
    expect(document.documentElement.dataset.theme).toBe("classic")
  })
})
