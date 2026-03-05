import { describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import { OverviewStats } from "./OverviewStats"

describe("overview stats", () => {
  it("renders all summary metrics", () => {
    render(
      <OverviewStats
        onlineNodes={3}
        totalNodes={5}
        avgCpu={48.5}
        avgMemory={62.2}
        alertCount={2}
        heartbeatLatencyMs={134}
      />
    )

    expect(screen.getByText("Online Nodes")).toBeInTheDocument()
    expect(screen.getByText("Avg CPU")).toBeInTheDocument()
    expect(screen.getByText("Avg Memory")).toBeInTheDocument()
    expect(screen.getByText("Alerts")).toBeInTheDocument()
    expect(screen.getByText("Heartbeat")).toBeInTheDocument()
    expect(screen.getByText("3 / 5")).toBeInTheDocument()
    expect(screen.getByText("48.5%")).toBeInTheDocument()
    expect(screen.getByText("62.2%")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
    expect(screen.getByText("134 ms")).toBeInTheDocument()
  })
})
