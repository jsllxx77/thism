import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { Dashboard } from "../pages/Dashboard"

vi.mock("../lib/api", () => ({
  api: {
    nodes: vi.fn().mockResolvedValue({
      nodes: [
        { id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", last_seen: 0 },
      ],
    }),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: () => {},
    off: () => {},
  }),
}))

describe("motion smoke", () => {
  it("renders dashboard sections with motion wrappers", async () => {
    render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    expect(screen.getByTestId("motion-dashboard-root")).toBeInTheDocument()
    expect(screen.getByTestId("motion-dashboard-content")).toBeInTheDocument()
  })
})
