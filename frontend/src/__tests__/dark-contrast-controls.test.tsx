import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { NodeFilters } from "../components/dashboard/NodeFilters"
import { ViewModeToggle } from "../components/dashboard/ViewModeToggle"

describe("dark contrast for dashboard controls", () => {
  it("applies dark classes to filter controls", () => {
    const { container } = render(
      <NodeFilters
        status="all"
        search=""
        onStatusChange={vi.fn()}
        onSearchChange={vi.fn()}
        onReset={vi.fn()}
      />
    )

    const statusGroup = screen.getByRole("group", { name: "Status filter" })
    const activeStatusButton = screen.getByRole("button", { name: "All" })
    const inactiveStatusButton = screen.getByRole("button", { name: "Online" })
    const searchInput = screen.getByLabelText("Search nodes")
    const resetButton = screen.getByRole("button", { name: "Reset filters" })
    const filterShell = container.firstElementChild as HTMLElement | null

    expect(filterShell?.className).toContain("enterprise-surface")
    expect(filterShell?.className).not.toContain("bg-[linear-gradient(135deg")
    expect(statusGroup.className).toContain("enterprise-inner-surface")
    expect(statusGroup.className).toContain("shadow-none")
    expect(activeStatusButton.className).toContain("dark:bg-slate-900")
    expect(activeStatusButton.className).toContain("dark:text-slate-50")
    expect(inactiveStatusButton.className).toContain("dark:text-slate-200")
    expect(searchInput.className).toContain("dark:bg-slate-950")
    expect(searchInput.className).toContain("dark:text-slate-100")
    expect(resetButton.className).toContain("dark:text-slate-200")
  })

  it("keeps inactive view toggle button readable in dark mode", () => {
    render(<ViewModeToggle mode="cards" onChange={vi.fn()} />)

    const inactiveButton = screen.getByRole("button", { name: "Table View" })
    expect(inactiveButton.className).toContain("dark:text-slate-200")
    expect(inactiveButton.className).toContain("dark:hover:bg-slate-900")
  })
})
