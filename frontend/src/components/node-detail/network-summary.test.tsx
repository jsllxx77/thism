import { render, screen, within } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { NetworkSummary } from "./NetworkSummary"

describe("NetworkSummary", () => {
  it("renders four compact network summary cards", () => {
    render(
      <NetworkSummary
        inboundTotal="12.3 GB"
        outboundTotal="8.9 GB"
        inboundSpeed="1.4 MB/s"
        outboundSpeed="512 KB/s"
      />,
    )

    const list = screen.getByRole("list")
    expect(within(list).getAllByRole("listitem")).toHaveLength(4)

    const items = [
      { label: "Inbound Total", value: "12.3 GB" },
      { label: "Outbound Total", value: "8.9 GB" },
      { label: "Inbound Speed", value: "1.4 MB/s" },
      { label: "Outbound Speed", value: "512 KB/s" },
    ]

    items.forEach(({ label, value }) => {
      const card = screen.getByText(label).closest("li")
      expect(card).toBeInTheDocument()
      expect(within(card as HTMLElement).getByText(value)).toBeInTheDocument()
    })

    const firstPanel = within(list).getAllByRole("listitem")[0]?.firstElementChild as HTMLElement | null
    expect(firstPanel?.className).toContain("enterprise-surface")
  })

  it("renders em dashes for missing values", () => {
    render(
      <NetworkSummary
        inboundTotal="—"
        outboundTotal="—"
        inboundSpeed="—"
        outboundSpeed="—"
      />,
    )

    expect(screen.getAllByText("—")).toHaveLength(4)
  })
})
