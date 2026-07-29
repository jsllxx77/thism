import { forwardRef, useEffect, type CSSProperties, type ReactNode } from "react"
import { createPortal } from "react-dom"
import { describe, expect, it } from "vitest"

import { runThemeConformanceChecks, type ThemeConformanceAdapter } from "./conformance"
import { REQUIRED_PORTALS, REQUIRED_PRIMITIVES, REQUIRED_SHELLS, REQUIRED_SLOTS, THEME_API_VERSION, type ThemeComponent, type ThemeRegistry } from "./contract"

const passthrough = forwardRef<HTMLDivElement, { children?: ReactNode; className?: string }>(function Passthrough({ children, className }, ref) {
  return <div ref={ref} className={className}>{children}</div>
}) as ThemeComponent

const Button = forwardRef<HTMLButtonElement, { children?: ReactNode; disabled?: boolean; loading?: boolean; variant?: string }>(function Button(
  { children, disabled, loading, variant },
  ref,
) {
  return <button ref={ref} disabled={disabled || loading} aria-busy={loading || undefined} data-variant={variant}>{children}</button>
}) as ThemeComponent

const Input = forwardRef<HTMLInputElement, { "aria-label"?: string }>(function Input(props, ref) {
  return <input ref={ref} {...props} />
}) as ThemeComponent

function Dialog({ open, children, onOpenChange }: { open?: boolean; children?: ReactNode; onOpenChange?: (open: boolean) => void }) {
  useEffect(() => {
    function close(event: KeyboardEvent) {
      if (event.key === "Escape") onOpenChange?.(false)
    }
    document.addEventListener("keydown", close)
    return () => document.removeEventListener("keydown", close)
  }, [onOpenChange])
  return open ? createPortal(<div role="dialog" aria-modal="true">{children}</div>, document.body) : null
}

const ResponsiveNavigation = ({ children }: { children?: ReactNode }) => <nav data-responsive="true">{children}</nav>

function registry(): ThemeRegistry {
  const primitives = Object.fromEntries(REQUIRED_PRIMITIVES.map((name) => [name, passthrough])) as unknown as ThemeRegistry["primitives"]
  primitives.Button = Button
  primitives.Input = Input
  primitives.Dialog = Dialog as ThemeComponent
  primitives.ResponsiveNavigation = ResponsiveNavigation as ThemeComponent
  return {
    apiVersion: THEME_API_VERSION,
    primitives,
    shells: Object.fromEntries(REQUIRED_SHELLS.map((name) => [name, passthrough])) as unknown as ThemeRegistry["shells"],
    slots: Object.fromEntries(REQUIRED_SLOTS.map((name) => [name, passthrough])) as unknown as ThemeRegistry["slots"],
    portals: Object.fromEntries(REQUIRED_PORTALS.map((name) => [name, passthrough])) as unknown as ThemeRegistry["portals"],
    tokens: { light: { background: "white" }, dark: { background: "black" } },
    settings: [],
  }
}

const adapter: ThemeConformanceAdapter = {
  createRegistry: registry,
  readStyle: (element) => element.style as CSSProperties,
}

runThemeConformanceChecks(adapter)

describe("shared theme conformance harness", () => {
  it("reports the behavior surfaces it enforces", () => {
    expect(runThemeConformanceChecks.coverage).toEqual([
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
    ])
  })
})
