import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { MemoryRouter } from "react-router-dom"
import App from "../App"
import { AppThemeProvider } from "../theme/theme"
import { ThemeModeProvider } from "../theme/mode"

vi.mock("../lib/api", () => ({
  api: {
    session: vi.fn().mockResolvedValue({ role: "admin" }),
    nodes: vi.fn().mockResolvedValue({ nodes: [] }),
    metrics: vi.fn().mockResolvedValue({ metrics: [] }),
    processes: vi.fn().mockResolvedValue([]),
    services: vi.fn().mockResolvedValue({ services: [] }),
    register: vi.fn(),
  },
}))

vi.mock("../lib/ws", () => ({
  getDashboardWS: () => ({
    on: () => {},
    off: () => {},
  }),
}))

function mockMatchMedia(matches: boolean) {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
      matches: query === "(prefers-color-scheme: dark)" ? matches : false,
      media: query,
      onchange: null,
      addEventListener: () => {},
      removeEventListener: () => {},
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

describe("theme mode", () => {
  beforeEach(() => {
    localStorage.clear()
    document.documentElement.classList.remove("dark")
  })

  it("follows system dark preference on first load", () => {
    mockMatchMedia(true)
    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={["/"]}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>
    )

    expect(document.documentElement.classList.contains("dark")).toBe(true)
  })

  it("toggles mode and persists to localStorage", async () => {
    mockMatchMedia(false)
    const user = userEvent.setup()

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <MemoryRouter initialEntries={["/"]}>
            <App />
          </MemoryRouter>
        </AppThemeProvider>
      </ThemeModeProvider>
    )

    const button = await screen.findByRole("button", { name: "Toggle dark mode" })
    await user.click(button)
    expect(document.documentElement.classList.contains("dark")).toBe(true)
    expect(localStorage.getItem("thism-theme")).toBe("dark")
  })
})
