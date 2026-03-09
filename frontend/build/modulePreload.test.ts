import { describe, expect, it } from "vitest"
import { filterModulePreloadDependencies } from "./modulePreload"

describe("module preload filtering", () => {
  it("drops vendor-recharts from html entry preload deps", () => {
    expect(
      filterModulePreloadDependencies(
        "/assets/index-main.js",
        ["assets/vendor-recharts-abc.js", "assets/Settings-xyz.js", "assets/index-helper.js"],
        { hostId: "/opt/thism/frontend/dist/index.html", hostType: "html" },
      ),
    ).toEqual(["assets/Settings-xyz.js", "assets/index-helper.js"])
  })

  it("keeps js chunk dependency lists unchanged", () => {
    expect(
      filterModulePreloadDependencies(
        "assets/index-main.js",
        ["assets/vendor-recharts-abc.js", "assets/index-helper.js"],
        { hostId: "/opt/thism/frontend/dist/assets/index-main.js", hostType: "js" },
      ),
    ).toEqual(["assets/vendor-recharts-abc.js", "assets/index-helper.js"])
  })
})
