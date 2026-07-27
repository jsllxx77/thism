import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type GeoIPSettings } from "../../lib/api"
import { Button } from "../ui/button"
import { Input } from "../ui/input"

function formatBytes(bytes: number | undefined) {
  if (!bytes || bytes <= 0) return "-"
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function formatTime(unix: number | undefined) {
  if (!unix) return "-"
  try {
    return new Date(unix * 1000).toLocaleString()
  } catch {
    return "-"
  }
}

export function GeoIPCard() {
  const { t, translateApiError } = useLanguage()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [updating, setUpdating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [settings, setSettings] = useState<GeoIPSettings | null>(null)
  const [provider, setProvider] = useState<"maxmind" | "ip2location">("maxmind")
  const [ip2Token, setIp2Token] = useState("")
  const [maxmindKey, setMaxmindKey] = useState("")

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await api.geoIPSettings()
      setSettings(response)
      setProvider(response.provider === "ip2location" ? "ip2location" : "maxmind")
      setIp2Token("")
      setMaxmindKey("")
    } catch (err) {
      setError(err instanceof Error ? translateApiError(err.message) : t("settingsPage.geoipLoadFailed"))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
  }, [t])

  const statusLabel = useMemo(() => {
    if (!settings) return t("Loading")
    if (settings.enabled) return t("settingsPage.geoipStatusEnabled")
    return t("settingsPage.geoipStatusDisabled")
  }, [settings, t])

  const handleSave = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      const response = await api.updateGeoIPSettings({
        provider,
        ip2location_token: ip2Token.trim() || undefined,
        maxmind_license_key: maxmindKey.trim() || undefined,
      })
      setSettings(response)
      setProvider(response.provider === "ip2location" ? "ip2location" : "maxmind")
      setIp2Token("")
      setMaxmindKey("")
      setSuccess(t("settingsPage.geoipSaved"))
    } catch (err) {
      setError(err instanceof Error ? translateApiError(err.message) : t("settingsPage.geoipSaveFailed"))
    } finally {
      setSaving(false)
    }
  }

  const handleUpdateDatabase = async () => {
    setUpdating(true)
    setError(null)
    setSuccess(null)
    try {
      // Persist current form values first so update uses the latest token/key.
      const saved = await api.updateGeoIPSettings({
        provider,
        ip2location_token: ip2Token.trim() || undefined,
        maxmind_license_key: maxmindKey.trim() || undefined,
      })
      setSettings(saved)
      const response = await api.updateGeoIPDatabase()
      setSettings(response)
      setIp2Token("")
      setMaxmindKey("")
      setSuccess(t("settingsPage.geoipUpdated"))
    } catch (err) {
      setError(err instanceof Error ? translateApiError(err.message) : t("settingsPage.geoipUpdateFailed"))
      void load()
    } finally {
      setUpdating(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.geoipTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.geoipDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
          {statusLabel}
        </span>
      </div>

      {loading || !settings ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : (
        <form className="mt-4 space-y-4" onSubmit={handleSave}>
          <fieldset>
            <legend className="mb-2 text-xs font-medium uppercase tracking-[0.12em] text-slate-500 dark:text-slate-400">
              {t("settingsPage.geoipProvider")}
            </legend>
            <div className="grid gap-3 md:grid-cols-2">
              {(
                [
                  { id: "maxmind" as const, label: t("settingsPage.geoipProviderMaxMind"), hint: t("settingsPage.geoipProviderMaxMindHint") },
                  { id: "ip2location" as const, label: t("settingsPage.geoipProviderIP2Location"), hint: t("settingsPage.geoipProviderIP2LocationHint") },
                ] as const
              ).map((option) => {
                const checked = provider === option.id
                return (
                  <label
                    key={option.id}
                    className={`motion-choice-card enterprise-inner-surface flex min-h-11 cursor-pointer flex-col rounded-2xl border px-4 py-3 ${
                      checked
                        ? "border-slate-300 bg-slate-50 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50"
                        : "border-slate-200 bg-white/80 text-slate-600 hover:bg-slate-50 dark:border-white/8 dark:bg-slate-950/80 dark:text-slate-200 dark:hover:bg-slate-900"
                    }`}
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div>
                        <input
                          type="radio"
                          name="geoip-provider"
                          className="sr-only"
                          checked={checked}
                          onChange={() => {
                            setProvider(option.id)
                            setError(null)
                            setSuccess(null)
                          }}
                        />
                        <span className="text-sm font-medium">{option.label}</span>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{option.hint}</p>
                      </div>
                      <span
                        aria-hidden="true"
                        className={`flex h-4 w-4 items-center justify-center rounded-full border ${
                          checked ? "border-slate-900 dark:border-slate-100" : "border-slate-300 dark:border-slate-600"
                        }`}
                      >
                        {checked && <span className="h-2 w-2 rounded-full bg-slate-900 dark:bg-slate-100" />}
                      </span>
                    </div>
                  </label>
                )
              })}
            </div>
          </fieldset>

          {provider === "ip2location" ? (
            <div className="space-y-2">
              <label className="text-xs font-medium text-slate-600 dark:text-slate-300" htmlFor="geoip-ip2-token">
                {t("settingsPage.geoipIP2LocationToken")}
              </label>
              <Input
                id="geoip-ip2-token"
                type="password"
                autoComplete="off"
                value={ip2Token}
                placeholder={
                  settings.ip2location_token_set
                    ? t("settingsPage.geoipTokenConfiguredPlaceholder")
                    : t("settingsPage.geoipTokenPlaceholder")
                }
                onChange={(event) => setIp2Token(event.target.value)}
              />
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {settings.ip2location_token_set
                  ? t("settingsPage.geoipTokenConfiguredHint")
                  : t("settingsPage.geoipIP2LocationTokenHint")}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              <label className="text-xs font-medium text-slate-600 dark:text-slate-300" htmlFor="geoip-maxmind-key">
                {t("settingsPage.geoipMaxMindLicenseKey")}
              </label>
              <Input
                id="geoip-maxmind-key"
                type="password"
                autoComplete="off"
                value={maxmindKey}
                placeholder={
                  settings.maxmind_license_key_set
                    ? t("settingsPage.geoipTokenConfiguredPlaceholder")
                    : t("settingsPage.geoipMaxMindLicenseKeyPlaceholder")
                }
                onChange={(event) => setMaxmindKey(event.target.value)}
              />
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {settings.maxmind_license_key_set
                  ? t("settingsPage.geoipTokenConfiguredHint")
                  : t("settingsPage.geoipMaxMindLicenseKeyHint")}
              </p>
            </div>
          )}

          <dl className="grid gap-3 rounded-2xl border border-slate-200 bg-white/60 p-4 text-xs dark:border-white/8 dark:bg-slate-950/50 sm:grid-cols-2">
            <div>
              <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("settingsPage.geoipDatabasePath")}</dt>
              <dd className="mt-1 break-all font-mono text-slate-800 dark:text-slate-100">{settings.database_path || "-"}</dd>
            </div>
            <div>
              <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("settingsPage.geoipDatabaseVersion")}</dt>
              <dd className="mt-1 font-mono text-slate-800 dark:text-slate-100">{settings.database_version || "-"}</dd>
            </div>
            <div>
              <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("settingsPage.geoipDatabaseSize")}</dt>
              <dd className="mt-1 text-slate-800 dark:text-slate-100">{formatBytes(settings.database_size_bytes)}</dd>
            </div>
            <div>
              <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("settingsPage.geoipDatabaseUpdatedAt")}</dt>
              <dd className="mt-1 text-slate-800 dark:text-slate-100">{formatTime(settings.database_modified_at)}</dd>
            </div>
          </dl>

          {settings.last_error ? (
            <p role="status" className="text-xs text-amber-700 dark:text-amber-300">
              {t("settingsPage.geoipLastError")}: {settings.last_error}
            </p>
          ) : null}

          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" disabled={saving || updating} className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium">
              {saving ? t("settingsPage.geoipSaving") : t("settingsPage.geoipSave")}
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={saving || updating}
              onClick={() => void handleUpdateDatabase()}
              className="h-10 rounded-xl px-4 text-sm font-medium"
            >
              {updating ? t("settingsPage.geoipUpdating") : t("settingsPage.geoipUpdateDatabase")}
            </Button>
            {success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && (
              <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">
                {error}
              </p>
            )}
          </div>
        </form>
      )}
    </section>
  )
}
