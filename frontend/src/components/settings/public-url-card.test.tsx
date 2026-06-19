import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { PublicURLCard } from "./PublicURLCard"

const publicURLSettingsMock = vi.fn()
const updatePublicURLSettingsMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    publicURLSettings: (...args: unknown[]) => publicURLSettingsMock(...args),
    updatePublicURLSettings: (...args: unknown[]) => updatePublicURLSettingsMock(...args),
  },
}))

describe("public URL settings card", () => {
  beforeEach(() => {
    publicURLSettingsMock.mockReset()
    updatePublicURLSettingsMock.mockReset()
    publicURLSettingsMock.mockResolvedValue({ public_url: "" })
    updatePublicURLSettingsMock.mockResolvedValue({ public_url: "https://thism.777114.xyz" })
  })

  it("loads the current public URL and saves updates", async () => {
    const user = userEvent.setup()

    render(<PublicURLCard />)

    expect(await screen.findByRole("heading", { name: "Public URL", level: 3 })).toBeInTheDocument()
    expect(screen.getByText("Auto-detect")).toBeInTheDocument()

    const input = screen.getByLabelText("Public URL")
    await user.type(input, "https://thism.777114.xyz")
    await user.click(screen.getByRole("button", { name: "Save public URL" }))

    await waitFor(() => {
      expect(updatePublicURLSettingsMock).toHaveBeenCalledWith({ public_url: "https://thism.777114.xyz" })
    })

    expect(await screen.findByText("Public URL updated.")).toBeInTheDocument()
    expect(screen.getByText("Configured")).toBeInTheDocument()
    expect(input).toHaveValue("https://thism.777114.xyz")
  })
})
