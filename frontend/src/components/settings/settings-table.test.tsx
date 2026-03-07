import { describe, expect, it, vi } from "vitest"
import { render, screen, within } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import type { Node } from "../../lib/api"
import { api } from "../../lib/api"
import { NodesTable } from "./NodesTable"

vi.mock("../../lib/api", () => ({
  api: {
    renameNode: vi.fn(),
    installCommand: vi.fn(),
    deleteNode: vi.fn(),
  },
}))

function node(overrides: Partial<Node>): Node {
  return {
    id: "id",
    name: "name",
    ip: "127.0.0.1",
    os: "linux",
    arch: "amd64",
    created_at: 1733011200,
    last_seen: 1733011200,
    online: true,
    ...overrides,
  }
}

describe("settings nodes table", () => {
  it("renders, filters, sorts, and runs node actions through dialogs", async () => {
    const user = userEvent.setup()
    const renameNodeMock = vi.mocked(api.renameNode)
    const installCommandMock = vi.mocked(api.installCommand)
    const deleteNodeMock = vi.mocked(api.deleteNode)
    const onUpdated = vi.fn().mockResolvedValue(undefined)

    renameNodeMock.mockResolvedValue({
      id: "n2",
      name: "renamed-node",
      ip: "127.0.0.1",
      os: "linux",
      arch: "amd64",
      created_at: 1733011200,
      last_seen: 1733011200,
      online: true,
    })
    installCommandMock.mockResolvedValue({ command: "curl -fsSL http://localhost/install.sh?token=t1&name=alpha | bash" })
    deleteNodeMock.mockResolvedValue({ ok: true })

    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
    })

    const { container } = render(
      <NodesTable
        onUpdated={onUpdated}
        nodes={[
          node({ id: "n1", name: "zeta", online: false }),
          node({ id: "n2", name: "alpha", online: true }),
        ]}
      />
    )

    expect(screen.getByText("Nodes")).toBeInTheDocument()
    const tableShell = container.firstElementChild as HTMLElement | null
    expect(tableShell?.className).toContain("enterprise-surface")
    const renameButtons = screen.getAllByRole("button", { name: "Rename" })
    const scriptButtons = screen.getAllByRole("button", { name: "Get Script" })
    const removeButtons = screen.getAllByRole("button", { name: "Delete" })

    expect(renameButtons.length).toBeGreaterThan(0)
    expect(scriptButtons.length).toBeGreaterThan(0)
    expect(removeButtons.length).toBeGreaterThan(0)

    await user.selectOptions(screen.getByLabelText("Settings status filter"), "online")
    expect(screen.getByText("alpha")).toBeInTheDocument()
    expect(screen.queryByText("zeta")).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText("Settings status filter"), "all")
    await user.click(screen.getByRole("button", { name: "Name" }))
    const rows = screen.getAllByRole("row").slice(1)
    expect(rows[0]).toHaveTextContent("alpha")

    await user.click(screen.getAllByRole("button", { name: "Rename" })[0])
    expect(await screen.findByText("Rename Node")).toBeInTheDocument()
    const nameInput = screen.getByLabelText("Node Name")
    await user.clear(nameInput)
    await user.type(nameInput, "renamed-node")
    await user.click(screen.getByRole("button", { name: "Save" }))
    expect(renameNodeMock).toHaveBeenCalledWith("n2", "renamed-node")
    expect(onUpdated).toHaveBeenCalledTimes(1)

    await user.click(screen.getAllByRole("button", { name: "Get Script" })[0])
    expect(installCommandMock).toHaveBeenCalledWith("n2")
    expect(await screen.findByText("Install Command")).toBeInTheDocument()
    expect(onUpdated).toHaveBeenCalledTimes(1)
    await user.click(screen.getByRole("button", { name: "Copy command" }))
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("curl -fsSL http://localhost/install.sh?token=t1&name=alpha | bash")
    await user.click(screen.getByRole("button", { name: "Done" }))
    expect(onUpdated).toHaveBeenCalledTimes(1)

    await user.click(screen.getAllByRole("button", { name: "Delete" })[0])
    const deleteDialog = await screen.findByRole("dialog", { name: "Delete Node" })
    await user.click(within(deleteDialog).getByRole("button", { name: /Delete/ }))
    expect(deleteNodeMock).toHaveBeenCalledWith("n2")
    expect(onUpdated).toHaveBeenCalledTimes(2)
  }, 10000)

  it("shows an inline error when copying install command is denied", async () => {
    const user = userEvent.setup()
    const installCommandMock = vi.mocked(api.installCommand)

    installCommandMock.mockResolvedValue({ command: "curl -fsSL http://localhost/install.sh?token=t1&name=alpha | bash" })
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockRejectedValue(new Error("permission denied")) },
      configurable: true,
    })

    render(
      <NodesTable
        nodes={[node({ id: "n2", name: "alpha", online: true })]}
      />
    )

    await user.click(screen.getByRole("button", { name: "Get Script" }))
    expect(await screen.findByText("Install Command")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "Copy command" }))
    expect(await screen.findByText("Clipboard access was denied. Please copy the command manually.")).toBeInTheDocument()
  })
})
