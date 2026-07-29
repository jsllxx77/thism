const HOST_EXTERNALS = ["react", "react-dom", "react/jsx-runtime", "@thism/theme-sdk"] as const
const APPROVED_PORTAL_SELECTORS = ["dialog", "sheet", "menu", "toast"].map((name) => `[data-thism-theme-portal="${name}"]`)
const THEME_SCOPE_SELECTORS = ["[data-thism-theme-root]", ...APPROVED_PORTAL_SELECTORS] as const

export type ThemeBuildIssueCode =
  | "missing-entry-export"
  | "bundled-host-runtime"
  | "unresolved-external"
  | "remote-import"
  | "second-react-root"
  | "css-import"
  | "missing-css-layer"
  | "unscoped-css"
  | "missing-focus-visible"
  | "missing-reduced-motion"
  | "missing-responsive-rules"

export interface ThemeBuildIssue {
  code: ThemeBuildIssueCode
  path: string
  message: string
}

export interface ThemeBuildInput {
  exports: readonly string[]
  bundledModules: readonly string[]
  externalImports: readonly string[]
  entrySource: string
  css: string
}

export type ThemeBuildResult =
  | { ok: true; issues: [] }
  | { ok: false; issues: ThemeBuildIssue[] }

function result(issues: ThemeBuildIssue[]): ThemeBuildResult {
  return issues.length === 0 ? { ok: true, issues: [] } : { ok: false, issues }
}

function isHostRuntime(moduleName: string) {
  return moduleName === "react" || moduleName.startsWith("react/") || moduleName === "react-dom" || moduleName.startsWith("react-dom/")
}

function isRemoteUrl(value: string) {
  return /^(?:https?:)?\/\//i.test(value)
}

function validateModules(input: ThemeBuildInput, issues: ThemeBuildIssue[]) {
  if (!input.exports.includes("theme")) {
    issues.push({ code: "missing-entry-export", path: "exports.theme", message: "Theme entry must export theme" })
  }

  for (const moduleName of input.bundledModules) {
    if (isHostRuntime(moduleName)) {
      issues.push({ code: "bundled-host-runtime", path: `bundledModules.${moduleName}`, message: `Host runtime ${moduleName} must remain external` })
    }
  }

  for (const moduleName of input.externalImports) {
    if (isRemoteUrl(moduleName)) {
      issues.push({ code: "remote-import", path: `externalImports.${moduleName}`, message: `Remote executable import ${moduleName} is forbidden` })
    } else if (!(HOST_EXTERNALS as readonly string[]).includes(moduleName)) {
      issues.push({ code: "unresolved-external", path: `externalImports.${moduleName}`, message: `Dependency ${moduleName} must be bundled into the theme` })
    }
  }

  if (/\b(?:createRoot|hydrateRoot)\s*\(/.test(input.entrySource)) {
    issues.push({ code: "second-react-root", path: "entrySource", message: "Theme entry must not create a React root" })
  }
  if (/\bimport\s*\(\s*["'](?:https?:)?\/\//i.test(input.entrySource)) {
    issues.push({ code: "remote-import", path: "entrySource", message: "Theme entry contains a remote dynamic import" })
  }
}

function topLevelRuleSelectors(css: string) {
  const selectors: string[] = []
  let depth = 0
  let start = 0
  for (let index = 0; index < css.length; index += 1) {
    if (css[index] === "{") {
      const prelude = css.slice(start, index).trim()
      if (depth === 0 && prelude && !prelude.startsWith("@")) selectors.push(...prelude.split(",").map((value) => value.trim()))
      depth += 1
      start = index + 1
    } else if (css[index] === "}") {
      depth = Math.max(0, depth - 1)
      start = index + 1
    } else if (css[index] === ";" && depth === 0) {
      start = index + 1
    }
  }
  return selectors
}

function allRuleSelectors(css: string) {
  const selectors: string[] = []
  const contexts: Array<"rules" | "keyframes"> = []
  let start = 0

  for (let index = 0; index < css.length; index += 1) {
    if (css[index] === "{") {
      const prelude = css.slice(start, index).trim()
      const insideKeyframes = contexts.at(-1) === "keyframes"
      if (!insideKeyframes && prelude && !prelude.startsWith("@")) {
        selectors.push(...prelude.split(",").map((selector) => selector.trim()))
      }
      contexts.push(insideKeyframes || /^@(?:-webkit-)?keyframes\b/i.test(prelude) ? "keyframes" : "rules")
      start = index + 1
    } else if (css[index] === "}") {
      contexts.pop()
      start = index + 1
    } else if (css[index] === ";") {
      start = index + 1
    }
  }

  return selectors
}

function validateCss(css: string, issues: ThemeBuildIssue[]) {
  if (/@import\b/i.test(css)) {
    issues.push({ code: "css-import", path: "css", message: "Theme CSS imports are forbidden" })
  }
  if (!/@layer\s+thism-theme\b/.test(css)) {
    issues.push({ code: "missing-css-layer", path: "css", message: "Theme CSS must use the thism-theme cascade layer" })
  }

  const selectors = /@layer\s+thism-theme\s*\{/.test(css) ? allRuleSelectors(css) : topLevelRuleSelectors(css)
  for (const selector of selectors) {
    if (!THEME_SCOPE_SELECTORS.some((scope) => selector.includes(scope))) {
      issues.push({ code: "unscoped-css", path: "css", message: `Theme CSS selector ${selector} is outside approved theme boundaries` })
    }
  }

  if (!/:focus-visible\b/.test(css)) {
    issues.push({ code: "missing-focus-visible", path: "css", message: "Theme CSS must define visible keyboard focus" })
  }
  if (!/@media\s*\([^)]*prefers-reduced-motion\s*:\s*reduce/i.test(css)) {
    issues.push({ code: "missing-reduced-motion", path: "css", message: "Theme CSS must honor prefers-reduced-motion" })
  }
  if (!/@media\s*\([^)]*(?:max-width|min-width)/i.test(css)) {
    issues.push({ code: "missing-responsive-rules", path: "css", message: "Theme CSS must provide responsive navigation rules" })
  }
}

export function validateThemeBuild(input: ThemeBuildInput): ThemeBuildResult {
  const issues: ThemeBuildIssue[] = []
  validateModules(input, issues)
  validateCss(input.css, issues)
  return result(issues)
}
