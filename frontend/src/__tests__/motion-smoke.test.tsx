import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import { Dashboard } from "../pages/Dashboard"
import { NodeCard } from "../components/NodeCard"
import { CollapsibleContent } from "../components/node-detail/CollapsibleContent"

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
    const { container } = render(<Dashboard onSelectNode={() => {}} />)

    await waitFor(() => {
      expect(screen.getByText("alpha")).toBeInTheDocument()
    })

    expect(screen.getByTestId("motion-dashboard-root")).toBeInTheDocument()
    expect(screen.getByTestId("motion-dashboard-content")).toBeInTheDocument()
    expect(container.querySelector(".motion-card-grid")).toBeInTheDocument()
  })

  it("keeps key component motion classes mounted", () => {
    const { container, rerender } = render(
      <>
        <NodeCard
          node={{ id: "n1", name: "alpha", online: true, ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 0, last_seen: 0 }}
          cpu={42}
          memUsed={512}
          memTotal={1024}
        />
        <CollapsibleContent open>
          <div>expanded content</div>
        </CollapsibleContent>
      </>
    )

    expect(container.querySelector(".node-card-shell")).toBeInTheDocument()
    expect(container.querySelector(".metric-value")).toBeInTheDocument()
    expect(container.querySelector(".motion-collapsible-content")).toBeInTheDocument()

    rerender(<CollapsibleContent open={false}>collapsed content</CollapsibleContent>)
    expect(screen.queryByText("collapsed content")).not.toBeInTheDocument()
  })
})
