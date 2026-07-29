import { describe, expect, it } from "vitest"

import { validateThemeBuild } from "./build"

const validBuild = {
  exports: ["theme"],
  bundledModules: ["lucide-react", "@radix-ui/react-dialog"],
  externalImports: ["react", "react-dom", "react/jsx-runtime", "@thism/theme-sdk"],
  entrySource: "export const theme = registry",
  css: `
    @layer thism-theme {
      [data-thism-theme-root] { color: hsl(var(--foreground)); }
      [data-thism-theme-portal="dialog"] { background: hsl(var(--background)); }
      [data-thism-theme-root] :focus-visible { outline: 2px solid currentColor; }
      @media (prefers-reduced-motion: reduce) { [data-thism-theme-root] * { animation-duration: 0.01ms; } }
      @media (max-width: 48rem) { [data-thism-theme-root] [data-thism-responsive-nav] { display: flex; } }
    }
  `,
}

describe("theme build contract", () => {
  it("accepts a self-contained build using only host-owned externals", () => {
    expect(validateThemeBuild(validBuild)).toEqual({ ok: true, issues: [] })
  })

  it("rejects host runtimes, unresolved externals, remote code, second roots, and missing exports", () => {
    expect(validateThemeBuild({
      ...validBuild,
      exports: [],
      bundledModules: ["react", "react-dom/client"],
      externalImports: ["react", "left-pad", "https://cdn.example/theme.js"],
      entrySource: "createRoot(document.body).render(app); import('https://cdn.example/runtime.js')",
    })).toEqual({
      ok: false,
      issues: [
        { code: "missing-entry-export", path: "exports.theme", message: "Theme entry must export theme" },
        { code: "bundled-host-runtime", path: "bundledModules.react", message: "Host runtime react must remain external" },
        { code: "bundled-host-runtime", path: "bundledModules.react-dom/client", message: "Host runtime react-dom/client must remain external" },
        { code: "unresolved-external", path: "externalImports.left-pad", message: "Dependency left-pad must be bundled into the theme" },
        { code: "remote-import", path: "externalImports.https://cdn.example/theme.js", message: "Remote executable import https://cdn.example/theme.js is forbidden" },
        { code: "second-react-root", path: "entrySource", message: "Theme entry must not create a React root" },
        { code: "remote-import", path: "entrySource", message: "Theme entry contains a remote dynamic import" },
      ],
    })
  })

  it("allows local keyframes inside the scoped theme layer", () => {
    expect(validateThemeBuild({
      ...validBuild,
      css: `${validBuild.css} @layer thism-theme { @keyframes pulse { from { opacity: 0.5 } to { opacity: 1 } } }`,
    })).toEqual({ ok: true, issues: [] })
  })

  it("rejects an unapproved portal surface", () => {
    expect(validateThemeBuild({
      ...validBuild,
      css: validBuild.css.replace('[data-thism-theme-portal="dialog"]', '[data-thism-theme-portal="login"]'),
    })).toEqual({
      ok: false,
      issues: [{ code: "unscoped-css", path: "css", message: 'Theme CSS selector [data-thism-theme-portal="login"] is outside approved theme boundaries' }],
    })
  })

  it("rejects CSS that escapes approved theme boundaries", () => {
    expect(validateThemeBuild({
      ...validBuild,
      css: `@import url("https://cdn.example/theme.css"); body { margin: 0 } .modal { color: red }`,
    })).toEqual({
      ok: false,
      issues: [
        { code: "css-import", path: "css", message: "Theme CSS imports are forbidden" },
        { code: "missing-css-layer", path: "css", message: "Theme CSS must use the thism-theme cascade layer" },
        { code: "unscoped-css", path: "css", message: "Theme CSS selector body is outside approved theme boundaries" },
        { code: "unscoped-css", path: "css", message: "Theme CSS selector .modal is outside approved theme boundaries" },
        { code: "missing-focus-visible", path: "css", message: "Theme CSS must define visible keyboard focus" },
        { code: "missing-reduced-motion", path: "css", message: "Theme CSS must honor prefers-reduced-motion" },
        { code: "missing-responsive-rules", path: "css", message: "Theme CSS must provide responsive navigation rules" },
      ],
    })
  })
})
