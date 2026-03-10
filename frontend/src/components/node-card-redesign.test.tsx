import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../lib/api"
import { NodeCard } from "./NodeCard"

function createNode(overrides: Partial<Node> = {}): Node {
  return {
    id: "n1",
    name: "alpha",
    ip: "10.0.0.1",
    os: "linux",
    arch: "amd64",
    created_at: 0,
    last_seen: 1733011200,
    online: true,
    ...overrides,
  }
}

describe("node card redesign", () => {
  it("renders status and metrics and supports click", async () => {
    const user = userEvent.setup()
    const onClick = vi.fn()

    const { container } = render(<NodeCard node={createNode()} cpu={42.3} memUsed={512} memTotal={1024} onClick={onClick} />)

    expect(screen.getByText("Online")).toBeInTheDocument()
    expect(screen.getByText("42.3%")).toBeInTheDocument()
    expect(screen.getByText("50.0%")).toBeInTheDocument()
    expect(screen.getByText(/Last seen/i)).toBeInTheDocument()

    const card = container.querySelector("button > div") as HTMLElement | null
    expect(card?.className).toContain("enterprise-surface")

    await user.click(screen.getByRole("button", { name: /alpha/i }))
    expect(onClick).toHaveBeenCalledTimes(1)
  })


  it("uses a softer orange progress tone in light mode for high load", () => {
    document.documentElement.classList.remove("dark")
    const { container } = render(<NodeCard node={createNode()} cpu={90} memUsed={900} memTotal={1000} />)

    const bars = Array.from(container.querySelectorAll('div[style]')) as HTMLDivElement[]
    expect(bars.length).toBeGreaterThanOrEqual(2)
    expect(bars[0]?.style.backgroundColor).toBe("rgb(193, 111, 111)")
    expect(bars[1]?.style.backgroundColor).toBe("rgb(193, 111, 111)")
  })

  it("uses a lighter green progress tone in light mode", () => {
    document.documentElement.classList.remove("dark")
    const { container } = render(<NodeCard node={createNode()} cpu={42.3} memUsed={512} memTotal={1024} />)

    const bars = Array.from(container.querySelectorAll('div[style]')) as HTMLDivElement[]
    expect(bars.length).toBeGreaterThanOrEqual(2)
    expect(bars[0]?.style.backgroundColor).toBe("rgb(125, 181, 140)")
    expect(bars[1]?.style.backgroundColor).toBe("rgb(125, 181, 140)")
  })

  it("shows offline state and last seen text", () => {
    render(<NodeCard node={createNode({ online: false })} cpu={0} memUsed={0} memTotal={1024} />)

    expect(screen.getByText("Offline")).toBeInTheDocument()
    expect(screen.getByText(/Last seen/i)).toBeInTheDocument()
  })

  it("shows placeholders when metrics have not arrived yet", () => {
    render(<NodeCard node={createNode()} />)

    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(2)
  })

  it('renders network speed when provided', () => {
    render(<NodeCard node={createNode()} cpu={12.3} memUsed={512} memTotal={1024} netRxSpeed={1024} netTxSpeed={2048} />)

    expect(screen.getByText(/1\.0 KB\/s/)).toBeInTheDocument()
    expect(screen.getByText(/2\.0 KB\/s/)).toBeInTheDocument()
  })

  it("can hide node IP for restricted views", () => {
    render(<NodeCard node={createNode()} showIP={false} />)

    expect(screen.queryByText("10.0.0.1")).not.toBeInTheDocument()
    expect(screen.getByText("linux/amd64")).toBeInTheDocument()
  })
})
