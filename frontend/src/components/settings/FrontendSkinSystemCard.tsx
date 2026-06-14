import { useCallback, useEffect, useMemo, useState, type ChangeEvent, type FormEvent } from "react"
import { CheckCircle2, DownloadCloud, Github, Layers3, RefreshCcw, ShieldCheck, Trash2, UploadCloud } from "lucide-react"

import { Badge } from "../ui/badge"
import { Button, buttonVariants } from "../ui/button"
import { Input } from "../ui/input"
import { useLanguage } from "../../i18n/language"
import { api, type FrontendSkin } from "../../lib/api"
import { cn } from "../../lib/utils"

const CLASSIC_SKIN_ID = "classic"

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
  const chunkSize = 0x8000
  for (let index = 0; index < bytes.length; index += chunkSize) {
    const chunk = bytes.subarray(index, index + chunkSize)
    binary += String.fromCharCode(...chunk)
  }
  return btoa(binary)
}

function skinDescription(skin: FrontendSkin, fallback: string) {
  return skin.description?.trim() || fallback
}

export function FrontendSkinSystemCard() {
  const { t } = useLanguage()
  const [skins, setSkins] = useState<FrontendSkin[]>([])
  const [activeSkinID, setActiveSkinID] = useState(CLASSIC_SKIN_ID)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState<string | null>(null)
  const [status, setStatus] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [repositoryUrl, setRepositoryUrl] = useState("")

  const activeSkin = useMemo(
    () => skins.find((skin) => skin.id === activeSkinID) ?? skins.find((skin) => skin.id === CLASSIC_SKIN_ID),
    [activeSkinID, skins],
  )
  const customSkinCount = skins.filter((skin) => skin.source === "custom").length

  const loadSkins = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await api.frontendSkins()
      setSkins(response.skins ?? [])
      setActiveSkinID(response.active_skin_id || CLASSIC_SKIN_ID)
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : t("settingsPage.frontendSkinsLoadFailed"))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void loadSkins()
  }, [loadSkins])

  const selectSkin = useCallback(async (id: string, name: string) => {
    setBusy(`select:${id}`)
    setError(null)
    setStatus(null)
    try {
      const response = await api.selectFrontendSkin(id)
      setActiveSkinID(response.active_skin_id)
      setStatus(t("settingsPage.frontendSkinSelected", { name }))
      await loadSkins()
    } catch (selectError) {
      setError(selectError instanceof Error ? selectError.message : t("settingsPage.frontendSkinSelectFailed"))
    } finally {
      setBusy(null)
    }
  }, [loadSkins, t])

  const handleArchiveUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    event.target.value = ""
    if (!file) return

    setBusy("archive")
    setError(null)
    setStatus(null)
    try {
      const response = await api.installFrontendSkinArchive(file.name, await fileToBase64(file))
      setActiveSkinID(response.active_skin_id)
      setStatus(t("settingsPage.frontendSkinInstalled", { name: response.skin.name }))
      await loadSkins()
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : t("settingsPage.frontendSkinInstallFailed"))
    } finally {
      setBusy(null)
    }
  }

  const handleRepositoryInstall = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setBusy("github")
    setError(null)
    setStatus(null)
    try {
      const response = await api.installFrontendSkinFromGitHub(repositoryUrl)
      setRepositoryUrl("")
      setActiveSkinID(response.active_skin_id)
      setStatus(t("settingsPage.frontendSkinGitHubInstalled", { name: response.skin.name }))
      await loadSkins()
    } catch (installError) {
      setError(installError instanceof Error ? installError.message : t("settingsPage.frontendSkinGitHubInstallFailed"))
    } finally {
      setBusy(null)
    }
  }

  const handleDeleteSkin = async (skin: FrontendSkin) => {
    setBusy(`delete:${skin.id}`)
    setError(null)
    setStatus(null)
    try {
      const response = await api.deleteFrontendSkin(skin.id)
      setActiveSkinID(response.active_skin_id)
      setStatus(t("settingsPage.frontendSkinRemoved", { name: skin.name }))
      await loadSkins()
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : t("settingsPage.frontendSkinRemoveFailed"))
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-200">{t("settingsPage.frontendSkinsTitle")}</h3>
          <p className="mt-1 max-w-3xl text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.frontendSkinsDescription")}</p>
        </div>
        <Badge variant="outline" className="w-fit border-border bg-card text-muted-foreground">
          {t("settingsPage.frontendSkinsInstalledCount", { count: skins.length })}
        </Badge>
      </div>

      <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
        <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.55fr)]">
          <div className="space-y-4">
            <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
              <form className="rounded-2xl border border-border bg-card/55 p-4" onSubmit={handleRepositoryInstall}>
                <div className="grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
                  <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                    {t("settingsPage.frontendSkinGitHubRepository")}
                    <Input
                      type="url"
                      value={repositoryUrl}
                      onChange={(event) => setRepositoryUrl(event.target.value)}
                      placeholder="https://github.com/owner/thism-skin"
                      aria-label={t("settingsPage.frontendSkinGitHubRepository")}
                      className="enterprise-outline-control mt-2 h-11 rounded-xl border"
                    />
                  </label>
                  <Button
                    type="submit"
                    variant="outline"
                    className="enterprise-outline-control h-11 rounded-xl px-4"
                    disabled={busy === "github" || repositoryUrl.trim() === ""}
                  >
                    {busy === "github" ? <DownloadCloud className="h-4 w-4 animate-pulse" aria-hidden /> : <Github className="h-4 w-4" aria-hidden />}
                    {busy === "github" ? t("settingsPage.frontendSkinInstalling") : t("settingsPage.frontendSkinGitHubInstall")}
                  </Button>
                </div>
              </form>

              <div className="flex flex-wrap gap-2">
                <label
                  htmlFor="frontend-skin-upload"
                  className={cn(buttonVariants({ variant: "outline", size: "default" }), "enterprise-outline-control h-11 cursor-pointer rounded-xl px-4")}
                >
                  <UploadCloud className="h-4 w-4" aria-hidden />
                  {t("settingsPage.frontendSkinUploadFile")}
                </label>
                <input
                  id="frontend-skin-upload"
                  type="file"
                  accept="application/zip,.zip,.thism-frontend-skin.zip"
                  aria-label={t("settingsPage.frontendSkinUploadFile")}
                  className="sr-only"
                  onChange={handleArchiveUpload}
                />
              </div>
            </div>

            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
            {status && <p role="status" className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{status}</p>}

            {loading ? (
              <div className="rounded-2xl border border-border bg-card/70 px-4 py-6 text-sm text-muted-foreground">
                {t("settingsPage.frontendSkinsLoading")}
              </div>
            ) : (
              <div role="list" aria-label={t("settingsPage.frontendSkinsInstalled")} className="divide-y divide-border overflow-hidden rounded-2xl border border-border bg-card/70">
                {skins.map((skin) => {
                  const active = skin.id === activeSkinID
                  return (
                    <div key={skin.id} role="listitem" className="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center sm:justify-between">
                      <div className="flex min-w-0 items-start gap-3">
                        <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border bg-background text-muted-foreground">
                          {skin.source === "built-in" ? <ShieldCheck className="h-4 w-4" aria-hidden /> : <Layers3 className="h-4 w-4" aria-hidden />}
                        </span>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="truncate text-sm font-semibold text-foreground">{skin.name}</p>
                            {active && (
                              <Badge className="gap-1 bg-primary text-primary-foreground">
                                <CheckCircle2 className="h-3 w-3" aria-hidden />
                                {t("settingsPage.themeActiveBadge")}
                              </Badge>
                            )}
                            <Badge variant="outline" className="border-border text-muted-foreground">
                              {skin.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeSourceBuiltIn")}
                            </Badge>
                          </div>
                          <p className="mt-1 text-xs text-muted-foreground">
                            {skinDescription(skin, skin.source === "built-in" ? t("settingsPage.frontendSkinBuiltInDescription") : t("settingsPage.frontendSkinCustomDescription"))}
                          </p>
                          <p className="mt-1 font-mono text-[11px] text-muted-foreground">{skin.api_version} / {skin.entry}</p>
                        </div>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        {!active && (
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            className="enterprise-outline-control rounded-xl"
                            disabled={busy === `select:${skin.id}`}
                            onClick={() => void selectSkin(skin.id, skin.name)}
                          >
                            <RefreshCcw className="h-4 w-4" aria-hidden />
                            {t("settingsPage.frontendSkinUseSkin")}
                          </Button>
                        )}
                        {skin.source === "custom" && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            aria-label={t("settingsPage.frontendSkinRemoveSkin", { name: skin.name })}
                            className="h-9 w-9 rounded-xl text-muted-foreground hover:text-destructive"
                            disabled={busy === `delete:${skin.id}`}
                            onClick={() => void handleDeleteSkin(skin)}
                          >
                            <Trash2 className="h-4 w-4" aria-hidden />
                          </Button>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          <aside className="rounded-2xl border border-border bg-card/55 p-4">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-xs font-medium uppercase tracking-[0.16em] text-muted-foreground">{t("settingsPage.frontendSkinRuntimeLabel")}</p>
                <p className="mt-2 truncate text-sm font-semibold text-foreground">{activeSkin?.name ?? t("settingsPage.frontendSkinsLoading")}</p>
              </div>
              <Layers3 className="h-4 w-4 text-muted-foreground" aria-hidden />
            </div>
            <div className="mt-4 grid gap-2 text-xs text-muted-foreground">
              <div className="flex items-center justify-between rounded-xl border border-border bg-background px-3 py-2">
                <span>{t("settingsPage.frontendSkinRuntimeMode")}</span>
                <span className="font-medium text-foreground">{activeSkin?.source === "custom" ? t("settingsPage.themeSourceCustom") : t("settingsPage.themeSourceBuiltIn")}</span>
              </div>
              <div className="flex items-center justify-between rounded-xl border border-border bg-background px-3 py-2">
                <span>{t("settingsPage.frontendSkinCustomCount")}</span>
                <span className="font-medium text-foreground">{customSkinCount}</span>
              </div>
              <a className={cn(buttonVariants({ variant: "outline", size: "sm" }), "enterprise-outline-control mt-2 rounded-xl")} href="/">
                <RefreshCcw className="h-4 w-4" aria-hidden />
                {t("settingsPage.frontendSkinOpenActive")}
              </a>
            </div>
          </aside>
        </div>
      </section>
    </div>
  )
}
