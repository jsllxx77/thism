import { useEffect, useMemo, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type DashboardSettings } from "../../lib/api"
import { Button } from "../ui/button"

const DEFAULT_SETTINGS: DashboardSettings = {
  show_dashboard_card_ip: true,
  show_system_pressure: true,
  show_memory_pressure: true,
}

export function DashboardVisibilityCard() {
  const { t } = useLanguage()
  const [settings, setSettings] = useState<DashboardSettings>(DEFAULT_SETTINGS)
  const [savedSettings, setSavedSettings] = useState<DashboardSettings | null>(null)
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
        const nextValue: DashboardSettings = {
          show_dashboard_card_ip: Boolean(response.show_dashboard_card_ip),
          show_system_pressure: response.show_system_pressure !== false,
          show_memory_pressure: response.show_memory_pressure !== false,
        }
        setSettings(nextValue)
        setSavedSettings(nextValue)
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

  const hasChanges =
    savedSettings !== null &&
    (settings.show_dashboard_card_ip !== savedSettings.show_dashboard_card_ip ||
      settings.show_system_pressure !== savedSettings.show_system_pressure ||
      settings.show_memory_pressure !== savedSettings.show_memory_pressure)

  const enabledCount = useMemo(
    () =>
      Number(settings.show_dashboard_card_ip) +
      Number(settings.show_system_pressure) +
      Number(settings.show_memory_pressure),
    [settings],
  )
  const statusLabel = t("settingsPage.dashboardVisibilityEnabledCount", { count: enabledCount, total: 3 })

  const updateSetting = (key: keyof DashboardSettings, checked: boolean) => {
    setSettings((current) => ({ ...current, [key]: checked }))
    setError(null)
    setSuccess(null)
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)

    try {
      const response = await api.updateDashboardSettings(settings)
      const nextValue: DashboardSettings = {
        show_dashboard_card_ip: Boolean(response.show_dashboard_card_ip),
        show_system_pressure: response.show_system_pressure !== false,
        show_memory_pressure: response.show_memory_pressure !== false,
      }
      setSettings(nextValue)
      setSavedSettings(nextValue)
      setSuccess(t("settingsPage.dashboardVisibilitySaved"))
    } catch {
      setError(t("settingsPage.dashboardVisibilityUpdateFailed"))
    } finally {
      setSaving(false)
    }
  }

  const toggles: Array<{ key: keyof DashboardSettings; labelKey: string }> = [
    { key: "show_dashboard_card_ip", labelKey: "settingsPage.dashboardVisibilityShowIp" },
    { key: "show_system_pressure", labelKey: "settingsPage.dashboardVisibilityShowSystemPressure" },
    { key: "show_memory_pressure", labelKey: "settingsPage.dashboardVisibilityShowMemoryPressure" },
  ]

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
          <div className="space-y-3">
            {toggles.map((item) => (
              <label
                key={item.key}
                className="enterprise-inner-surface flex min-h-11 cursor-pointer items-center justify-between rounded-2xl border px-4 py-3"
              >
                <span className="pr-4 text-sm font-medium text-slate-700 dark:text-slate-200">{t(item.labelKey)}</span>
                <input
                  type="checkbox"
                  className="h-4 w-4 rounded border-slate-300 text-slate-900 focus:ring-slate-500 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-100"
                  aria-label={t(item.labelKey)}
                  checked={settings[item.key]}
                  onChange={(event) => updateSetting(item.key, event.target.checked)}
                />
              </label>
            ))}
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button
              type="submit"
              disabled={saving || !hasChanges}
              className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
            >
              {saving ? t("settingsPage.dashboardVisibilitySaving") : t("settingsPage.dashboardVisibilitySave")}
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
