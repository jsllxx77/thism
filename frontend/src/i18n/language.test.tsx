import { beforeEach, describe, expect, it } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { LanguageProvider, useLanguage } from "./language"

function Probe() {
  const { language, toggleLanguage, labelForLanguageToggle } = useLanguage()

  return (
    <div>
      <span>{language}</span>
      <span>{labelForLanguageToggle}</span>
      <button type="button" onClick={toggleLanguage}>
        toggle
      </button>
    </div>
  )
}

describe("language provider", () => {
  beforeEach(() => {
    window.localStorage.clear()
    document.cookie = "thism-lang=; path=/; max-age=0; samesite=lax"
  })

  it("loads the preferred Chinese language from storage", async () => {
    window.localStorage.setItem("thism-language", "zh-CN")

    render(
      <LanguageProvider>
        <Probe />
      </LanguageProvider>,
    )

    expect(await screen.findByText("zh-CN")).toBeInTheDocument()
    expect(await screen.findByText("English")).toBeInTheDocument()
  })

  it("defaults to English and toggles to Chinese with persisted label", async () => {
    window.localStorage.clear()
    const user = userEvent.setup()

    render(
      <LanguageProvider>
        <Probe />
      </LanguageProvider>,
    )

    expect(screen.getByText("en")).toBeInTheDocument()
    expect(screen.getByText("中文")).toBeInTheDocument()

    await user.click(screen.getByRole("button", { name: "toggle" }))

    expect(screen.getByText("zh-CN")).toBeInTheDocument()
    expect(screen.getByText("English")).toBeInTheDocument()
    expect(window.localStorage.getItem("thism-language")).toBe("zh-CN")
    expect(document.cookie).toContain("thism-lang=zh-CN")
  })
})
