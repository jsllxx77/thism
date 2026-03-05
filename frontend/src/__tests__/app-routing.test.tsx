import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"

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

describe("app routing", () => {
  it("renders dashboard on root route", () => {
    render(
      <MemoryRouter initialEntries={["/"]}>
        <App />
      </MemoryRouter>
    )

    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument()
  })

  it("navigates between settings and dashboard using history", async () => {
    const user = userEvent.setup()

    render(
      <MemoryRouter initialEntries={["/", "/settings"]} initialIndex={1}>
        <App />
      </MemoryRouter>
    )

    expect(screen.getByRole("heading", { name: "Settings" })).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /back to dashboard/i }))
    expect(screen.getByRole("heading", { name: "Dashboard" })).toBeInTheDocument()
  })
})
