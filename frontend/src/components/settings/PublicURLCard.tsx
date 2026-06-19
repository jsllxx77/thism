import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api } from "../../lib/api"
import { Button } from "../ui/button"
import { Input } from "../ui/input"

export function PublicURLCard() {
  const { t, translateApiError } = useLanguage()
  const [publicURL, setPublicURL] = useState("")
  const [savedPublicURL, setSavedPublicURL] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await api.publicURLSettings()
        if (cancelled) {
          return
        }
        const nextValue = response.public_url ?? ""
        setPublicURL(nextValue)
        setSavedPublicURL(nextValue)
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.publicURLUpdateFailed"))
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [t])

  const hasChanges = savedPublicURL !== null && publicURL !== savedPublicURL
  const statusLabel = useMemo(
    () => (publicURL.trim() === "" ? t("settingsPage.publicURLAutoDetect") : t("settingsPage.publicURLConfigured")),
    [publicURL, t],
  )

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)

    try {
      const response = await api.updatePublicURLSettings({ public_url: publicURL })
      const nextValue = response.public_url ?? ""
      setPublicURL(nextValue)
      setSavedPublicURL(nextValue)
      setSuccess(t("settingsPage.publicURLSaved"))
    } catch (saveError) {
      const message = saveError instanceof Error ? translateApiError(saveError.message) : t("settingsPage.publicURLUpdateFailed")
      setError(message)
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.publicURLTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.publicURLDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
          {statusLabel}
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : (
        <form className="mt-4 space-y-4" onSubmit={handleSubmit}>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
            {t("settingsPage.publicURLFieldLabel")}
            <Input
              type="url"
              inputMode="url"
              autoComplete="url"
              placeholder="https://thism.example.com"
              aria-label={t("settingsPage.publicURLFieldLabel")}
              value={publicURL}
              onChange={(event) => {
                setPublicURL(event.target.value)
                setError(null)
                setSuccess(null)
              }}
              className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
            />
          </label>

          <p className="text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.publicURLHint")}</p>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              disabled={saving || !hasChanges}
              className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
            >
              {saving ? t("settingsPage.publicURLSaving") : t("settingsPage.publicURLSave")}
            </Button>
            {success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
          </div>
        </form>
      )}
    </section>
  )
}
