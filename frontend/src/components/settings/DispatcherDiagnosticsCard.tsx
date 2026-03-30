import { useCallback, useEffect, useState } from "react"
import { api, type DispatcherRuntimeStats } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { Button } from "../ui/button"

type Props = {
  refreshNonce?: number
}

const DIAGNOSTICS_REFRESH_INTERVAL_MS = 5000

export function DispatcherDiagnosticsCard({ refreshNonce = 0 }: Props) {
  const { t } = useLanguage()
  const diagnosticsRequest = (api as { dispatcherRuntimeStats?: () => Promise<DispatcherRuntimeStats> }).dispatcherRuntimeStats
  const supported = typeof diagnosticsRequest === "function"
  const [stats, setStats] = useState<DispatcherRuntimeStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const loadDiagnostics = useCallback(async (showLoading = false) => {
    if (!supported || !diagnosticsRequest) {
      return
    }

    setError(null)
    if (showLoading) {
      setLoading(true)
    }

    try {
      const response = await diagnosticsRequest()
      setStats(response)
    } catch {
      setError(t("Dispatcher diagnostics are currently unavailable."))
    } finally {
      setLoading(false)
    }
  }, [diagnosticsRequest, supported, t])

  useEffect(() => {
    if (!supported) {
      return
    }

    void loadDiagnostics(true)
    const intervalId = window.setInterval(() => {
      void loadDiagnostics()
    }, DIAGNOSTICS_REFRESH_INTERVAL_MS)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [loadDiagnostics, supported])

  useEffect(() => {
    if (!supported || refreshNonce === 0) {
      return
    }
    void loadDiagnostics(true)
  }, [loadDiagnostics, refreshNonce, supported])

  if (!supported) {
    return null
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("Dispatcher Diagnostics")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
            {t("Runtime visibility for alert queue pressure and delivery throughput.")}
          </p>
        </div>
        <Button type="button" className="h-9 rounded-xl px-3 text-xs" onClick={() => void loadDiagnostics()}>
          {t("Retry")}
        </Button>
      </div>

      {loading && !stats ? (
        <p role="status" className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading dispatcher diagnostics...")}</p>
      ) : error && !stats ? (
        <p role="status" className="mt-4 text-xs text-slate-500 dark:text-slate-400">{error}</p>
      ) : stats ? (
        <div className="mt-4 grid gap-3 text-xs sm:grid-cols-2 xl:grid-cols-4">
          <Metric label={t("Active dispatchers")} value={stats.active_dispatchers} />
          <Metric label={t("Total capacity")} value={stats.total_capacity} />
          <Metric label={t("Queue depth")} value={stats.queue_depth} />
          <Metric label={t("High watermark")} value={stats.high_watermark} />
          <Metric label={t("Enqueued")} value={stats.enqueued} />
          <Metric label={t("Processed")} value={stats.processed} />
          <Metric label={t("Dropped")} value={stats.dropped} />
        </div>
      ) : null}

      {error && stats ? (
        <p role="status" className="mt-3 text-xs text-amber-600 dark:text-amber-300">{error}</p>
      ) : null}
    </section>
  )
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="enterprise-inner-surface rounded-2xl border border-slate-200 px-4 py-3 dark:border-white/10">
      <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{label}</dt>
      <dd className="mt-1 font-mono text-sm text-slate-800 dark:text-slate-100">{value}</dd>
    </div>
  )
}
