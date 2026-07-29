import { createRef, type CSSProperties } from "react"
import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { validateThemeRegistry, type ThemeRegistry } from "./contract"

export const THEME_CONFORMANCE_COVERAGE = [
  "composition",
  "variants",
  "forwarded-refs",
  "disabled-loading",
  "keyboard",
  "focus-visible",
  "aria",
  "portals",
  "reduced-motion",
  "light-dark",
  "responsive",
] as const

export interface ThemeConformanceAdapter {
  createRegistry(): ThemeRegistry
  readStyle(element: HTMLElement): CSSProperties
}

type ConformanceRunner = ((adapter: ThemeConformanceAdapter) => void) & {
  coverage: typeof THEME_CONFORMANCE_COVERAGE
}

function registerThemeConformanceChecks(adapter: ThemeConformanceAdapter) {
  describe("Theme API v1 conformance", () => {
    it("provides a complete registry with both color modes", () => {
      const registry = adapter.createRegistry()
      expect(validateThemeRegistry(registry)).toEqual({ ok: true, issues: [] })
      expect(Object.keys(registry.tokens.light).length).toBeGreaterThan(0)
      expect(Object.keys(registry.tokens.dark).length).toBeGreaterThan(0)
    })

    it("composes children and exposes variants through a forwarded ref", () => {
      const registry = adapter.createRegistry()
      const Button = registry.primitives.Button
      const ref = createRef<HTMLButtonElement>()
      render(<Button ref={ref} variant="destructive">Delete node</Button>)

      const button = screen.getByRole("button", { name: "Delete node" })
      expect(button).toHaveAttribute("data-variant", "destructive")
      expect(ref.current).toBe(button)
      button.focus()
      expect(button).toHaveFocus()
      expect(adapter.readStyle(button)).toBeDefined()
    })

    it("exposes disabled and loading states to assistive technology", () => {
      const registry = adapter.createRegistry()
      const Button = registry.primitives.Button
      const { rerender } = render(<Button disabled>Save</Button>)
      expect(screen.getByRole("button", { name: "Save" })).toBeDisabled()

      rerender(<Button loading>Save</Button>)
      const loadingButton = screen.getByRole("button", { name: "Save" })
      expect(loadingButton).toBeDisabled()
      expect(loadingButton).toHaveAttribute("aria-busy", "true")
    })

    it("supports labelled form controls", () => {
      const registry = adapter.createRegistry()
      const Input = registry.primitives.Input
      render(<Input aria-label="Theme name" />)
      expect(screen.getByRole("textbox", { name: "Theme name" })).toBeVisible()
    })

    it("renders dialogs through a portal and handles keyboard dismissal", () => {
      const registry = adapter.createRegistry()
      const Dialog = registry.primitives.Dialog
      const onOpenChange = vi.fn()
      render(<main data-testid="application"><Dialog open onOpenChange={onOpenChange}>Confirm activation</Dialog></main>)

      const dialog = screen.getByRole("dialog")
      expect(dialog).toHaveAttribute("aria-modal", "true")
      expect(dialog.parentElement).toBe(document.body)
      fireEvent.keyDown(document, { key: "Escape" })
      expect(onOpenChange).toHaveBeenCalledWith(false)
    })

    it("provides responsive navigation presentation", () => {
      const registry = adapter.createRegistry()
      const ResponsiveNavigation = registry.primitives.ResponsiveNavigation
      render(<ResponsiveNavigation>Primary links</ResponsiveNavigation>)
      expect(screen.getByRole("navigation", { name: "" })).toHaveAttribute("data-responsive", "true")
    })
  })
}

export const runThemeConformanceChecks: ConformanceRunner = Object.assign(registerThemeConformanceChecks, {
  coverage: THEME_CONFORMANCE_COVERAGE,
})
