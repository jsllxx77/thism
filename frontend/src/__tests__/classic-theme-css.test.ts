import { readFileSync } from "node:fs"
import { resolve } from "node:path"
import { describe, expect, it } from "vitest"

const css = readFileSync(resolve(__dirname, "../index.css"), "utf8")

function blockFor(selector: string) {
  const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
  const match = css.match(new RegExp(`${escaped}\\s*\\{([\\s\\S]*?)\\n\\}`))
  return match?.[1] ?? ""
}

describe("classic theme CSS", () => {
  it("keeps the original classic surface settings as the default theme", () => {
    expect(blockFor(".app-surface-bg")).toContain("linear-gradient(180deg, #f8f9fb 0%, #f1f3f5 100%)")
    expect(blockFor(".dark .app-surface-bg")).toContain("linear-gradient(180deg, #121417 0%, #17191d 100%)")
    expect(blockFor(".panel-card")).toContain("background: rgba(255, 255, 255, 0.94)")
    expect(blockFor(".panel-card")).toContain("box-shadow: 0 10px 28px rgba(15, 23, 42, 0.06)")
    expect(blockFor(":root")).toContain("--theme-card-radius: 0.75rem")
    expect(blockFor(":root")).toContain("--theme-panel-radius: 0.75rem")
    expect(blockFor(":root")).toContain("--theme-control-radius: 0.5rem")
    expect(blockFor(".panel-card")).toContain("border-radius: var(--theme-panel-radius)")
    expect(blockFor(".enterprise-surface")).toContain("rgba(247, 249, 251, 0.92)")
    expect(blockFor(".enterprise-accent-button")).toContain("background: #2859ad")
  })

  it("reserves space between table accent bars and first-column content", () => {
    expect(blockFor(".motion-table-row > td:first-child")).toContain("padding-left: var(--motion-table-accent-gutter, 1rem);")
  })

  it("keeps custom solid themes flat without background overlays", () => {
    expect(blockFor('[data-theme-source="custom"] .app-surface-bg')).toContain("background: hsl(var(--background));")
    expect(css).toContain("background: hsl(var(--card));\n  border: var(--theme-border-width, 1px) solid hsl(var(--border) / 0.92);")
    expect(blockFor('[data-theme-source="custom"] .enterprise-inner-surface')).toContain("background: hsl(var(--card));")
    expect(blockFor('[data-theme-source="custom"] .enterprise-inner-surface')).toContain("box-shadow: none;")
    expect(blockFor('[data-theme-source="custom"] .node-card-shell::before')).toContain("opacity: 0;")
    expect(blockFor('[data-theme-source="custom"] .node-card-shell::before')).toContain("background: none;")
    expect(blockFor('[data-theme-source="custom"] .node-card-shell::after')).toContain("background: none;")
  })
})
