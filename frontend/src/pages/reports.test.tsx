import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { api } from "../lib/api"
import { Reports } from "./Reports"

vi.mock("../lib/api", () => ({
  api: {
    availabilityReport: vi.fn(),
  },
}))

function report(overrides = {}) {
  return {
    range: { from: 100, to: 200 },
    filter: {},
    available_tags: ["dev", "prod"],
    overview: {
      total_nodes: 1,
      average_availability_percent: 99.5,
      nodes_below_99: 0,
      total_offline_duration_seconds: 60,
      highest_latency_p95_ms: 42.5,
    },
    nodes: [
      {
        node_id: "node-a",
        name: "alpha",
        tags: ["prod", "hk"],
        last_seen: Math.floor(Date.now() / 1000),
        availability_percent: 99.5,
        expected_samples: 20,
        observed_samples: 19,
        offline_duration_seconds: 60,
        outage_count: 1,
        latency_p95_ms: 42.5,
      },
    ],
    ...overrides,
  }
}

describe("reports page", () => {
  it("loads availability data and refetches when filters change", async () => {
    const user = userEvent.setup()
    const availabilityReportMock = vi.mocked(api.availabilityReport)
    availabilityReportMock.mockResolvedValue(report())

    render(<Reports />)

    expect(await screen.findByRole("heading", { name: "Reports" })).toBeInTheDocument()
    expect((await screen.findAllByText("99.50%")).length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.getAllByText("prod").length).toBeGreaterThanOrEqual(2)
    expect(screen.getAllByText("42.5 ms").length).toBeGreaterThanOrEqual(2)

    await user.selectOptions(screen.getByLabelText("Tag filter"), "prod")

    await waitFor(() => {
      expect(availabilityReportMock).toHaveBeenLastCalledWith(expect.any(Number), expect.any(Number), "prod")
    })
  })

  it("shows an empty state when the report has no rows", async () => {
    vi.mocked(api.availabilityReport).mockResolvedValue(report({ nodes: [], overview: { ...report().overview, total_nodes: 0 } }))

    render(<Reports />)

    expect(await screen.findByText("No nodes match this report filter.")).toBeInTheDocument()
  })

  it("renders nodes when an older report payload has null tags", async () => {
    vi.mocked(api.availabilityReport).mockResolvedValue(report({
      available_tags: [],
      nodes: [
        {
          node_id: "node-a",
          name: "alpha",
          tags: null,
          last_seen: Math.floor(Date.now() / 1000),
          availability_percent: 99.5,
          expected_samples: 20,
          observed_samples: 19,
          offline_duration_seconds: 60,
          outage_count: 1,
          latency_p95_ms: 42.5,
        },
      ],
    }))

    render(<Reports />)

    expect(await screen.findByText("alpha")).toBeInTheDocument()
    expect(screen.getByText("—")).toBeInTheDocument()
  })
})
