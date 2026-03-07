import { describe, expect, it } from "vitest"
import { loadLanguageMessages } from "./messages"

describe("language message loading", () => {
  it("loads the Chinese catalog on demand", async () => {
    const messages = await loadLanguageMessages("zh-CN")
    expect(messages.common.brand).toBe("ThisM 控制台")
  })
})
