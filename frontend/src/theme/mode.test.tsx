import { act, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"
import { ThemeModeProvider, useThemeMode } from "./mode"
function Probe() { const { preference, mode } = useThemeMode(); return <output>{preference}:{mode}</output> }
describe("ThemeModeProvider", () => {
  beforeEach(() => { window.localStorage.clear() })
  it("owns system preference and reacts to operating-system changes", () => {
    let listener: (() => void) | undefined
    let dark = false
    Object.defineProperty(window, "matchMedia", { configurable: true, value: vi.fn(() => ({ get matches() { return dark }, media: "", onchange: null, addListener: vi.fn(), removeListener: vi.fn(), addEventListener: (_type: string, next: EventListenerOrEventListenerObject) => { listener = next as () => void }, removeEventListener: vi.fn(), dispatchEvent: vi.fn() } as MediaQueryList)) })
    render(<ThemeModeProvider><Probe /></ThemeModeProvider>)
    expect(screen.getByText("system:light")).toBeInTheDocument()
    expect(window.localStorage.getItem("thism-theme")).toBe("system")
    dark = true
    act(() => listener?.())
    expect(screen.getByText("system:dark")).toBeInTheDocument()
  })
})
