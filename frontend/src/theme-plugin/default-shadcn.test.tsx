import type React from "react"
import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { runThemeConformanceChecks } from "@thism/theme-sdk/testing"
import { DEFAULT_SHADCN_PLUGIN, EMBEDDED_THEME_PLUGINS } from "./default-shadcn"

runThemeConformanceChecks({
  createRegistry: () => DEFAULT_SHADCN_PLUGIN.registry,
  readStyle: (element) => element.style as React.CSSProperties,
})

describe("embedded default-shadcn plugin", () => {
  it("exposes compound components and semantic surfaces", () => {
    const Card = DEFAULT_SHADCN_PLUGIN.registry.primitives.Card as typeof DEFAULT_SHADCN_PLUGIN.registry.primitives.Card & { Content?: unknown }
    const Tabs = DEFAULT_SHADCN_PLUGIN.registry.primitives.Tabs as typeof DEFAULT_SHADCN_PLUGIN.registry.primitives.Tabs & { List?: unknown; Trigger?: unknown; Content?: unknown }
    expect(Card.Content).toBeDefined()
    expect(Tabs.List).toBeDefined()
    expect(Tabs.Trigger).toBeDefined()
    expect(Tabs.Content).toBeDefined()

    const Tooltip = DEFAULT_SHADCN_PLUGIN.registry.primitives.Tooltip
    const DropdownMenu = DEFAULT_SHADCN_PLUGIN.registry.primitives.DropdownMenu
    const ErrorState = DEFAULT_SHADCN_PLUGIN.registry.shells.ErrorState
    render(<><Tooltip>Capacity details</Tooltip><DropdownMenu>Actions</DropdownMenu><ErrorState>Unable to load</ErrorState></>)
    expect(screen.getByRole("tooltip")).toHaveTextContent("Capacity details")
    expect(screen.getByRole("menu")).toHaveTextContent("Actions")
    expect(screen.getByRole("alert")).toHaveTextContent("Unable to load")
  })

  it("is permanently installed and active for fresh runtime state", () => {
    expect(EMBEDDED_THEME_PLUGINS).toEqual([DEFAULT_SHADCN_PLUGIN])
    expect(DEFAULT_SHADCN_PLUGIN).toMatchObject({
      id: "default-shadcn",
      version: "1.0.0",
      source: "embedded",
      removable: false,
      activeByDefault: true,
    })
  })
})
