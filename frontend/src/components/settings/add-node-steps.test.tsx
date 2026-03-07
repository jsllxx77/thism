import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { AddNodeModal } from "../AddNodeModal"

const registerMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    register: (...args: unknown[]) => registerMock(...args),
  },
}))

beforeEach(() => {
  registerMock.mockReset()
  registerMock.mockResolvedValue({ id: "n1", token: "token-123" })
  Object.defineProperty(navigator, "clipboard", {
    value: {
      writeText: vi.fn().mockResolvedValue(undefined),
    },
    configurable: true,
  })
})

describe("add node modal steps", () => {
  it("validates name, advances to command step, and supports copy", async () => {
    const user = userEvent.setup()

    render(<AddNodeModal onClose={() => {}} onCreated={() => {}} />)
    expect(screen.getByText("Add Node")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: /get install command/i }))
    expect(screen.getByText("Node name is required")).toBeInTheDocument()

    await user.type(screen.getByLabelText("Node Name"), "prod-node")
    await user.click(screen.getByRole("button", { name: /get install command/i }))

    expect(await screen.findByText("Install Command")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Copy command" }))
    expect(await screen.findByText("Copied")).toBeInTheDocument()
  })

  it("shows an inline error when clipboard copy is denied", async () => {
    const user = userEvent.setup()
    Object.defineProperty(navigator, "clipboard", {
      value: {
        writeText: vi.fn().mockRejectedValue(new Error("permission denied")),
      },
      configurable: true,
    })

    render(<AddNodeModal onClose={() => {}} onCreated={() => {}} />)

    await user.type(screen.getByLabelText("Node Name"), "prod-node")
    await user.click(screen.getByRole("button", { name: /get install command/i }))
    expect(await screen.findByText("Install Command")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Copy command" }))
    expect(await screen.findByText("Clipboard access was denied. Please copy the command manually.")).toBeInTheDocument()
  })
})
