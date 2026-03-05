import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { theme } from "antd"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"

vi.mock("../pages/Dashboard", () => ({
  Dashboard: () => <div>Dashboard</div>,
}))

vi.mock("../pages/Settings", () => ({
  Settings: () => <div>Settings</div>,
}))

vi.mock("../pages/NodeDetail", () => ({
  NodeDetail: ({ nodeId }: { nodeId: string }) => <div>{nodeId}</div>,
}))

function TokenProbe() {
  const { token } = theme.useToken()
  return <span data-testid="color-primary">{token.colorPrimary}</span>
}

describe("theme provider", () => {
  it("provides custom antd token values", () => {
    render(
      <AppThemeProvider>
        <TokenProbe />
      </AppThemeProvider>
    )

    expect(screen.getByTestId("color-primary")).toHaveTextContent("#34d399")
  })

  it("applies app gradient class at shell root", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>
    )

    expect(document.querySelector(".app-gradient-bg")).toBeInTheDocument()
  })
})
