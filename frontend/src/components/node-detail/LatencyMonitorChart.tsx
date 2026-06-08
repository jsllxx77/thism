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

function formatLatency(value?: number | null): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "—"
  }
  return `${value.toFixed(1)} ms`
}

function formatJitter(value?: number | null): string {
  return formatLatency(value)
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
      className="motion-chart-tooltip rounded-xl border border-slate-200 bg-white/95 px-3 py-2 text-xs shadow-xl dark:border-white/10 dark:bg-slate-950/95"
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
  const averageLatencyByMonitorID = useMemo(() => {
    const totals = new Map<string, { sum: number; count: number }>()
    const knownMonitorIDs = new Set(monitors.map((monitor) => monitor.id))

    for (const result of results) {
      if (!knownMonitorIDs.has(result.monitor_id) || !result.success || typeof result.latency_ms !== "number" || !Number.isFinite(result.latency_ms)) {
        continue
      }
      const current = totals.get(result.monitor_id) ?? { sum: 0, count: 0 }
      current.sum += result.latency_ms
      current.count += 1
      totals.set(result.monitor_id, current)
    }

    const averages = new Map<string, number>()
    for (const [monitorID, total] of totals.entries()) {
      if (total.count > 0) {
        averages.set(monitorID, total.sum / total.count)
      }
    }
    return averages
  }, [monitors, results])

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
    <section className="motion-chart-panel panel-card enterprise-surface rounded-[24px] p-4">
      <div className="flex flex-col gap-4">
        <div className="space-y-2">
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("nodeDetail.latencyMonitorsTitle")}</h3>
          <p className="text-xs text-slate-500 dark:text-slate-400">{t("nodeDetail.latencyMonitorsDescription")}</p>
        </div>

        <div className="grid grid-cols-1 gap-2.5 sm:grid-cols-2">
          {monitors.map((monitor, index) => {
            const active = normalizedVisibleSeries[monitor.id] ?? true
            const seriesColor = SERIES_COLORS[index % SERIES_COLORS.length]
            const latestResult = latestResultByMonitorID.get(monitor.id)
            return (
              <button
                key={monitor.id}
                type="button"
                aria-pressed={active}
                onClick={() => setVisibleSeries((current) => ({ ...current, [monitor.id]: !(current[monitor.id] ?? true) }))}
                className={`group flex min-h-[98px] flex-col rounded-2xl border px-3.5 py-3 text-left transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-400/70 ${
                  active
                    ? "border-slate-300 bg-white/90 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-50"
                    : "border-slate-200 bg-white/55 text-slate-500 opacity-70 dark:border-white/8 dark:bg-slate-950/55 dark:text-slate-400"
                }`}
                style={
                  active
                    ? {
                        borderColor: `${seriesColor}66`,
                        boxShadow: `inset 0 0 0 1px ${seriesColor}22, 0 10px 26px -20px ${seriesColor}`,
                      }
                    : undefined
                }
              >
                <span className="flex min-w-0 items-center gap-2.5">
                  <span
                    className="h-2.5 w-2.5 shrink-0 rounded-full"
                    style={{ backgroundColor: seriesColor, boxShadow: active ? `0 0 12px ${seriesColor}99` : undefined }}
                    aria-hidden="true"
                  />
                  <span className="block min-w-0 truncate text-[13px] font-semibold leading-snug tracking-tight text-slate-900 dark:text-slate-50">
                    {monitor.name}
                  </span>
                </span>

                <span className="mt-2.5 flex items-end justify-between gap-3">
                  <span className="min-w-0">
                    <span className="block text-[10px] font-medium leading-none text-slate-500 dark:text-slate-400">
                      {t("nodeDetail.averageLatencyLabel")}
                    </span>{" "}
                    <span className="mt-1 block text-[20px] font-semibold leading-none tracking-tight text-slate-950 dark:text-white tabular-nums">
                      {formatLatency(averageLatencyByMonitorID.get(monitor.id))}
                    </span>
                  </span>
                  <span className="grid min-w-[118px] grid-cols-2 gap-1.5">
                    <span className="rounded-xl bg-slate-100/80 px-2 py-1.5 dark:bg-white/[0.04]">
                      <span className="block text-[10px] font-medium leading-none text-slate-500 dark:text-slate-400">
                        {t("nodeDetail.lossLabel")}
                      </span>{" "}
                      <span className="mt-1 block text-[12px] font-semibold leading-none text-slate-800 dark:text-slate-100 tabular-nums">
                        {formatPercent(latestResult?.loss_percent)}
                      </span>
                    </span>
                    <span className="rounded-xl bg-slate-100/80 px-2 py-1.5 dark:bg-white/[0.04]">
                      <span className="block text-[10px] font-medium leading-none text-slate-500 dark:text-slate-400">
                        {t("nodeDetail.jitterLabel")}
                      </span>{" "}
                      <span className="mt-1 block text-[12px] font-semibold leading-none text-slate-800 dark:text-slate-100 tabular-nums">
                        {formatJitter(latestResult?.jitter_ms)}
                      </span>
                    </span>
                  </span>
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
              cursor={{ stroke: "rgba(37, 99, 235, 0.34)", strokeWidth: 1.5, strokeDasharray: "4 4" }}
              wrapperStyle={{ outline: "none", transition: "opacity 160ms ease, transform 160ms ease" }}
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
                  activeDot={{ r: 5, stroke: "hsl(var(--background))", strokeWidth: 2, fill: SERIES_COLORS[index % SERIES_COLORS.length] }}
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
