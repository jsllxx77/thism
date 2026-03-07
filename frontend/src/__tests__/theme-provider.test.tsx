import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
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

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <div>{nodeId}</div>,
}))

function Probe() {
  return <span data-testid="theme-probe">theme-probe</span>
}

describe("theme provider", () => {
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

  it("applies app surface class at shell root", () => {
    render(
      <ThemeModeProvider>
        <MemoryRouter initialEntries={["/"]}>
          <App />
        </MemoryRouter>
      </ThemeModeProvider>,
    )

    expect(document.querySelector(".app-surface-bg")).toBeInTheDocument()
  })
})
