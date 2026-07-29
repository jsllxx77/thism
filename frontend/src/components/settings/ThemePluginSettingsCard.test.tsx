import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"
import { ThemeModeProvider } from "../../theme/mode"
import { DEFAULT_SHADCN_PLUGIN } from "../../theme-plugin/default-shadcn"
import { ThemePluginRuntimeProvider } from "../../theme-plugin/runtime"
import { ThemePluginSettingsCard } from "./ThemePluginSettingsCard"

const defaults = { version: "1.0.0", values: { compact: false, navigation: "sidebar", contentWidth: 1280, brandLabel: "thisM" } }
describe("ThemePluginSettingsCard", () => {
  it("renders declared controls and persists only valid values", () => {
    const onSettingsChange = vi.fn()
    render(<ThemeModeProvider><ThemePluginRuntimeProvider plugin={DEFAULT_SHADCN_PLUGIN} settings={defaults} onSettingsChange={onSettingsChange}><ThemePluginSettingsCard /></ThemePluginRuntimeProvider></ThemeModeProvider>)
    expect(screen.getByLabelText("Compact density")).toBeInTheDocument()
    expect(screen.getByLabelText("Navigation style")).toBeInTheDocument()
    expect(screen.getByLabelText("Content width")).toBeInTheDocument()
    expect(screen.getByLabelText("Brand label")).toBeInTheDocument()
    fireEvent.change(screen.getByLabelText("Content width"), { target: { value: "1700" } })
    expect(screen.getByRole("alert")).toBeInTheDocument()
    expect(onSettingsChange).not.toHaveBeenCalled()
    fireEvent.change(screen.getByLabelText("Content width"), { target: { value: "1320" } })
    expect(onSettingsChange).toHaveBeenCalledWith(expect.objectContaining({ values: expect.objectContaining({ contentWidth: 1320 }) }))
  })
})
