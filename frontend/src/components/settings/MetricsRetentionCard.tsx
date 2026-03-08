import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api } from "../../lib/api"
import { Button } from "../ui/button"

const DEFAULT_RETENTION_DAYS = 7
const DEFAULT_OPTIONS = [7, 30]

function normalizeOptions(options: number[] | undefined, fallback: number) {
  const values = Array.isArray(options)
    ? options.filter((value): value is number => typeof value === "number" && Number.isFinite(value))
    : []
  const merged = values.includes(fallback) ? values : [...values, fallback]
  return Array.from(new Set(merged)).sort((left, right) => left - right)
}

function getRetentionLabel(t: (key: string) => string, days: number) {
  if (days === 30) {
    return t("settingsPage.metricsRetention30Days")
  }
  return t("settingsPage.metricsRetention7Days")
}

export function MetricsRetentionCard() {
  const { t } = useLanguage()
  const [options, setOptions] = useState<number[]>(DEFAULT_OPTIONS)
  const [selectedDays, setSelectedDays] = useState(DEFAULT_RETENTION_DAYS)
  const [savedDays, setSavedDays] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const loadRetention = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await api.metricsRetention()
        if (cancelled) {
          return
        }
        const retentionDays = typeof response.retention_days === "number" ? response.retention_days : DEFAULT_RETENTION_DAYS
        setOptions(normalizeOptions(response.options, retentionDays))
        setSelectedDays(retentionDays)
        setSavedDays(retentionDays)
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.metricsRetentionUpdateFailed"))
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void loadRetention()

    return () => {
      cancelled = true
    }
  }, [t])

  const currentLabel = useMemo(() => getRetentionLabel(t, selectedDays), [selectedDays, t])
  const hasChanges = savedDays !== null && selectedDays !== savedDays

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    setSuccess(null)
    setSaving(true)

    try {
      const response = await api.updateMetricsRetention(selectedDays)
      const retentionDays = typeof response.retention_days === "number" ? response.retention_days : selectedDays
      setOptions(normalizeOptions(response.options, retentionDays))
      setSelectedDays(retentionDays)
      setSavedDays(retentionDays)
      setSuccess(t("settingsPage.metricsRetentionSaved"))
    } catch {
      setError(t("settingsPage.metricsRetentionUpdateFailed"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.metricsRetentionTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.metricsRetentionDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
          {currentLabel}
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : (
        <form className="mt-4 space-y-4" onSubmit={handleSubmit}>
          <fieldset>
            <legend className="sr-only">{t("settingsPage.metricsRetentionTitle")}</legend>
            <div className="grid gap-3 md:grid-cols-2">
              {options.map((days) => {
                const checked = selectedDays === days
                return (
                  <label
                    key={days}
                    className={`enterprise-inner-surface flex min-h-11 cursor-pointer items-center justify-between rounded-2xl border px-4 py-3 transition-colors ${
                      checked
                        ? "border-slate-300 bg-slate-50 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50"
                        : "border-slate-200 bg-white/80 text-slate-600 hover:bg-slate-50 dark:border-white/8 dark:bg-slate-950/80 dark:text-slate-200 dark:hover:bg-slate-900"
                    }`}
                  >
                    <div>
                      <input
                        type="radio"
                        name="metrics-retention"
                        className="sr-only"
                        checked={checked}
                        onChange={() => {
                          setSelectedDays(days)
                          setError(null)
                          setSuccess(null)
                        }}
                      />
                      <span className="text-sm font-medium">{getRetentionLabel(t, days)}</span>
                    </div>
                    <span
                      aria-hidden="true"
                      className={`flex h-4 w-4 items-center justify-center rounded-full border ${
                        checked
                          ? "border-slate-900 dark:border-slate-100"
                          : "border-slate-300 dark:border-slate-600"
                      }`}
                    >
                      {checked && <span className="h-2 w-2 rounded-full bg-slate-900 dark:bg-slate-100" />}
                    </span>
                  </label>
                )
              })}
            </div>
          </fieldset>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              disabled={saving || !hasChanges}
              className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
            >
              {saving ? t("settingsPage.metricsRetentionSaving") : t("settingsPage.metricsRetentionSave")}
            </Button>
            {success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
          </div>
        </form>
      )}
    </section>
  )
}
