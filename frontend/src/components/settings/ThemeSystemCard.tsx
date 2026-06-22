import { useMemo, useRef, useState, type ChangeEvent, type FormEvent } from "react"
import { CheckCircle2, FileJson, Github, Loader2, Palette, Trash2, UploadCloud } from "lucide-react"

import { Badge } from "../ui/badge"
import { Button, buttonVariants } from "../ui/button"
import { Input } from "../ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../ui/select"
import { useLanguage } from "../../i18n/language"
import { cn } from "../../lib/utils"
import { type AppThemeDefinition, type AppThemeName, useAppTheme } from "../../theme/theme-context"
import { loadThemePackageFromGitHub } from "../../theme/theme-repository"

function getThemeLabel(theme: AppThemeDefinition, labels: Record<string, string>) {
  return theme.source === "built-in" ? labels[theme.labelKey] : theme.label
}

function getThemeDescription(theme: AppThemeDefinition, sourceLabel: string) {
  if (theme.source === "custom") {
    return theme.description || sourceLabel
  }
  return sourceLabel
}

async function readThemeFile(file: File) {
  if (typeof file.text === "function") {
    return file.text()
  }

  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ""))
    reader.onerror = () => reject(reader.error)
    reader.readAsText(file)
  })
}

export function ThemeSystemCard() {
  const { messages, t } = useLanguage()
  const { theme, setTheme, themes, importThemePackage, removeTheme } = useAppTheme()
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [repositoryUrl, setRepositoryUrl] = useState("")
  const [importingRepository, setImportingRepository] = useState(false)
  const inputRef = useRef<HTMLInputElement | null>(null)
  const activeTheme = useMemo(() => themes.find((option) => option.name === theme) ?? themes[0], [theme, themes])
  const activeThemeLabel = getThemeLabel(activeTheme, messages.shell.themePicker)
  const customThemeCount = themes.filter((option) => option.source === "custom").length

  const handleThemeChange = (value: string) => {
    setError(null)
    setStatus(null)
    setTheme(value as AppThemeName)
  }

  const handleThemeUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ""
    if (!file) return

    setError(null)
    setStatus(null)

    try {
      const imported = importThemePackage(await readThemeFile(file))
      setStatus(t("settingsPage.themeImportSuccess", { name: imported.label }))
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : t("settingsPage.themeImportFailed"))
    }
  }

  const handleRepositoryImport = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    setStatus(null)
    setImportingRepository(true)

    try {
      const imported = importThemePackage(await loadThemePackageFromGitHub(repositoryUrl))
      setRepositoryUrl("")
      setStatus(t("settingsPage.themeGitHubImportSuccess", { name: imported.label }))
    } catch (repositoryError) {
      setError(repositoryError instanceof Error ? repositoryError.message : t("settingsPage.themeGitHubImportFailed"))
    } finally {
      setImportingRepository(false)
    }
  }

  const handleRemoveTheme = (themeName: AppThemeName) => {
    const removingTheme = themes.find((option) => option.name === themeName)
    removeTheme(themeName)
    setStatus(t("settingsPage.themeRemoved", { name: removingTheme ? getThemeLabel(removingTheme, messages.shell.themePicker) : "" }))
    setError(null)
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.themeSystemTitle")}</h3>
          <p className="mt-1 max-w-3xl text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.themeSystemDescription")}</p>
        </div>
        <Badge variant="outline" className="w-fit border-border bg-card text-muted-foreground">
          {t("settingsPage.themeInstalledCount", { count: themes.length })}
        </Badge>
      </div>

      <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.72fr)]">
          <div className="space-y-5">
            <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                {t("settingsPage.themeActiveTheme")}
                <Select value={theme} onValueChange={handleThemeChange}>
                  <SelectTrigger
                    aria-label={t("settingsPage.themeActiveTheme")}
                    className="enterprise-outline-control mt-2 h-11 rounded-xl border bg-background text-foreground"
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <span
                        className="h-3 w-3 shrink-0 rounded-full border border-border"
                        style={{ backgroundColor: activeTheme.accent }}
                        aria-hidden
                      />
                      <SelectValue>{activeThemeLabel}</SelectValue>
                    </span>
                  </SelectTrigger>
                  <SelectContent align="start">
                    {themes.map((option) => (
                      <SelectItem key={option.name} value={option.name}>
                        <span className="flex items-center gap-2">
                          <span
                            className="h-2.5 w-2.5 rounded-full border border-border"
                            style={{ backgroundColor: option.accent }}
                            aria-hidden
                          />
                          {getThemeLabel(option, messages.shell.themePicker)}
                        </span>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </label>

              <div className="flex flex-wrap gap-2">
                <label
                  htmlFor="theme-package-upload"
                  className={cn(buttonVariants({ variant: "outline", size: "default" }), "enterprise-outline-control h-11 cursor-pointer rounded-xl px-4")}
                >
                  <UploadCloud className="h-4 w-4" aria-hidden />
                  {t("settingsPage.themeUploadFile")}
                </label>
                <input
                  ref={inputRef}
                  id="theme-package-upload"
                  type="file"
                  accept="application/json,.json,.thism-theme.json"
                  aria-label={t("settingsPage.themeUploadFile")}
                  className="sr-only"
                  onChange={handleThemeUpload}
                />
              </div>
            </div>

            <form className="rounded-2xl border border-border bg-card/55 p-4" onSubmit={handleRepositoryImport}>
              <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("settingsPage.themeGitHubRepository")}
                  <Input
                    type="url"
                    value={repositoryUrl}
                    onChange={(event) => setRepositoryUrl(event.target.value)}
                    placeholder="https://github.com/owner/repo"
                    aria-label={t("settingsPage.themeGitHubRepository")}
                    className="enterprise-outline-control mt-2 h-11 rounded-xl border"
                  />
                </label>
                <Button
                  type="submit"
                  variant="outline"
                  className="enterprise-outline-control h-11 rounded-xl px-4"
                  disabled={importingRepository || repositoryUrl.trim() === ""}
                >
                  {importingRepository ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Github className="h-4 w-4" aria-hidden />}
                  {importingRepository ? t("settingsPage.themeGitHubInstalling") : t("settingsPage.themeGitHubInstall")}
                </Button>
              </div>
            </form>

            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
            {status && <p role="status" className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{status}</p>}

            <div role="list" aria-label={t("settingsPage.themeInstalledThemes")} className="divide-y divide-border overflow-hidden rounded-2xl border border-border bg-card/70">
              {themes.map((option) => {
                const label = getThemeLabel(option, messages.shell.themePicker)
                const active = option.name === theme

                return (
                  <div key={option.name} role="listitem" className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
                    <div className="flex min-w-0 items-start gap-3">
                      <span
                        className="mt-1 h-3.5 w-3.5 shrink-0 rounded-full border border-border shadow-sm"
                        style={{ backgroundColor: option.accent }}
                        aria-hidden
                      />
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="truncate text-sm font-semibold text-foreground">{label}</p>
                          {active && (
                            <Badge className="gap-1 bg-primary text-primary-foreground">
                              <CheckCircle2 className="h-3 w-3" aria-hidden />
                              {t("settingsPage.themeActiveBadge")}
                            </Badge>
                          )}
                          <Badge variant="outline" className="border-border text-muted-foreground">
                            {option.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeSourceBuiltIn")}
                          </Badge>
                        </div>
                        <p className="mt-1 text-xs text-muted-foreground">
                          {getThemeDescription(option, option.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeBuiltInDescription"))}
                        </p>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {option.name !== theme && (
                        <Button type="button" variant="outline" size="sm" className="enterprise-outline-control rounded-xl" onClick={() => setTheme(option.name)}>
                          <Palette className="h-4 w-4" aria-hidden />
                          {t("settingsPage.themeUseTheme")}
                        </Button>
                      )}
                      {option.source === "custom" && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t("settingsPage.themeRemoveTheme", { name: label })}
                          className="h-9 w-9 rounded-xl text-muted-foreground hover:text-destructive"
                          onClick={() => handleRemoveTheme(option.name)}
                        >
                          <Trash2 className="h-4 w-4" aria-hidden />
                        </Button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>

          <aside className="theme-preview-canvas rounded-2xl border border-border p-4">
            <div className="mb-4 flex items-center justify-between gap-3">
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-foreground">{activeThemeLabel}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {activeTheme.source === "custom"
                    ? t("settingsPage.themePreviewCustom", { count: customThemeCount })
                    : t("settingsPage.themePreviewBuiltIn")}
                </p>
              </div>
              <FileJson className="h-4 w-4 text-muted-foreground" aria-hidden />
            </div>
            <div className="theme-preview-window rounded-xl border border-border bg-card p-3">
              <div className="mb-3 flex gap-1.5">
                <span className="h-2.5 w-2.5 rounded-full bg-primary" />
                <span className="h-2.5 w-2.5 rounded-full bg-accent" />
                <span className="h-2.5 w-2.5 rounded-full bg-muted" />
              </div>
              <div className="space-y-2">
                <div className="h-2 w-3/4 rounded-full bg-foreground/70" />
                <div className="h-2 w-1/2 rounded-full bg-muted-foreground/50" />
                <div className="mt-3 grid grid-cols-3 gap-2">
                  <span className="h-12 rounded-lg bg-primary/90" />
                  <span className="h-12 rounded-lg bg-accent" />
                  <span className="h-12 rounded-lg bg-muted" />
                </div>
                <div className="mt-3 space-y-1.5">
                  <span className="block h-7 rounded-md border border-border bg-background" />
                  <span className="block h-7 rounded-md border border-border bg-background" />
                </div>
              </div>
            </div>
          </aside>
        </div>
      </section>
    </div>
  )
}
