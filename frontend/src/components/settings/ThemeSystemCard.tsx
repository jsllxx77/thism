import { useMemo, useState, type ChangeEvent, type FormEvent } from "react"
import { Archive, CheckCircle2, Github, Loader2, UploadCloud } from "lucide-react"

import { Badge } from "../ui/badge"
import { Button, buttonVariants } from "../ui/button"
import { Input } from "../ui/input"
import { useLanguage } from "../../i18n/language"
import { cn } from "../../lib/utils"
import { api } from "../../lib/api"
import { type AppThemeDefinition, useAppTheme } from "../../theme/theme-context"
import { loadThemePackageFromGitHub } from "../../theme/theme-repository"

function getThemeLabel(theme: AppThemeDefinition, labels: Record<string, string>) {
  return theme.source === "built-in" ? labels[theme.labelKey] : theme.label
}

function getThemeDescription(theme: AppThemeDefinition, sourceLabel: string) {
  return theme.source === "custom" ? theme.description || sourceLabel : sourceLabel
}

async function fileToBase64(file: File) {
  if (typeof file.arrayBuffer !== "function") {
    return new Promise<string>((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = () => {
        const value = String(reader.result ?? "")
        const marker = "base64,"
        const index = value.indexOf(marker)
        resolve(index >= 0 ? value.slice(index + marker.length) : value)
      }
      reader.onerror = () => reject(reader.error)
      reader.readAsDataURL(file)
    })
  }

  const bytes = new Uint8Array(await file.arrayBuffer())
  let binary = ""
  for (let index = 0; index < bytes.length; index += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000))
  }
  return btoa(binary)
}

export function ThemeSystemCard() {
  const { messages, t } = useLanguage()
  const { theme, themes, importThemePackage } = useAppTheme()
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [repositoryUrl, setRepositoryUrl] = useState("")
  const [importingRepository, setImportingRepository] = useState(false)
  const [importingPackage, setImportingPackage] = useState(false)
  const activeTheme = useMemo(() => themes.find((option) => option.name === theme) ?? themes[0], [theme, themes])
  const activeThemeLabel = getThemeLabel(activeTheme, messages.shell.themeLabels)
  const customThemeCount = themes.filter((option) => option.source === "custom").length

  const resetFeedback = () => {
    setError(null)
    setStatus(null)
  }

  const importArchive = async (filename: string, data: string) => {
    const response = await api.importThemeArchive(filename, data)
    const imported = importThemePackage(JSON.stringify(response.theme))
    setStatus(t("settingsPage.themeImportSuccess", { name: imported.label }))
  }

  const handleThemeUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ""
    if (!file) return

    resetFeedback()
    if (!file.name.toLowerCase().endsWith(".zip")) {
      setError(t("settingsPage.themeArchiveOnly"))
      return
    }

    setImportingPackage(true)
    try {
      await importArchive(file.name, await fileToBase64(file))
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : t("settingsPage.themeImportFailed"))
    } finally {
      setImportingPackage(false)
    }
  }

  const handleRepositoryImport = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    resetFeedback()
    setImportingRepository(true)

    try {
      const archive = await loadThemePackageFromGitHub(repositoryUrl)
      await importArchive(archive.filename, archive.data)
      setRepositoryUrl("")
    } catch (repositoryError) {
      setError(repositoryError instanceof Error ? repositoryError.message : t("settingsPage.themeGitHubImportFailed"))
    } finally {
      setImportingRepository(false)
    }
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.themeSystemTitle")}</h3>
          <p className="mt-1 max-w-3xl text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.themeSystemDescription")}</p>
        </div>
        <Badge variant="outline" className="w-fit border-border bg-card text-muted-foreground">
          {t("settingsPage.themeInstalledCount", { count: themes.length })}
        </Badge>
      </div>

      <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5 sm:px-6">
        <div className="space-y-6">
          <div className="rounded-2xl border border-border bg-card/65 p-4 sm:p-5">
            <div className="flex items-start gap-3">
              <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-primary">
                <Archive className="h-4 w-4" aria-hidden />
              </span>
              <div className="min-w-0">
                <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{t("settingsPage.themeActiveTheme")}</p>
                <p className="mt-1 truncate text-base font-semibold text-foreground">{activeThemeLabel}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {getThemeDescription(activeTheme, activeTheme.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeBuiltInDescription"))}
                </p>
              </div>
              <Badge className="ml-auto shrink-0 gap-1 bg-primary text-primary-foreground">
                <CheckCircle2 className="h-3 w-3" aria-hidden />
                {t("settingsPage.themeActiveBadge")}
              </Badge>
            </div>
          </div>

          <div>
            <div className="mb-3 flex items-center justify-between gap-3">
              <div>
                <p className="text-sm font-semibold text-foreground">{t("settingsPage.themeInstalledThemes")}</p>
                <p className="mt-1 text-xs text-muted-foreground">{t("settingsPage.themeListDescription")}</p>
              </div>
            </div>
            <div role="list" aria-label={t("settingsPage.themeInstalledThemes")} className="divide-y divide-border overflow-hidden rounded-2xl border border-border bg-card/70">
              {themes.map((option) => {
                const label = getThemeLabel(option, messages.shell.themeLabels)
                const active = option.name === theme

                return (
                  <div key={option.name} role="listitem" className="flex items-start gap-3 px-4 py-3.5">
                    <span className="mt-1 h-3.5 w-3.5 shrink-0 rounded-full border border-border shadow-sm" style={{ backgroundColor: option.accent }} aria-hidden />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="truncate text-sm font-semibold text-foreground">{label}</p>
                        {active && <Badge className="gap-1 bg-primary text-primary-foreground"><CheckCircle2 className="h-3 w-3" aria-hidden />{t("settingsPage.themeActiveBadge")}</Badge>}
                        <Badge variant="outline" className="border-border text-muted-foreground">
                          {option.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeSourceBuiltIn")}
                        </Badge>
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {getThemeDescription(option, option.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeBuiltInDescription"))}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
            {customThemeCount > 0 && <p className="mt-2 text-[11px] text-muted-foreground">{t("settingsPage.themeImportOnlyHint")}</p>}
          </div>

          <div className="grid gap-4 border-t border-border/70 pt-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
            <form className="rounded-2xl border border-border bg-card/55 p-4" onSubmit={handleRepositoryImport}>
              <label className="block text-xs font-medium text-slate-600 dark:text-slate-300" htmlFor="theme-github-repository">
                {t("settingsPage.themeGitHubRepository")}
                <Input
                  id="theme-github-repository"
                  type="url"
                  value={repositoryUrl}
                  onChange={(event) => setRepositoryUrl(event.target.value)}
                  placeholder="https://github.com/owner/theme-repository"
                  aria-describedby="theme-github-repository-hint"
                  className="enterprise-outline-control mt-2 h-11 rounded-xl border"
                />
              </label>
              <p id="theme-github-repository-hint" className="mt-2 text-[11px] text-muted-foreground">{t("settingsPage.themeGitHubReleaseHint")}</p>
              <Button
                type="submit"
                variant="outline"
                className="enterprise-outline-control mt-3 h-11 rounded-xl px-4"
                disabled={importingRepository || importingPackage || repositoryUrl.trim() === ""}
              >
                {importingRepository ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Github className="h-4 w-4" aria-hidden />}
                {importingRepository ? t("settingsPage.themeGitHubInstalling") : t("settingsPage.themeGitHubInstall")}
              </Button>
            </form>

            <label
              htmlFor="theme-package-upload"
              className={cn(buttonVariants({ variant: "outline", size: "default" }), "enterprise-outline-control inline-flex h-11 cursor-pointer rounded-xl px-4")}
            >
              {importingPackage ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <UploadCloud className="h-4 w-4" aria-hidden />}
              {importingPackage ? t("settingsPage.themePackageInstalling") : t("settingsPage.themeUploadArchive")}
              <input id="theme-package-upload" type="file" accept="application/zip,.zip,.thism-theme.zip" aria-label={t("settingsPage.themeUploadArchive")} className="sr-only" onChange={handleThemeUpload} disabled={importingPackage || importingRepository} />
            </label>
          </div>

          {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
          {status && <p role="status" className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{status}</p>}
        </div>
      </section>
    </div>
  )
}
