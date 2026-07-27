import { useEffect, useMemo, useRef, useState } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type DashboardSettings } from "../../lib/api"

const DEFAULT_SETTINGS: DashboardSettings = {
  show_dashboard_card_ip: true,
  show_system_pressure: true,
  show_memory_pressure: true,
}

function normalizeSettings(response: DashboardSettings): DashboardSettings {
  return {
    show_dashboard_card_ip: Boolean(response.show_dashboard_card_ip),
    show_system_pressure: response.show_system_pressure !== false,
    show_memory_pressure: response.show_memory_pressure !== false,
  }
}

export function DashboardVisibilityCard() {
  const { t } = useLanguage()
  const [settings, setSettings] = useState<DashboardSettings>(DEFAULT_SETTINGS)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const saveGenerationRef = useRef(0)
  const settingsRef = useRef(settings)
  settingsRef.current = settings

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
        const nextValue = normalizeSettings(response)
        setSettings(nextValue)
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

  const enabledCount = useMemo(
    () =>
      Number(settings.show_dashboard_card_ip) +
      Number(settings.show_system_pressure) +
      Number(settings.show_memory_pressure),
    [settings],
  )
  const statusLabel = t("settingsPage.dashboardVisibilityEnabledCount", { count: enabledCount, total: 3 })

  const updateSetting = (key: keyof DashboardSettings, checked: boolean) => {
    const previous = settingsRef.current
    if (previous[key] === checked) {
      return
    }

    const nextValue: DashboardSettings = { ...previous, [key]: checked }
    const generation = ++saveGenerationRef.current
    setSettings(nextValue)
    settingsRef.current = nextValue
    setError(null)
    setSuccess(null)
    setSaving(true)

    void (async () => {
      try {
        const response = await api.updateDashboardSettings(nextValue)
        if (generation !== saveGenerationRef.current) {
          return
        }
        const saved = normalizeSettings(response)
        setSettings(saved)
        settingsRef.current = saved
        setSuccess(t("settingsPage.dashboardVisibilitySaved"))
      } catch {
        if (generation !== saveGenerationRef.current) {
          return
        }
        // Resync from server so a failed write never leaves a divergent local toggle state.
        try {
          const response = await api.dashboardSettings()
          if (generation !== saveGenerationRef.current) {
            return
          }
          const saved = normalizeSettings(response)
          setSettings(saved)
          settingsRef.current = saved
        } catch {
          if (generation !== saveGenerationRef.current) {
            return
          }
          setSettings(previous)
          settingsRef.current = previous
        }
        setError(t("settingsPage.dashboardVisibilityUpdateFailed"))
      } finally {
        if (generation === saveGenerationRef.current) {
          setSaving(false)
        }
      }
    })()
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
        <div className="mt-4 space-y-4">
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

          <div className="flex flex-wrap items-center gap-3 min-h-5">
            {saving && <p className="text-xs font-medium text-slate-500 dark:text-slate-400">{t("settingsPage.dashboardVisibilitySaving")}</p>}
            {!saving && success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && (
              <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">
                {error}
              </p>
            )}
          </div>
        </div>
      )}
    </section>
  )
}
