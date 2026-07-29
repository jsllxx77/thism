import type React from "react"
import { render, screen } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { beforeEach, describe, expect, it, vi } from "vitest"

import App from "../App"
import { ThemeModeProvider } from "../theme/mode"
import { AppThemeProvider } from "../theme/theme"
import { DEFAULT_SHADCN_PLUGIN } from "./default-shadcn"
import { ThemePluginRuntimeProvider, useThemePlugin, type ThemePluginModule } from "./runtime"

function RuntimeProbe() {
  const plugin = useThemePlugin()
  return <output>{plugin.id}@{plugin.version}</output>
}

vi.mock("../lib/api", () => ({
  api: { session: vi.fn().mockResolvedValue({ role: "admin" }) },
}))

function CustomRootShell({ children }: { children?: React.ReactNode }) {
  return <section aria-label="Custom theme shell">{children}</section>
}

describe("theme plugin runtime", () => {
  beforeEach(() => {
    window.localStorage.clear()
    document.documentElement.classList.remove("dark")
  })

  it("uses default-shadcn as the fresh runtime context", () => {
    render(<RuntimeProbe />)
    expect(screen.getByText("default-shadcn@1.0.0")).toBeInTheDocument()
  })

  it("boots the embedded default plugin with mode-bound root metadata", () => {
    render(
      <ThemeModeProvider>
        <ThemePluginRuntimeProvider plugin={DEFAULT_SHADCN_PLUGIN}>
          <RuntimeProbe />
        </ThemePluginRuntimeProvider>
      </ThemeModeProvider>,
    )

    expect(screen.getByText("default-shadcn@1.0.0")).toBeInTheDocument()
    const root = screen.getByTestId("theme-plugin-root")
    expect(root).toHaveAttribute("data-thism-theme-id", "default-shadcn")
    expect(root).toHaveAttribute("data-thism-theme-version", "1.0.0")
    expect(root).toHaveAttribute("data-thism-theme-mode", "light")
    expect(root.style.getPropertyValue("--thism-plugin-background")).toBe("0 0% 100%")
  })

  it("injects dark-mode metadata and semantic plugin tokens", () => {
    window.localStorage.setItem("thism-theme", "dark")
    render(
      <ThemeModeProvider>
        <ThemePluginRuntimeProvider plugin={DEFAULT_SHADCN_PLUGIN}>
          <RuntimeProbe />
        </ThemePluginRuntimeProvider>
      </ThemeModeProvider>,
    )

    const root = screen.getByTestId("theme-plugin-root")
    expect(root).toHaveAttribute("data-thism-theme-mode", "dark")
    expect(root.style.getPropertyValue("--thism-plugin-background")).toBe("222 47% 11%")
  })

  it("lets the host router render through a registry-provided root shell", async () => {
    const plugin: ThemePluginModule = {
      ...DEFAULT_SHADCN_PLUGIN,
      id: "test-shell",
      registry: {
        ...DEFAULT_SHADCN_PLUGIN.registry,
        shells: { ...DEFAULT_SHADCN_PLUGIN.registry.shells, RootShell: CustomRootShell },
      },
    }

    render(
      <ThemeModeProvider>
        <AppThemeProvider>
          <ThemePluginRuntimeProvider plugin={plugin}>
            <MemoryRouter initialEntries={["/missing"]}>
              <App />
            </MemoryRouter>
          </ThemePluginRuntimeProvider>
        </AppThemeProvider>
      </ThemeModeProvider>,
    )

    expect(await screen.findByRole("region", { name: "Custom theme shell" })).toBeInTheDocument()
    expect(await screen.findByText("Nothing here")).toBeInTheDocument()
  })

  it("rejects an incomplete plugin before rendering children", () => {
    const invalidPlugin = {
      ...DEFAULT_SHADCN_PLUGIN,
      registry: { ...DEFAULT_SHADCN_PLUGIN.registry, primitives: {} },
    } as ThemePluginModule

    expect(() => render(
      <ThemeModeProvider>
        <ThemePluginRuntimeProvider plugin={invalidPlugin}>
          <p>Must not render</p>
        </ThemePluginRuntimeProvider>
      </ThemeModeProvider>,
    )).toThrow(/invalid theme plugin default-shadcn/i)
    expect(screen.queryByText("Must not render")).not.toBeInTheDocument()
  })
})
