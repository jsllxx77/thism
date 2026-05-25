import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { CountryFlag } from "./CountryFlag"

describe("CountryFlag", () => {
  it("renders a compact emoji flag with stable country-code fallback text", () => {
    render(<CountryFlag countryCode="HK" />)

    const flag = screen.getByRole("img", { name: "HK" })
    expect(flag).toHaveClass("country-flag")
    expect(flag).not.toHaveClass("fi")
    expect(flag).not.toHaveClass("fi-hk")
    expect(flag).toHaveAttribute("data-country-code", "HK")
    expect(screen.getByText("HK")).toHaveClass("country-flag__code")
  })

  it("does not render for invalid country codes", () => {
    const { container } = render(<CountryFlag countryCode="HKG" />)

    expect(container.firstChild).toBeNull()
  })
})
