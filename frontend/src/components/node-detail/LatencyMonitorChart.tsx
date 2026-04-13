import { useMemo, useState, type ReactNode } from "react"
import { CartesianGrid, Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { useLanguage } from "../../i18n/language"
import type { LatencyMonitor, LatencyMonitorResult } from "../../lib/api"
import { buildLatencyMonitorSeries } from "./latency-monitor-series"
import { shouldRenderLatencyDots } from "./latency-monitor-chart-utils"

type Props = {
  monitors: LatencyMonitor[]
  results: LatencyMonitorResult[]
  range: number
}

const SERIES_COLORS = ["#2563eb", "#0f766e", "#d97706", "#dc2626", "#7c3aed", "#0891b2"]

function summarizeLatencyFailure(errorMessage: string, language: string): string {
  const normalized = errorMessage.trim().toLowerCase()
  if (!normalized) {
    return language === "zh-CN" ? "探测失败" : "Probe failed"
  }
  if (normalized.includes("timeout")) {
    return language === "zh-CN" ? "超时" : "Timeout"
  }
  if (normalized.includes("connection refused")) {
    return language === "zh-CN" ? "连接被拒绝" : "Connection refused"
  }
  if (normalized.includes("no such host")) {
    return language === "zh-CN" ? "解析失败" : "DNS failed"
  }
  if (normalized.includes("network is unreachable")) {
    return language === "zh-CN" ? "网络不可达" : "Network unreachable"
  }
  return language === "zh-CN" ? "探测失败" : "Probe failed"
}

function formatPercent(value?: number | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "—"
  }
  return `${value.toFixed(value % 1 === 0 ? 0 : 1)}%`
}

function formatJitter(value?: number | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "—"
  }
  return `${value.toFixed(1)} ms`
}

function TooltipContent({
  active,
  label,
  resultsByTimestamp,
  monitorByID,
  visibleSeries,
}: {
  active?: boolean
  label?: string | number
  resultsByTimestamp: Record<number, LatencyMonitorResult[]>
  monitorByID: Map<string, LatencyMonitor>
  visibleSeries: Record<string, boolean>
}) {
  const { language } = useLanguage()

  if (!active || typeof label !== "number") {
    return null
  }

  const rows = (resultsByTimestamp[label] ?? []).filter((result) => visibleSeries[result.monitor_id] ?? true)
  if (rows.length === 0) {
    return null
  }

  return (
    <div
      className="rounded-xl border border-slate-200 bg-white/95 px-3 py-2 text-xs shadow-xl dark:border-white/10 dark:bg-slate-950/95"
      role="status"
    >
      <p className="font-medium text-slate-800 dark:text-slate-100">
        {new Date(label * 1000).toLocaleString(language, {
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })}
      </p>
      <div className="mt-2 space-y-1.5">
        {rows.map((result) => {
          const monitor = monitorByID.get(result.monitor_id)
          const value: ReactNode =
            result.success && result.latency_ms != null
              ? `${result.latency_ms.toFixed(1)} ms`
              : summarizeLatencyFailure(result.error_message ?? "", language)
          return (
            <div key={`${result.monitor_id}-${result.ts}`} className="flex items-start justify-between gap-3">
              <span className="text-slate-600 dark:text-slate-300">{monitor?.name ?? result.monitor_id}</span>
              <span className={`text-right font-medium ${result.success ? "text-slate-900 dark:text-slate-100" : "text-red-600 dark:text-red-300"}`}>
                {value}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function LatencyMonitorChart({ monitors, results, range }: Props) {
  const { t, language } = useLanguage()
  const [visibleSeries, setVisibleSeries] = useState<Record<string, boolean>>({})
  const normalizedVisibleSeries = useMemo(() => {
    const next: Record<string, boolean> = {}
    for (const monitor of monitors) {
      next[monitor.id] = visibleSeries[monitor.id] ?? true
    }
    return next
  }, [monitors, visibleSeries])

  const monitorByID = useMemo(() => {
    const map = new Map<string, LatencyMonitor>()
    for (const monitor of monitors) {
      map.set(monitor.id, monitor)
    }
    return map
  }, [monitors])

  const chartState = useMemo(() => buildLatencyMonitorSeries(monitors, results, range), [monitors, range, results])
  const showDots = shouldRenderLatencyDots(range, chartState.chartData.length)
  const latestResultByMonitorID = useMemo(() => {
    const latest = new Map<string, LatencyMonitorResult>()
    for (const result of results) {
      const current = latest.get(result.monitor_id)
      if (!current || result.ts >= current.ts) {
        latest.set(result.monitor_id, result)
      }
    }
    return latest
  }, [results])

  const showDateInTimeLabels = range >= 86400
  const formatXAxis = (value: number) =>
    showDateInTimeLabels
      ? new Date(value * 1000).toLocaleString(language, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
      : new Date(value * 1000).toLocaleTimeString(language, { hour: "2-digit", minute: "2-digit" })

  if (monitors.length === 0) {
    return (
      <section className="panel-card enterprise-surface rounded-[24px] p-4">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("nodeDetail.latencyMonitorsTitle")}</h3>
        <p className="mt-3 text-sm text-slate-500 dark:text-slate-400">{t("nodeDetail.latencyMonitorsEmpty")}</p>
      </section>
    )
  }

  return (
    <section className="panel-card enterprise-surface rounded-[24px] p-4">
      <div className="flex flex-col gap-4">
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("nodeDetail.latencyMonitorsTitle")}</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">{t("nodeDetail.latencyMonitorsDescription")}</p>
        </div>

        <div className="flex flex-wrap gap-2">
          {monitors.map((monitor, index) => {
            const active = normalizedVisibleSeries[monitor.id] ?? true
            return (
              <button
                key={monitor.id}
                type="button"
                aria-pressed={active}
                onClick={() => setVisibleSeries((current) => ({ ...current, [monitor.id]: !(current[monitor.id] ?? true) }))}
                className={`rounded-full border px-3 py-1.5 text-xs font-medium transition-colors ${
                  active
                    ? "border-slate-300 bg-slate-100 text-slate-900 dark:border-white/10 dark:bg-slate-900 dark:text-slate-50"
                    : "border-slate-200 bg-white/80 text-slate-500 dark:border-white/8 dark:bg-slate-950/80 dark:text-slate-400"
                }`}
                style={active ? { boxShadow: `inset 0 0 0 1px ${SERIES_COLORS[index % SERIES_COLORS.length]}33` } : undefined}
              >
                <span className="block text-left leading-tight">{monitor.name}</span>
                <span className="mt-1 block text-[10px] font-medium leading-tight text-slate-500 dark:text-slate-400">
                  {`${t("nodeDetail.lossLabel")} ${formatPercent(latestResultByMonitorID.get(monitor.id)?.loss_percent)}  ${t("nodeDetail.jitterLabel")} ${formatJitter(latestResultByMonitorID.get(monitor.id)?.jitter_ms)}`}
                </span>
              </button>
            )
          })}
        </div>

        <ResponsiveContainer width="100%" height={220}>
          <LineChart data={chartState.chartData} margin={{ top: 10, right: 12, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="rgba(100,116,139,0.18)" />
            <XAxis
              dataKey="ts"
              tickFormatter={(value) => formatXAxis(Number(value))}
              tick={{ fill: "#6b7280", fontSize: 10 }}
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              tickFormatter={(value) => `${Number(value).toFixed(0)} ms`}
              tick={{ fill: "#6b7280", fontSize: 10 }}
              axisLine={false}
              tickLine={false}
              domain={[0, "auto"]}
            />
            <Tooltip
              content={(
                <TooltipContent
                  resultsByTimestamp={chartState.resultsByTimestamp}
                  monitorByID={monitorByID}
                  visibleSeries={normalizedVisibleSeries}
                />
              )}
            />
            {monitors.map((monitor, index) =>
              (normalizedVisibleSeries[monitor.id] ?? true) ? (
                <Line
                  key={monitor.id}
                  dataKey={monitor.id}
                  name={monitor.name}
                  stroke={SERIES_COLORS[index % SERIES_COLORS.length]}
                  strokeWidth={2}
                  dot={showDots ? { r: 2, strokeWidth: 0, fill: SERIES_COLORS[index % SERIES_COLORS.length] } : false}
                  activeDot={{ r: 4 }}
                  connectNulls
                  isAnimationActive={false}
                />
              ) : null
            )}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </section>
  )
}
