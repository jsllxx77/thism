import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../../lib/api"
import { NodeHero } from "./NodeHero"
import { MetricTabs } from "./MetricTabs"

const node: Node = {
  id: "node-1",
  name: "alpha",
  ip: "10.0.0.7",
  os: "linux",
  arch: "amd64",
  created_at: 0,
  last_seen: 1733011200,
  online: true,
}

describe("node detail metrics", () => {
  it("renders hero with node identity", () => {
    render(<NodeHero node={node} />)

    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getByText("Online")).toBeInTheDocument()
    expect(screen.getByText(/10.0.0.7/)).toBeInTheDocument()
  })

  it("switches metric tabs and range", async () => {
    const user = userEvent.setup()
    const onRangeChange = vi.fn()

    const points = [
      { ts: 1700000000, value: 30 },
      { ts: 1700000300, value: 44 },
    ]

    render(
      <MetricTabs
        range={3600}
        onRangeChange={onRangeChange}
        cpuData={points}
        memData={points}
        netRxData={points}
        netTxData={points}
        diskData={points}
      />
    )

    expect(screen.getByRole("button", { name: "CPU" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Network" }))
    expect(screen.getByText("Network RX")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "6h" }))
    expect(onRangeChange).toHaveBeenCalledWith(21600)
  })
})
