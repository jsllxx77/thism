import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react"
import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { api, type AvailabilityReport, type NodeAvailabilityReport } from "../lib/api"
import { useLanguage } from "../i18n/language"
import { NodeTagChips } from "../components/NodeTagChips"
import { MotionSection } from "../motion/transitions"

const RANGE_OPTIONS = [
  { key: "24h", seconds: 24 * 60 * 60, labelKey: "reportsPage.range24h" },
  { key: "7d", seconds: 7 * 24 * 60 * 60, labelKey: "reportsPage.range7d" },
  { key: "30d", seconds: 30 * 24 * 60 * 60, labelKey: "reportsPage.range30d" },
] as const

type RangeKey = (typeof RANGE_OPTIONS)[number]["key"]

const SLA_DISTRIBUTION_COLORS = {
  excellent: "#0f766e",
  standard: "#2563eb",
  attention: "#d97706",
}

function formatPercent(value: number) {
  return `${value.toFixed(2)}%`
}

function formatSeconds(seconds: number) {
  if (seconds <= 0) return "0m"
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return hours > 0 ? `${days}d ${hours}h` : `${days}d`
  if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
  return `${Math.max(1, minutes)}m`
}

function formatLatency(value?: number | null) {
  return typeof value === "number" ? `${value.toFixed(1)} ms` : "—"
}

function availabilityColor(value: number) {
  if (value >= 99.9) return "#0f766e"
  if (value >= 99) return "#2563eb"
  return "#d97706"
}

function chartTooltipStyle() {
  return {
    background: "hsl(var(--popover))",
    border: "1px solid hsl(var(--border))",
    borderRadius: 12,
    color: "hsl(var(--popover-foreground))",
    fontSize: 12,
    boxShadow: "0 16px 32px rgba(15, 23, 42, 0.18)",
  }
}

function ReportMetric({ label, value }: { label: string; value: string }) {
  return (
    <div className="panel-card enterprise-surface rounded-[24px] p-5">
      <p className="text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{label}</p>
      <p className="mt-3 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-50">{value}</p>
    </div>
  )
}

function ReportChartPanel({ title, className = "", children }: { title: string; className?: string; children: ReactNode }) {
  return (
    <section aria-label={title} className={`panel-card enterprise-surface rounded-[24px] p-4 ${className}`.trim()}>
      <h2 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{title}</h2>
      {children}
    </section>
  )
}

function AvailabilityRankingChart({ rows, title }: { rows: NodeAvailabilityReport[]; title: string }) {
  const data = useMemo(
    () =>
      [...rows]
        .sort((left, right) => right.availability_percent - left.availability_percent || left.name.localeCompare(right.name))
        .slice(0, 8)
        .map((row) => ({
          nodeID: row.node_id,
          name: row.name,
          availability: Number(row.availability_percent.toFixed(2)),
        })),
    [rows],
  )

  return (
    <ReportChartPanel title={title} className="xl:col-span-2">
      <div className="mt-4 h-[260px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} layout="vertical" margin={{ top: 4, right: 18, bottom: 0, left: 4 }}>
            <CartesianGrid strokeDasharray="3 3" horizontal={false} stroke="hsl(var(--border))" />
            <XAxis
              type="number"
              domain={[0, 100]}
              tickFormatter={(value) => `${Number(value).toFixed(0)}%`}
              tick={{ fill: "#64748b", fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <YAxis
              type="category"
              dataKey="name"
              width={88}
              tick={{ fill: "#64748b", fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip
              cursor={{ fill: "rgba(148, 163, 184, 0.12)" }}
              contentStyle={chartTooltipStyle()}
              itemStyle={{ color: "hsl(var(--popover-foreground))" }}
              labelStyle={{ color: "hsl(var(--popover-foreground))" }}
              formatter={(value) => [formatPercent(Number(value)), title]}
            />
            <Bar dataKey="availability" radius={[0, 8, 8, 0]} isAnimationActive={false}>
              {data.map((row) => (
                <Cell key={row.nodeID} fill={availabilityColor(row.availability)} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </ReportChartPanel>
  )
}

function OfflineImpactChart({
  rows,
  title,
  outageLabel,
  language,
}: {
  rows: NodeAvailabilityReport[]
  title: string
  outageLabel: string
  language: string
}) {
  const data = useMemo(
    () =>
      [...rows]
        .sort(
          (left, right) =>
            right.offline_duration_seconds - left.offline_duration_seconds ||
            right.outage_count - left.outage_count ||
            left.name.localeCompare(right.name),
        )
        .slice(0, 5),
    [rows],
  )
  const maxOffline = Math.max(1, ...data.map((row) => row.offline_duration_seconds))

  return (
    <ReportChartPanel title={title}>
      <div className="mt-4 space-y-3">
        {data.map((row) => {
          const width = row.offline_duration_seconds > 0 ? Math.max(5, (row.offline_duration_seconds / maxOffline) * 100) : 0
          return (
            <div key={row.node_id} className="space-y-1.5">
              <div className="flex items-center justify-between gap-3 text-xs">
                <span className="min-w-0 truncate font-medium text-slate-700 dark:text-slate-200">{row.name}</span>
                <span className="shrink-0 text-slate-500 dark:text-slate-400">
                  {formatSeconds(row.offline_duration_seconds)} · {row.outage_count.toLocaleString(language)} {outageLabel}
                </span>
              </div>
              <div className="h-2 rounded-full bg-slate-100 dark:bg-slate-800">
                <div className="h-full rounded-full bg-amber-500" style={{ width: `${width}%` }} />
              </div>
            </div>
          )
        })}
      </div>
    </ReportChartPanel>
  )
}

function SlaDistributionChart({
  rows,
  title,
  labels,
  nodeCountLabel,
}: {
  rows: NodeAvailabilityReport[]
  title: string
  labels: { excellent: string; standard: string; attention: string }
  nodeCountLabel: (count: number) => string
}) {
  const distribution = useMemo(() => {
    const next = { excellent: 0, standard: 0, attention: 0 }
    for (const row of rows) {
      if (row.availability_percent >= 99.9) {
        next.excellent += 1
      } else if (row.availability_percent >= 99) {
        next.standard += 1
      } else {
        next.attention += 1
      }
    }
    return next
  }, [rows])

  const total = Math.max(1, rows.length)
  const segments = [
    { key: "excellent" as const, label: labels.excellent, count: distribution.excellent, color: SLA_DISTRIBUTION_COLORS.excellent },
    { key: "standard" as const, label: labels.standard, count: distribution.standard, color: SLA_DISTRIBUTION_COLORS.standard },
    { key: "attention" as const, label: labels.attention, count: distribution.attention, color: SLA_DISTRIBUTION_COLORS.attention },
  ]

  return (
    <ReportChartPanel title={title}>
      <div className="mt-5 flex h-4 overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800">
        {segments.map((segment) =>
          segment.count > 0 ? (
            <div
              key={segment.key}
              className="h-full"
              style={{ flexBasis: `${(segment.count / total) * 100}%`, backgroundColor: segment.color }}
              title={`${segment.label}: ${nodeCountLabel(segment.count)}`}
            />
          ) : null,
        )}
      </div>
      <div className="mt-5 space-y-3">
        {segments.map((segment) => (
          <div key={segment.key} className="flex items-center justify-between gap-3 text-xs">
            <span className="inline-flex min-w-0 items-center gap-2 text-slate-600 dark:text-slate-300">
              <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: segment.color }} />
              <span className="truncate">{segment.label}</span>
            </span>
            <span className="shrink-0 font-medium text-slate-800 dark:text-slate-100">{nodeCountLabel(segment.count)}</span>
          </div>
        ))}
      </div>
    </ReportChartPanel>
  )
}

function ReportCharts({ rows }: { rows: NodeAvailabilityReport[] }) {
  const { language, t } = useLanguage()

  return (
    <section className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      <AvailabilityRankingChart rows={rows} title={t("reportsPage.chartAvailabilityRanking")} />
      <div className="grid grid-cols-1 gap-4">
        <OfflineImpactChart rows={rows} title={t("reportsPage.chartOfflineImpact")} outageLabel={t("reportsPage.outages")} language={language} />
        <SlaDistributionChart
          rows={rows}
          title={t("reportsPage.chartSlaDistribution")}
          labels={{
            excellent: t("reportsPage.slaExcellent"),
            standard: t("reportsPage.slaStandard"),
            attention: t("reportsPage.slaAttention"),
          }}
          nodeCountLabel={(count) => t("reportsPage.nodeCount", { count })}
        />
      </div>
    </section>
  )
}

export function Reports() {
  const { language, t, formatRelativeLastSeen } = useLanguage()
  const [rangeKey, setRangeKey] = useState<RangeKey>("24h")
  const [tagFilter, setTagFilter] = useState("all")
  const [report, setReport] = useState<AvailabilityReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const selectedRange = RANGE_OPTIONS.find((option) => option.key === rangeKey) ?? RANGE_OPTIONS[0]

  const loadReport = useCallback(async () => {
    setLoading(true)
    setError(null)
    const to = Math.floor(Date.now() / 1000)
    const from = to - selectedRange.seconds

    try {
      const nextReport = await api.availabilityReport(from, to, tagFilter === "all" ? undefined : tagFilter)
      setReport(nextReport)
    } catch {
      setError(t("reportsPage.loadError"))
    } finally {
      setLoading(false)
    }
  }, [selectedRange.seconds, tagFilter, t])

  useEffect(() => {
    void loadReport()
  }, [loadReport])

  const availableTags = report?.available_tags ?? []
  const nowMs = Date.now()

  const rows = useMemo(() => report?.nodes ?? [], [report])

  return (
    <MotionSection className="mx-auto max-w-[1440px] space-y-6" testId="motion-reports-root">
      <section className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t("reportsPage.title")}</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">{t("reportsPage.subtitle")}</p>
      </section>

      <section className="panel-card enterprise-surface rounded-[24px] p-4">
        <div className="flex flex-col gap-3 md:flex-row md:items-end">
          <div className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
            <span>{t("reportsPage.rangeLabel")}</span>
            <div role="group" aria-label={t("reportsPage.rangeLabel")} className="enterprise-inner-surface inline-flex w-full gap-1 rounded-2xl p-1.5 shadow-none md:w-auto md:p-1">
              {RANGE_OPTIONS.map((option) => {
                const active = rangeKey === option.key
                return (
                  <button
                    key={option.key}
                    type="button"
                    aria-pressed={active}
                    onClick={() => setRangeKey(option.key)}
                    className={`h-11 flex-1 cursor-pointer rounded-xl border border-transparent px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] transition-all duration-200 focus:outline-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-inset active:translate-y-px active:scale-[0.99] md:h-10 md:flex-initial ${
                      active
                        ? "border border-slate-200/80 bg-slate-50/90 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50 dark:ring-1 dark:ring-inset dark:ring-white/10 dark:shadow-none"
                        : "text-slate-600 hover:bg-white/85 dark:text-slate-200 dark:hover:bg-slate-900"
                    }`}
                  >
                    {t(option.labelKey)}
                  </button>
                )
              })}
            </div>
          </div>

          <label className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300 md:min-w-48">
            {t("reportsPage.tagLabel")}
            <select
              aria-label={t("reportsPage.tagLabel")}
              value={tagFilter}
              onChange={(event) => setTagFilter(event.target.value)}
              className="enterprise-outline-control h-11 rounded-xl border bg-white px-3 text-sm text-slate-800 shadow-none transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:bg-slate-950 dark:text-slate-100 md:h-10"
            >
              <option value="all">{t("reportsPage.allTags")}</option>
              {availableTags.map((tag) => (
                <option key={tag} value={tag}>
                  {tag}
                </option>
              ))}
            </select>
          </label>
        </div>
      </section>

      {loading ? (
        <section className="panel-card rounded-2xl border border-slate-200 p-6 dark:border-slate-700">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.loading")}</p>
          <div className="mt-4 space-y-3">
            <div className="h-4 w-1/3 animate-pulse rounded bg-slate-200/80 dark:bg-slate-700/70" />
            <div className="h-24 animate-pulse rounded-xl bg-slate-200/70 dark:bg-slate-800/70" />
          </div>
        </section>
      ) : error ? (
        <section role="alert" className="panel-card rounded-2xl border border-red-200 bg-red-50/80 p-5 dark:border-red-900/60 dark:bg-red-950/30">
          <p className="text-sm font-medium text-red-700 dark:text-red-300">{error}</p>
          <button
            type="button"
            onClick={() => void loadReport()}
            className="mt-3 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:border-red-800 dark:bg-red-950/50 dark:text-red-200 dark:hover:bg-red-900/40"
          >
            {t("common.retry")}
          </button>
        </section>
      ) : report ? (
        <>
          <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <ReportMetric label={t("reportsPage.averageAvailability")} value={formatPercent(report.overview.average_availability_percent)} />
            <ReportMetric label={t("reportsPage.nodesBelow99")} value={String(report.overview.nodes_below_99)} />
            <ReportMetric label={t("reportsPage.totalOffline")} value={formatSeconds(report.overview.total_offline_duration_seconds)} />
            <ReportMetric label={t("reportsPage.highestP95")} value={formatLatency(report.overview.highest_latency_p95_ms)} />
          </section>

          {rows.length === 0 ? (
            <section className="panel-card rounded-2xl border border-slate-200 px-6 py-14 text-center dark:border-slate-700">
              <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.empty")}</p>
            </section>
          ) : (
            <>
              <ReportCharts rows={rows} />
              <section className="panel-card enterprise-surface overflow-x-auto rounded-[24px] p-4">
                <p className="mb-3 text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("reportsPage.tableTitle")}</p>
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-[0.18em] text-slate-500 dark:border-slate-700 dark:text-slate-400">
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.node")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.tags")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.availability")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.offline")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.outages")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.p95Latency")}</th>
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.lastSeen")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rows.map((row) => (
                      <tr key={row.node_id} className="border-b border-slate-100 hover:bg-white/80 dark:border-slate-800 dark:hover:bg-white/[0.02]">
                        <td className="py-2.5 pr-3 text-slate-900 dark:text-slate-100">
                          <div className="font-medium">{row.name}</div>
                          <div className="text-xs text-slate-500 dark:text-slate-400">
                            {t("reportsPage.samples", { observed: row.observed_samples, expected: row.expected_samples })}
                          </div>
                        </td>
                        <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">
                          <NodeTagChips tags={row.tags} emptyLabel="—" />
                        </td>
                        <td className="py-2.5 pr-3 font-medium text-slate-700 dark:text-slate-200">{formatPercent(row.availability_percent)}</td>
                        <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{formatSeconds(row.offline_duration_seconds)}</td>
                        <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{row.outage_count.toLocaleString(language)}</td>
                        <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{formatLatency(row.latency_p95_ms)}</td>
                        <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{row.last_seen > 0 ? formatRelativeLastSeen(row.last_seen, nowMs) : "—"}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            </>
          )}
        </>
      ) : null}
    </MotionSection>
  )
}
