import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { ThemeModeProvider } from "../theme/mode"
import { LanguageProvider } from "../i18n/language"

const sessionMock = vi.fn()
const versionMetaMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    session: (...args: unknown[]) => sessionMock(...args),
    versionMeta: (...args: unknown[]) => versionMetaMock(...args),
  },
}))

vi.mock("../pages/Dashboard", () => ({
  Dashboard: () => <div>Dashboard page</div>,
}))

vi.mock("../pages/Settings", () => ({
  Settings: () => <div>Settings page</div>,
}))

vi.mock("../pages/Reports", () => ({
  Reports: () => <div>Reports page</div>,
}))

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <div>{nodeId}</div>,
}))

function renderApp(path = "/") {
  window.history.replaceState({}, "", path)
  return render(
    <LanguageProvider>
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={[path]}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>
    </LanguageProvider>,
  )
}

describe("shell footer version and GitHub link", () => {
  beforeEach(() => {
    sessionMock.mockReset()
    sessionMock.mockResolvedValue({ role: "admin" })
    versionMetaMock.mockReset()
    versionMetaMock.mockResolvedValue({ version: "v0.6.33", commit: "abc1234", build_time: "2026-03-18T04:00:00Z" })
    window.localStorage.clear()
    document.cookie = "thism-lang=; Path=/; Max-Age=0; SameSite=Lax"
  })

  it("renders the server version in the footer", async () => {
    renderApp("/")

    expect(await screen.findByText(/v0\.6\.33/)).toBeInTheDocument()
    expect(screen.getByText(/^ThisM server v0\.6\.33$/)).toBeInTheDocument()
    expect(versionMetaMock).toHaveBeenCalledTimes(1)
  })

  it("links to the GitHub repository with an accessible label", async () => {
    renderApp("/")

    const link = await screen.findByRole("link", { name: "View source code on GitHub" })
    expect(link).toHaveAttribute("href", "https://github.com/jsllxx77/thism")
    expect(link).toHaveAttribute("target", "_blank")
    expect(link).toHaveAttribute("rel", "noreferrer")
  })

  it("keeps the footer visible when version metadata is unavailable", async () => {
    versionMetaMock.mockRejectedValue(new Error("boom"))
    renderApp("/")

    expect(await screen.findByText("ThisM server")).toBeInTheDocument()
    expect(await screen.findByRole("link", { name: "View source code on GitHub" })).toBeInTheDocument()
  })
})
