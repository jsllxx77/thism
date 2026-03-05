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

    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
      expect(screen.getByText("beta")).toBeInTheDocument()
    })

    await user.selectOptions(screen.getByLabelText("Status filter"), "online")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.queryByText("beta")).not.toBeInTheDocument()

    await user.clear(screen.getByLabelText("Search nodes"))
    await user.type(screen.getByLabelText("Search nodes"), "alp")
    expect(screen.getByText("alpha")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Table" }))
    expect(screen.getByText("Node table view")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Reset filters" }))
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getByText("beta")).toBeInTheDocument()
  })
})
