import { describe, expect, it, vi } from "vitest"
import { render, screen, waitFor, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { api } from "../lib/api"
import { Reports } from "./Reports"

type RechartsProps = {
  children?: React.ReactNode
  data?: Array<{ nodeID: string; name: string }>
}

vi.mock("../lib/api", () => ({
  api: {
    availabilityReport: vi.fn(),
  },
}))

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: RechartsProps) => <div data-testid="responsive-container">{children}</div>,
  BarChart: ({ children, data = [] }: RechartsProps) => (
    <svg data-testid="bar-chart">
      {data.map((row) => (
        <text key={row.nodeID} data-testid="availability-ranking-row">
          {row.name}
        </text>
      ))}
      {children}
    </svg>
  ),
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  Tooltip: () => <div data-testid="tooltip-proxy" />,
  Bar: ({ children }: RechartsProps) => <div data-testid="bar-series">{children}</div>,
  Cell: () => <div data-testid="bar-cell" />,
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
        last_outage_start: Math.floor(Date.now() / 1000) - 120,
        last_outage_end: Math.floor(Date.now() / 1000) - 60,
        latency_p95_ms: 42.5,
      },
    ],
    ...overrides,
  }
}

describe("reports page", () => {
  function tableNodeNames() {
    const table = screen.getByRole("table")
    return within(table)
      .getAllByRole("row")
      .slice(1)
      .map((row) => within(row).getAllByRole("cell")[0]?.textContent ?? "")
      .map((text) => text.replace(/\d+ \/ \d+ samples/, "").trim())
  }

  it("loads availability data and refetches when filters change", async () => {
    const user = userEvent.setup()
    const availabilityReportMock = vi.mocked(api.availabilityReport)
    availabilityReportMock.mockResolvedValue(report())

    render(<Reports />)

    expect(await screen.findByRole("heading", { name: "Reports" })).toBeInTheDocument()
    expect((await screen.findAllByText("99.50%")).length).toBeGreaterThanOrEqual(2)
    expect(screen.getAllByText("alpha").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("prod").length).toBeGreaterThanOrEqual(2)
    expect(screen.getAllByText("42.5 ms").length).toBeGreaterThanOrEqual(2)
    expect(screen.getByText("Availability ranking")).toBeInTheDocument()
    expect(screen.getByText("Offline impact")).toBeInTheDocument()
    expect(screen.getByText("SLA distribution")).toBeInTheDocument()

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

  it("orders the availability ranking from highest availability to lowest", async () => {
    vi.mocked(api.availabilityReport).mockResolvedValue(report({
      nodes: [
        {
          node_id: "node-a",
          name: "alpha",
          tags: ["prod"],
          last_seen: Math.floor(Date.now() / 1000),
          availability_percent: 99.5,
          expected_samples: 20,
          observed_samples: 19,
          offline_duration_seconds: 60,
          outage_count: 1,
          latency_p95_ms: 42.5,
        },
        {
          node_id: "node-b",
          name: "bravo",
          tags: ["prod"],
          last_seen: Math.floor(Date.now() / 1000),
          availability_percent: 99.99,
          expected_samples: 20,
          observed_samples: 20,
          offline_duration_seconds: 0,
          outage_count: 0,
          latency_p95_ms: 24.1,
        },
        {
          node_id: "node-c",
          name: "charlie",
          tags: ["dev"],
          last_seen: Math.floor(Date.now() / 1000),
          availability_percent: 99.7,
          expected_samples: 20,
          observed_samples: 20,
          offline_duration_seconds: 20,
          outage_count: 1,
          latency_p95_ms: 31.2,
        },
      ],
    }))

    render(<Reports />)

    const rankingRows = await screen.findAllByTestId("availability-ranking-row")
    expect(rankingRows.map((row) => row.textContent)).toEqual(["bravo", "charlie", "alpha"])
  })

  it("sorts the SLA table and shows recent outage timing", async () => {
    const user = userEvent.setup()
    const now = Math.floor(Date.now() / 1000)
    vi.mocked(api.availabilityReport).mockResolvedValue(report({
      nodes: [
        {
          node_id: "node-a",
          name: "alpha",
          tags: ["prod"],
          last_seen: now,
          availability_percent: 99.5,
          expected_samples: 20,
          observed_samples: 19,
          offline_duration_seconds: 60,
          outage_count: 1,
          last_outage_start: now - 180,
          last_outage_end: now - 120,
          latency_p95_ms: 42.5,
        },
        {
          node_id: "node-b",
          name: "bravo",
          tags: ["prod"],
          last_seen: now,
          availability_percent: 99.99,
          expected_samples: 20,
          observed_samples: 20,
          offline_duration_seconds: 0,
          outage_count: 0,
          latency_p95_ms: 24.1,
        },
        {
          node_id: "node-c",
          name: "charlie",
          tags: ["dev"],
          last_seen: now,
          availability_percent: 98.2,
          expected_samples: 20,
          observed_samples: 18,
          offline_duration_seconds: 180,
          outage_count: 2,
          last_outage_start: now - 360,
          last_outage_end: now - 240,
          latency_p95_ms: 31.2,
        },
      ],
    }))

    render(<Reports />)

    await screen.findByRole("columnheader", { name: /Recent outage/i })

    expect(tableNodeNames()).toEqual(["charlie", "alpha", "bravo"])
    expect(screen.getByText("Recent outage")).toBeInTheDocument()
    expect(screen.getAllByText("1m").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("2m").length).toBeGreaterThanOrEqual(1)

    await user.click(screen.getByRole("button", { name: /Node/i }))
    expect(tableNodeNames()).toEqual(["alpha", "bravo", "charlie"])

    await user.click(screen.getByRole("button", { name: /Node/i }))
    expect(tableNodeNames()).toEqual(["charlie", "bravo", "alpha"])
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

    expect((await screen.findAllByText("alpha")).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(1)
  })
})
