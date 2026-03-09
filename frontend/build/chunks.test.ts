import { describe, expect, it } from "vitest"
import { getManualChunkName } from "./chunks"

describe("manual chunk rules", () => {
  it("splits recharts into a dedicated chunk", () => {
    expect(getManualChunkName("/opt/thism/frontend/node_modules/recharts/es6/chart/LineChart.js")).toBe("vendor-recharts")
  })

  it("leaves unrelated modules in default chunking", () => {
    expect(getManualChunkName("/opt/thism/frontend/src/pages/Dashboard.tsx")).toBeUndefined()
    expect(getManualChunkName("/opt/thism/frontend/node_modules/react/index.js")).toBeUndefined()
  })
})
