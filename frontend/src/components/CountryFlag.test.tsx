import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"
import { CountryFlag } from "./CountryFlag"

describe("CountryFlag", () => {
  it("renders a compact local SVG flag with country-code metadata", () => {
    render(<CountryFlag countryCode="HK" />)

    const flag = screen.getByRole("img", { name: "HK" })
    expect(flag).toHaveClass("country-flag")
    expect(flag).not.toHaveClass("fi")
    expect(flag).not.toHaveClass("fi-hk")
    expect(flag).toHaveAttribute("data-country-code", "HK")
    expect(flag).toHaveAttribute("data-country-flag-source", "local-svg")
    expect(flag.querySelector("img")).toHaveAttribute("src", "/assets/flags/HK.svg")
    expect(screen.queryByText("HK")).not.toBeInTheDocument()
  })

  it.each(["HK", "SG", "AU", "US"])("loads the local SVG asset for %s", (countryCode) => {
    render(<CountryFlag countryCode={countryCode} />)

    const flag = screen.getByRole("img", { name: countryCode })
    const image = flag.querySelector(".country-flag__image")
    expect(image).toBeInstanceOf(HTMLImageElement)
    expect(image).toHaveAttribute("src", `/assets/flags/${countryCode}.svg`)
  })

  it("accepts a regional-indicator emoji and resolves it to a local SVG", () => {
    render(<CountryFlag countryCode="🇭🇰" />)

    const flag = screen.getByRole("img", { name: "HK" })
    expect(flag).toHaveAttribute("data-country-code", "HK")
    expect(flag.querySelector("img")).toHaveAttribute("src", "/assets/flags/HK.svg")
  })

  it("does not render for invalid country codes", () => {
    const { container } = render(<CountryFlag countryCode="HKG" />)

    expect(container.firstChild).toBeNull()
  })
})
