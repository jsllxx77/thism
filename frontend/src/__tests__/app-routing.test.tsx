import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { ThemeModeProvider } from "../theme/mode"

const sessionMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    session: (...args: unknown[]) => sessionMock(...args),
  },
}))

vi.mock("../pages/Dashboard", () => ({
  Dashboard: ({ onSelectNode }: { onSelectNode: (id: string) => void }) => (
    <div>
      <h1>Dashboard</h1>
      <p>Nodes</p>
      <button onClick={() => onSelectNode("node-1")}>Open Node</button>
    </div>
  ),
}))

vi.mock("../pages/Settings", () => ({
  Settings: () => <h1>Settings</h1>,
}))

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <h1>Node {nodeId}</h1>,
}))

vi.mock("../pages/NotFound", () => ({
  NotFound: () => (
    <div>
      <h1>Nothing here</h1>
      <button type="button">Back to dashboard</button>
    </div>
  ),
}))

describe("app routing", () => {
  beforeEach(() => {
    sessionMock.mockReset()
  })

  function renderApp(entries: string[], index?: number) {
    sessionMock.mockResolvedValue({ role: "admin" })

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={entries} initialIndex={index}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>
    )
  }

  it("renders dashboard on root route", async () => {
    renderApp(["/"])

    expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeInTheDocument()
  })

  it("redirects guest users away from settings", async () => {
    sessionMock.mockResolvedValue({ role: "guest" })

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={["/settings"]}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>
    )

    expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeInTheDocument()
    expect(screen.queryByRole("heading", { name: "Settings" })).not.toBeInTheDocument()
  })

  it("navigates between settings and dashboard using history", async () => {
    const user = userEvent.setup()
    renderApp(["/", "/settings"], 1)

    expect(await screen.findByRole("heading", { name: "Settings" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /back to dashboard/i }))
    expect(await screen.findByRole("heading", { name: "Dashboard" })).toBeInTheDocument()
  })

  it("renders a not found page for unknown routes", async () => {
    renderApp(["/does-not-exist"])

    expect(await screen.findByRole("heading", { name: "Nothing here" })).toBeInTheDocument()
    expect(await screen.findByRole("button", { name: "Back to dashboard" })).toBeInTheDocument()
  })
})
