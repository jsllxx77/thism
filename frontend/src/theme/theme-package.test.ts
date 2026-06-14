import { beforeEach, describe, expect, it } from "vitest"

import { applyThemeDefinition, parseThemePackage } from "./theme-context"

const auroraThemePackage = {
  type: "thism-theme",
  version: 1,
  id: "aurora-command",
  name: "Aurora Command",
  description: "A high-contrast operations theme with compact command surfaces.",
  accent: "#8b5cf6",
  tokens: {
    light: {
      background: "248 80% 98%",
      foreground: "246 38% 12%",
      card: "0 0% 100%",
      "card-foreground": "246 38% 12%",
      popover: "0 0% 100%",
      "popover-foreground": "246 38% 12%",
      primary: "265 83% 58%",
      "primary-foreground": "0 0% 100%",
      secondary: "252 62% 94%",
      "secondary-foreground": "246 38% 14%",
      muted: "252 44% 92%",
      "muted-foreground": "248 13% 42%",
      accent: "176 66% 43%",
      "accent-foreground": "246 38% 12%",
      destructive: "0 78% 56%",
      "destructive-foreground": "0 0% 100%",
      border: "252 30% 84%",
      input: "252 30% 82%",
      ring: "265 83% 58%",
      "chart-1": "265 83% 58%",
      "chart-2": "176 66% 43%",
      "chart-3": "35 92% 52%",
      "chart-4": "330 81% 60%",
      "chart-5": "205 88% 54%",
    },
    dark: {
      background: "246 42% 8%",
      foreground: "248 80% 96%",
      card: "246 35% 12%",
      "card-foreground": "248 80% 96%",
      popover: "246 35% 12%",
      "popover-foreground": "248 80% 96%",
      primary: "265 91% 70%",
      "primary-foreground": "246 42% 8%",
      secondary: "248 28% 18%",
      "secondary-foreground": "248 80% 96%",
      muted: "248 24% 16%",
      "muted-foreground": "250 22% 72%",
      accent: "176 70% 52%",
      "accent-foreground": "246 42% 8%",
      destructive: "0 66% 52%",
      "destructive-foreground": "0 0% 100%",
      border: "248 22% 24%",
      input: "248 22% 22%",
      ring: "265 91% 70%",
      "chart-1": "265 91% 70%",
      "chart-2": "176 70% 52%",
      "chart-3": "35 92% 62%",
      "chart-4": "330 81% 70%",
      "chart-5": "205 88% 64%",
    },
  },
  appearance: {
    radius: "1.25rem",
    cardRadius: "1.5rem",
    panelRadius: "0.75rem",
    controlRadius: "0.375rem",
    density: "compact",
    surface: "command",
    background: "grid",
    navigation: "solid",
    cardPadding: "0.875rem",
    panelPadding: "1rem",
    fontFamily: "\"Fira Sans\", \"Segoe UI\", sans-serif",
    monoFontFamily: "\"Fira Code\", \"SFMono-Regular\", monospace",
    shadow: "0 18px 46px rgba(32, 18, 96, 0.22)",
  },
}

describe("theme packages", () => {
  beforeEach(() => {
    document.documentElement.removeAttribute("data-theme")
    document.documentElement.removeAttribute("data-theme-source")
    document.documentElement.removeAttribute("data-theme-surface")
    document.documentElement.removeAttribute("data-theme-background")
    document.documentElement.removeAttribute("data-theme-navigation")
    document.documentElement.removeAttribute("style")
  })

  it("parses a thism theme package into a custom shadcn theme definition", () => {
    const theme = parseThemePackage(JSON.stringify(auroraThemePackage))

    expect(theme.name).toBe("custom:aurora-command")
    expect(theme.label).toBe("Aurora Command")
    expect(theme.source).toBe("custom")
    expect(theme.accent).toBe("#8b5cf6")
    expect(theme.package.tokens.dark.primary).toBe("265 91% 70%")
    expect(theme.package.appearance.surface).toBe("command")
  })

  it("applies custom theme package tokens and appearance variables at runtime", () => {
    const theme = parseThemePackage(JSON.stringify(auroraThemePackage))

    applyThemeDefinition(theme, "dark")

    expect(document.documentElement.dataset.theme).toBe("custom:aurora-command")
    expect(document.documentElement.dataset.themeSource).toBe("custom")
    expect(document.documentElement.dataset.themeSurface).toBe("command")
    expect(document.documentElement.dataset.themeBackground).toBe("grid")
    expect(document.documentElement.style.getPropertyValue("--primary")).toBe("265 91% 70%")
    expect(document.documentElement.style.getPropertyValue("--card")).toBe("246 35% 12%")
    expect(document.documentElement.style.getPropertyValue("--theme-card-radius")).toBe("1.5rem")
    expect(document.documentElement.style.getPropertyValue("--theme-panel-radius")).toBe("0.75rem")
    expect(document.documentElement.style.getPropertyValue("--app-font-family")).toContain("Fira Sans")
  })

  it("rejects theme files that are not thism theme packages", () => {
    expect(() => parseThemePackage(JSON.stringify({ type: "other-theme", name: "Nope" }))).toThrow("Invalid theme file")
  })
})
