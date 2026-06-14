import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { useAppTheme } from "../theme/theme-context"
import { ThemeModeProvider } from "../theme/mode"

const sessionMock = vi.fn().mockResolvedValue({ role: "admin" })

vi.mock("../lib/api", () => ({
  api: {
    session: (...args: unknown[]) => sessionMock(...args),
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

describe("theme provider", () => {
  beforeEach(() => {
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
})
