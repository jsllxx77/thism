import { describe, expect, it, vi } from "vitest"
import { CLIPBOARD_COPY_DENIED_MESSAGE, copyTextToClipboard } from "./clipboard"

describe("copyTextToClipboard", () => {
  it("returns ok when clipboard write succeeds", async () => {
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
      configurable: true,
    })

    await expect(copyTextToClipboard("hello")).resolves.toEqual({ ok: true })
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith("hello")
  })

  it("returns a friendly error when clipboard write is denied", async () => {
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText: vi.fn().mockRejectedValue(new Error("permission denied")) },
      configurable: true,
    })

    await expect(copyTextToClipboard("hello")).resolves.toEqual({
      ok: false,
      message: CLIPBOARD_COPY_DENIED_MESSAGE,
    })
  })

  it("returns a friendly error when clipboard API is unavailable", async () => {
    Object.defineProperty(navigator, "clipboard", {
      value: undefined,
      configurable: true,
    })

    await expect(copyTextToClipboard("hello")).resolves.toEqual({
      ok: false,
      message: CLIPBOARD_COPY_DENIED_MESSAGE,
    })
  })
})
