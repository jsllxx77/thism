import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import App from "../App"

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
  it("renders desktop navigation items and mobile trigger", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>
    )

    expect(screen.getByRole("navigation", { name: "Primary Navigation" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Dashboard" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Settings" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Open navigation" })).toBeInTheDocument()
  })

  it("marks active route in navigation", () => {
    render(
      <MemoryRouter initialEntries={["/settings"]}>
        <App />
      </MemoryRouter>
    )

    expect(screen.getByRole("button", { name: "Settings" })).toHaveAttribute("aria-current", "page")
  })
})
