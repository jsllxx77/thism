import { useEffect, useMemo, useState } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type AgentReleaseManifest } from "../../lib/api"

type ReleaseState = {
  amd64: AgentReleaseManifest | null
  arm64: AgentReleaseManifest | null
}

function formatInterval(t: (key: string, params?: Record<string, string | number | undefined>) => string, seconds: number) {
  const minutes = Math.max(1, Math.floor(seconds / 60))
  return t("settingsPage.autoUpdateChecksEveryMinutes", { count: minutes })
}

export function AgentAutoUpdateCard() {
  const { t } = useLanguage()
  const [releaseState, setReleaseState] = useState<ReleaseState>({ amd64: null, arm64: null })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const [amd64, arm64] = await Promise.all([
          api.agentRelease("linux", "amd64"),
          api.agentRelease("linux", "arm64"),
        ])
        if (!cancelled) {
          setReleaseState({ amd64, arm64 })
        }
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.autoUpdateUnavailable"))
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

  const intervalLabel = useMemo(() => {
    const seconds = releaseState.amd64?.check_interval_seconds ?? releaseState.arm64?.check_interval_seconds
    return seconds ? formatInterval(t, seconds) : null
  }, [releaseState, t])

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.autoUpdateTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.autoUpdateDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-emerald-700 dark:text-emerald-300">
          {t("settingsPage.autoUpdateEnabled")}
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-300">{error}</p>
      ) : (
        <div className="mt-4 space-y-3">
          {intervalLabel && (
            <p className="text-xs font-medium text-slate-600 dark:text-slate-300">{intervalLabel}</p>
          )}
          <div className="grid gap-3 md:grid-cols-2">
            {releaseState.amd64 && (
              <div className="enterprise-inner-surface rounded-2xl px-4 py-3 text-sm">
                <p className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">linux/amd64</p>
                <p className="mt-1 font-medium text-slate-800 dark:text-slate-100">linux/amd64 · {releaseState.amd64.target_version}</p>
              </div>
            )}
            {releaseState.arm64 && (
              <div className="enterprise-inner-surface rounded-2xl px-4 py-3 text-sm">
                <p className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">linux/arm64</p>
                <p className="mt-1 font-medium text-slate-800 dark:text-slate-100">linux/arm64 · {releaseState.arm64.target_version}</p>
              </div>
            )}
          </div>
        </div>
      )}
    </section>
  )
}
