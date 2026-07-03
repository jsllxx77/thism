import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { useAppTheme } from "../theme/theme-context"
import { ThemeModeProvider } from "../theme/mode"

const sessionMock = vi.fn().mockResolvedValue({ role: "admin" })
const themeSettingsMock = vi.fn()
const updateThemeSettingsMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    session: (...args: unknown[]) => sessionMock(...args),
    themeSettings: (...args: unknown[]) => themeSettingsMock(...args),
    updateThemeSettings: (...args: unknown[]) => updateThemeSettingsMock(...args),
  },
}))

vi.mock("../pages/Dashboard", () => ({
  Dashboard: () => <div>Dashboard</div>,
}))

vi.mock("../pages/Settings", () => ({
  Settings: () => <div>Settings</div>,
}))

vi.mock("../pages/Reports", () => ({
  Reports: () => <div>Reports</div>,
}))

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <div>{nodeId}</div>,
}))

function Probe() {
  return <span data-testid="theme-probe">theme-probe</span>
}

function ThemeProbe() {
  const { theme, setTheme } = useAppTheme()

  return (
    <>
      <span data-testid="theme-name">{theme}</span>
      <button type="button" onClick={() => setTheme("ocean")}>
        Set ocean theme
      </button>
    </>
  )
}

const themeTokens = {
  background: "248 80% 98%",
  foreground: "246 38% 12%",
  card: "0 0% 100%",
  "card-foreground": "246 38% 12%",
  primary: "265 83% 58%",
  "primary-foreground": "0 0% 100%",
  border: "252 30% 84%",
  input: "252 30% 82%",
  ring: "265 83% 58%",
}

const serverThemePackage = {
  type: "thism-theme",
  version: 1,
  id: "server-command",
  name: "Server Command",
  accent: "#8b5cf6",
  tokens: {
    light: themeTokens,
    dark: {
      ...themeTokens,
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
  appearance: {},
}

describe("theme provider", () => {
  beforeEach(() => {
    themeSettingsMock.mockReset()
    updateThemeSettingsMock.mockReset()
    themeSettingsMock.mockResolvedValue({ theme: "classic", custom_themes: [], configured: false })
    updateThemeSettingsMock.mockResolvedValue({ theme: "classic", custom_themes: [], configured: true })
    localStorage.clear()
    document.documentElement.removeAttribute("data-theme")
  })

  it("renders children unchanged", () => {
    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <Probe />
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    expect(screen.getByTestId("theme-probe")).toHaveTextContent("theme-probe")
  })

  it("applies and persists the selected runtime theme", async () => {
    const user = userEvent.setup()

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <ThemeProbe />
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    expect(screen.getByTestId("theme-name")).toHaveTextContent("classic")
    expect(document.documentElement.dataset.theme).toBe("classic")

    await user.click(screen.getByRole("button", { name: "Set ocean theme" }))

    expect(screen.getByTestId("theme-name")).toHaveTextContent("ocean")
    expect(document.documentElement.dataset.theme).toBe("ocean")
    expect(localStorage.getItem("thism-color-theme")).toBe("ocean")
  })

  it("applies app surface class at shell root", () => {
    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={["/"]}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    expect(document.querySelector(".app-surface-bg")).toBeInTheDocument()
  })

  it("loads configured custom themes from the server", async () => {
    themeSettingsMock.mockResolvedValue({
      theme: "custom:server-command",
      custom_themes: [serverThemePackage],
      configured: true,
    })

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <ThemeProbe />
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("theme-name")).toHaveTextContent("custom:server-command")
    })
    expect(document.documentElement.dataset.theme).toBe("custom:server-command")
    expect(localStorage.getItem("thism-custom-themes")).toContain("Server Command")
  })

  it("migrates a local custom theme to the server when the server has no theme settings", async () => {
    localStorage.setItem("thism-color-theme", "custom:server-command")
    localStorage.setItem("thism-custom-themes", JSON.stringify([serverThemePackage]))

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <ThemeProbe />
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId("theme-name")).toHaveTextContent("custom:server-command")
    })
    await waitFor(() => {
      expect(updateThemeSettingsMock).toHaveBeenCalledWith({
        theme: "custom:server-command",
        custom_themes: [serverThemePackage],
      })
    })
  })
})
