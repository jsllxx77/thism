import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Dashboard } from "../../pages/Dashboard"

vi.mock("../../lib/api", () => ({
  api: {
    nodes: vi.fn().mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 0 },
        { id: "n2", name: "beta", online: false, ip: "1.1.1.2", os: "linux", arch: "arm64", last_seen: 0 },
      ],
    }),
  },
}))

vi.mock("../../lib/ws", () => ({
  getDashboardWS: () => ({
    on: () => {},
    off: () => {},
  }),
}))

describe("dashboard filtering", () => {
  it("filters by status, searches, toggles view mode, and resets", async () => {
    const user = userEvent.setup()

    const { container } = render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
      expect(screen.getByText("beta")).toBeInTheDocument()
    })
    expect(screen.getByText("Nodes")).toBeInTheDocument()

    expect(screen.getByLabelText("Status filter")).toHaveClass("h-11")
    expect(screen.getByLabelText("Search nodes")).toHaveClass("h-11")
    expect(screen.getByRole("button", { name: "Reset filters" })).toHaveClass("h-11")

    const filterShell = container.querySelector("section.panel-card") as HTMLElement | null
    expect(filterShell?.className).toContain("enterprise-surface")
    expect(screen.getByLabelText("Status filter").className).toContain("shadow-none")
    expect(screen.getByLabelText("Search nodes").className).toContain("shadow-none")

    const toggleShell = screen.getByRole("button", { name: "Cards View" }).parentElement as HTMLElement | null
    expect(toggleShell?.className).toContain("enterprise-inner-surface")
    expect(screen.getByRole("button", { name: "Cards View" }).className).toContain("bg-slate-50/90")
    expect(screen.getByRole("button", { name: "Cards View" }).className).toContain("dark:border-white/10")
    expect(screen.getByRole("button", { name: "Cards View" }).className).toContain("dark:ring-white/10")

    await user.selectOptions(screen.getByLabelText("Status filter"), "online")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.queryByText("beta")).not.toBeInTheDocument()

    await user.clear(screen.getByLabelText("Search nodes"))
    await user.type(screen.getByLabelText("Search nodes"), "alp")
    expect(screen.getByText("alpha")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Table View" }))
    expect(screen.getByRole("button", { name: "Table View" }).className).toContain("bg-slate-50/90")
    expect(screen.getByRole("button", { name: "Table View" }).className).toContain("dark:border-white/10")
    expect(await screen.findByText("Node Inventory")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Reset filters" }))
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getByText("beta")).toBeInTheDocument()
  })

  it("shows an empty state when no node matches active filters", async () => {
    const user = userEvent.setup()

    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
      expect(screen.getByText("beta")).toBeInTheDocument()
    })

    await user.type(screen.getByLabelText("Search nodes"), "zzz")

    expect(screen.getByText("No nodes match these filters")).toBeInTheDocument()
  })
})
