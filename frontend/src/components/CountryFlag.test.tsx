import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { CountryFlag } from "./CountryFlag"

describe("CountryFlag", () => {
  it("renders a compact emoji flag with country-code metadata", () => {
    render(<CountryFlag countryCode="HK" />)

    const flag = screen.getByRole("img", { name: "HK" })
    expect(flag).toHaveClass("country-flag")
    expect(flag).not.toHaveClass("fi")
    expect(flag).not.toHaveClass("fi-hk")
    expect(flag).toHaveAttribute("data-country-code", "HK")
    expect(screen.getByText("🇭🇰")).toHaveClass("country-flag__emoji")
    expect(screen.queryByText("HK")).not.toBeInTheDocument()
  })

  it.each(["HK", "SG", "AU"])("includes a graphical fallback for %s when flag emoji fonts collapse", (countryCode) => {
    render(<CountryFlag countryCode={countryCode} />)

    const flag = screen.getByRole("img", { name: countryCode })
    const fallback = flag.querySelector(".country-flag__fallback")
    expect(fallback).toBeInstanceOf(SVGElement)
    expect(flag).toHaveAttribute("data-country-flag-fallback", "svg")
  })

  it("does not render for invalid country codes", () => {
    const { container } = render(<CountryFlag countryCode="HKG" />)

    expect(container.firstChild).toBeNull()
  })
})
