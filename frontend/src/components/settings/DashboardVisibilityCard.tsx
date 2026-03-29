import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api } from "../../lib/api"
import { Button } from "../ui/button"

export function DashboardVisibilityCard() {
  const { t } = useLanguage()
  const [showDashboardCardIP, setShowDashboardCardIP] = useState(false)
  const [savedShowDashboardCardIP, setSavedShowDashboardCardIP] = useState<boolean | null>(null)
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
        const response = await api.dashboardSettings()
        if (cancelled) {
          return
        }
        const nextValue = Boolean(response.show_dashboard_card_ip)
        setShowDashboardCardIP(nextValue)
        setSavedShowDashboardCardIP(nextValue)
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.dashboardVisibilityUpdateFailed"))
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

  const hasChanges = savedShowDashboardCardIP !== null && showDashboardCardIP !== savedShowDashboardCardIP
  const statusLabel = useMemo(
    () => (showDashboardCardIP ? t("settingsPage.dashboardVisibilityVisible") : t("settingsPage.dashboardVisibilityHidden")),
    [showDashboardCardIP, t],
  )

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)

    try {
      const response = await api.updateDashboardSettings({ show_dashboard_card_ip: showDashboardCardIP })
      const nextValue = Boolean(response.show_dashboard_card_ip)
      setShowDashboardCardIP(nextValue)
      setSavedShowDashboardCardIP(nextValue)
      setSuccess(t("settingsPage.dashboardVisibilitySaved"))
    } catch {
      setError(t("settingsPage.dashboardVisibilityUpdateFailed"))
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.dashboardVisibilityTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.dashboardVisibilityDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
          {statusLabel}
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : (
        <form className="mt-4 space-y-4" onSubmit={handleSubmit}>
          <label className="enterprise-inner-surface flex min-h-11 cursor-pointer items-center justify-between rounded-2xl border px-4 py-3">
            <span className="pr-4 text-sm font-medium text-slate-700 dark:text-slate-200">
              {t("settingsPage.dashboardVisibilityShowIp")}
            </span>
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-slate-300 text-slate-900 focus:ring-slate-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
              aria-label={t("settingsPage.dashboardVisibilityShowIp")}
              checked={showDashboardCardIP}
              onChange={(event) => {
                setShowDashboardCardIP(event.target.checked)
                setError(null)
                setSuccess(null)
              }}
            />
          </label>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              disabled={saving || !hasChanges}
              className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
            >
              {saving ? t("settingsPage.dashboardVisibilitySaving") : t("settingsPage.dashboardVisibilitySave")}
            </Button>
            {success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
          </div>
        </form>
      )}
    </section>
  )
}
