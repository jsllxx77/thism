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
const dashboardSettingsMock = vi.fn()
const updateDashboardSettingsMock = vi.fn()
const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()
const versionMetaMock = vi.fn()
const frontendSkinsMock = vi.fn()
const installFrontendSkinFromGitHubMock = vi.fn()
const installFrontendSkinArchiveMock = vi.fn()
const selectFrontendSkinMock = vi.fn()
const deleteFrontendSkinMock = vi.fn()

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
    frontendSkins: (...args: unknown[]) => frontendSkinsMock(...args),
    installFrontendSkinFromGitHub: (...args: unknown[]) => installFrontendSkinFromGitHubMock(...args),
    installFrontendSkinArchive: (...args: unknown[]) => installFrontendSkinArchiveMock(...args),
    selectFrontendSkin: (...args: unknown[]) => selectFrontendSkinMock(...args),
    deleteFrontendSkin: (...args: unknown[]) => deleteFrontendSkinMock(...args),
  },
}))

const auroraThemePackage = {
  type: "thism-theme",
  version: 1,
  id: "aurora-command",
  name: "Aurora Command",
  description: "A high-contrast operations theme with compact command surfaces.",
  accent: "#8b5cf6",
  tokens: {
    light: {
      background: "248 80% 98%",
      foreground: "246 38% 12%",
      card: "0 0% 100%",
      "card-foreground": "246 38% 12%",
      popover: "0 0% 100%",
      "popover-foreground": "246 38% 12%",
      primary: "265 83% 58%",
      "primary-foreground": "0 0% 100%",
      secondary: "252 62% 94%",
      "secondary-foreground": "246 38% 14%",
      muted: "252 44% 92%",
      "muted-foreground": "248 13% 42%",
      accent: "176 66% 43%",
      "accent-foreground": "246 38% 12%",
      destructive: "0 78% 56%",
      "destructive-foreground": "0 0% 100%",
      border: "252 30% 84%",
      input: "252 30% 82%",
      ring: "265 83% 58%",
    },
    dark: {
      background: "246 42% 8%",
      foreground: "248 80% 96%",
      card: "246 35% 12%",
      "card-foreground": "248 80% 96%",
      popover: "246 35% 12%",
      "popover-foreground": "248 80% 96%",
      primary: "265 91% 70%",
      "primary-foreground": "246 42% 8%",
      secondary: "248 28% 18%",
      "secondary-foreground": "248 80% 96%",
      muted: "248 24% 16%",
      "muted-foreground": "250 22% 72%",
      accent: "176 70% 52%",
      "accent-foreground": "246 42% 8%",
      destructive: "0 66% 52%",
      "destructive-foreground": "0 0% 100%",
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
    dashboardSettingsMock.mockReset()
    updateDashboardSettingsMock.mockReset()
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    versionMetaMock.mockReset()
    frontendSkinsMock.mockReset()
    installFrontendSkinFromGitHubMock.mockReset()
    installFrontendSkinArchiveMock.mockReset()
    selectFrontendSkinMock.mockReset()
    deleteFrontendSkinMock.mockReset()

    nodesMock.mockResolvedValue({ nodes: [] })
    agentReleaseMock.mockResolvedValue({ target_version: "abc", download_url: "https://example.com/agent", sha256: "sha", check_interval_seconds: 1800 })
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
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
    frontendSkinsMock.mockResolvedValue({
      active_skin_id: "classic",
      skins: [
        {
          id: "classic",
          name: "Classic",
          description: "Bundled thisM React frontend.",
          source: "built-in",
          entry: "index.html",
          api_version: "thism.v1",
        },
      ],
    })
    installFrontendSkinFromGitHubMock.mockResolvedValue({
      active_skin_id: "shadcn-dashboard",
      skin: {
        id: "shadcn-dashboard",
        name: "Shadcn Dashboard",
        description: "Blocks-style dashboard skin.",
        source: "custom",
        entry: "index.html",
        api_version: "thism.v1",
      },
    })
    installFrontendSkinArchiveMock.mockResolvedValue({
      active_skin_id: "shadcn-dashboard",
      skin: {
        id: "shadcn-dashboard",
        name: "Shadcn Dashboard",
        source: "custom",
        entry: "index.html",
        api_version: "thism.v1",
      },
    })
    selectFrontendSkinMock.mockResolvedValue({ active_skin_id: "shadcn-dashboard" })
    deleteFrontendSkinMock.mockResolvedValue({ ok: true, active_skin_id: "classic" })
    window.scrollTo = vi.fn()
  })

  it("adds an appearance settings section for runtime themes", async () => {
    renderSettings()

    expect(await screen.findByRole("tab", { name: "Appearance" })).toBeInTheDocument()
    expect(await screen.findByRole("heading", { name: "Theme System", level: 3 })).toBeInTheDocument()
    expect(screen.getByRole("combobox", { name: "Active theme" })).toBeInTheDocument()
    expect(screen.getByLabelText("Upload theme file")).toBeInTheDocument()
    expect(await screen.findByRole("heading", { name: "Frontend Skins", level: 3 })).toBeInTheDocument()
    expect(screen.getAllByText("Classic").length).toBeGreaterThanOrEqual(1)
    expect(screen.getByLabelText("Upload skin zip")).toBeInTheDocument()
  })

  it("imports a theme file, applies it, and persists it for runtime switching", async () => {
    const user = userEvent.setup()
    renderSettings()

    const file = new File([JSON.stringify(auroraThemePackage)], "aurora-command.thism-theme.json", {
      type: "application/json",
    })

    await user.upload(await screen.findByLabelText("Upload theme file"), file)

    await waitFor(() => {
      expect(screen.getAllByText("Aurora Command").length).toBeGreaterThanOrEqual(1)
    })
    await waitFor(() => {
      expect(document.documentElement.dataset.theme).toBe("custom:aurora-command")
      expect(document.documentElement.dataset.themeSource).toBe("custom")
    })
    expect(document.documentElement.style.getPropertyValue("--primary")).toBe("265 83% 58%")
    expect(document.documentElement.style.getPropertyValue("--theme-card-radius")).toBe("1.5rem")
    expect(window.localStorage.getItem("thism-color-theme")).toBe("custom:aurora-command")
    expect(window.localStorage.getItem("thism-custom-themes")).toContain("Aurora Command")
  })

  it("imports and applies a theme package from a GitHub repository URL", async () => {
    const fetchMock = vi.spyOn(window, "fetch").mockImplementation(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url === "https://api.github.com/repos/acme/thism-themes/releases/latest") {
        return new Response(JSON.stringify({
          assets: [
            {
              name: "aurora-command.thism-theme.json",
              url: "https://api.github.com/repos/acme/thism-themes/releases/assets/1",
              browser_download_url: "https://github.com/acme/thism-themes/releases/download/v1/aurora-command.thism-theme.json",
            },
          ],
        }))
      }
      if (url === "https://api.github.com/repos/acme/thism-themes/releases/assets/1") {
        return new Response(JSON.stringify(auroraThemePackage))
      }
      return new Response("not found", { status: 404 })
    })

    const user = userEvent.setup()
    renderSettings()

    await user.type(await screen.findByLabelText("GitHub theme repository"), "https://github.com/acme/thism-themes")
    await user.click(screen.getByRole("button", { name: "Install from GitHub" }))

    await waitFor(() => {
      expect(document.documentElement.dataset.theme).toBe("custom:aurora-command")
    })
    expect(window.localStorage.getItem("thism-custom-themes")).toContain("Aurora Command")
    expect(await screen.findByText("Imported and applied Aurora Command from GitHub.")).toBeInTheDocument()

    fetchMock.mockRestore()
  })

  it("reports invalid uploaded theme files without switching away from classic", async () => {
    const user = userEvent.setup()
    renderSettings()

    const file = new File([JSON.stringify({ type: "other-theme", name: "Nope" })], "bad-theme.json", {
      type: "application/json",
    })

    await user.upload(await screen.findByLabelText("Upload theme file"), file)

    expect(await screen.findByRole("alert")).toHaveTextContent("Invalid theme file")
    expect(document.documentElement.dataset.theme).toBe("classic")
    expect(window.localStorage.getItem("thism-color-theme")).toBe("classic")
  })

  it("installs and selects a frontend skin from a GitHub repository URL", async () => {
    const user = userEvent.setup()
    renderSettings()

    await user.type(await screen.findByLabelText("GitHub skin repository"), "https://github.com/acme/thism-shadcn-skin")
    await user.click(screen.getByRole("button", { name: "Install skin" }))

    await waitFor(() => {
      expect(installFrontendSkinFromGitHubMock).toHaveBeenCalledWith("https://github.com/acme/thism-shadcn-skin")
    })
    expect(await screen.findByText("Installed and selected Shadcn Dashboard from GitHub.")).toBeInTheDocument()
  })

  it("uploads a frontend skin zip package", async () => {
    const user = userEvent.setup()
    renderSettings()

    const file = new File(["skin archive"], "shadcn-dashboard.thism-frontend-skin.zip", {
      type: "application/zip",
    })

    await user.upload(await screen.findByLabelText("Upload skin zip"), file)

    await waitFor(() => {
      expect(installFrontendSkinArchiveMock).toHaveBeenCalled()
    })
    expect(installFrontendSkinArchiveMock.mock.calls[0][0]).toBe("shadcn-dashboard.thism-frontend-skin.zip")
    expect(typeof installFrontendSkinArchiveMock.mock.calls[0][1]).toBe("string")
    expect(await screen.findByText("Installed and selected Shadcn Dashboard.")).toBeInTheDocument()
  })
})
