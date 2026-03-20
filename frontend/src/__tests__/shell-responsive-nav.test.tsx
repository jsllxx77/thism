import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { ThemeModeProvider } from "../theme/mode"
import { LanguageProvider } from "../i18n/language"

const sessionMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    session: (...args: unknown[]) => sessionMock(...args),
  },
}))

vi.mock("../pages/Dashboard", () => ({
  Dashboard: () => <div>Dashboard page</div>,
}))

vi.mock("../pages/Settings", () => ({
  Settings: () => <div>Settings page</div>,
}))

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <div>{nodeId}</div>,
}))

describe("responsive shell nav", () => {
  beforeEach(() => {
    sessionMock.mockReset()
    sessionMock.mockResolvedValue({ role: "admin" })
    window.localStorage.clear()
    document.cookie = "thism-lang=; Path=/; Max-Age=0; SameSite=Lax"
  })

  function renderApp(route = "/") {
    render(
      <LanguageProvider>
        <ThemeModeProvider>
          <AppThemeProvider>
            <MemoryRouter initialEntries={[route]}>
              <App />
            </MemoryRouter>
          </AppThemeProvider>
        </ThemeModeProvider>
      </LanguageProvider>
    )
  }

  it("renders top-toolbar actions without sidebar navigation", async () => {
    renderApp("/")

    expect(screen.queryByRole("navigation", { name: "Primary Navigation" })).not.toBeInTheDocument()
    expect(await screen.findByText("ThisM Console")).toBeInTheDocument()
    const languageButton = await screen.findByRole("button", { name: "中文" })
    expect(languageButton).toBeInTheDocument()
    expect(languageButton.className).toContain("text-slate-900")
    expect(languageButton.className).toContain("hover:text-slate-950")
    expect(languageButton.className).toContain("dark:text-slate-100")
    expect(languageButton.className).toContain("dark:hover:text-slate-50")
    expect(await screen.findByRole("button", { name: "Refresh data" })).toBeInTheDocument()
    expect(await screen.findByRole("button", { name: "Toggle dark mode" })).toBeInTheDocument()
    expect(await screen.findByRole("button", { name: "Open settings" })).toBeInTheDocument()
  })

  it("shows a guest mode badge linking back to login", async () => {
    sessionMock.mockResolvedValue({ role: "guest" })
    renderApp("/")

    const guestLink = await screen.findByRole("link", { name: "Return to login" })
    expect(guestLink).toHaveAttribute("href", "/login")
    expect(guestLink).toHaveTextContent("Guest mode")
    expect(screen.queryByRole("button", { name: "Open settings" })).not.toBeInTheDocument()
  })

  it("toggles header labels to Chinese", async () => {
    const user = userEvent.setup()
    renderApp("/")

    await user.click(await screen.findByRole("button", { name: "中文" }))

    expect(await screen.findByRole("button", { name: "刷新数据" })).toBeInTheDocument()
    expect(await screen.findByRole("button", { name: "切换深色模式" })).toBeInTheDocument()
    expect(await screen.findByRole("button", { name: "打开设置" })).toBeInTheDocument()
  })

  it("navigates to settings from the header shortcut", async () => {
    const user = userEvent.setup()
    renderApp("/")

    await user.click(await screen.findByRole("button", { name: "Open settings" }))
    expect(await screen.findByText("Settings page")).toBeInTheDocument()
  })
})
