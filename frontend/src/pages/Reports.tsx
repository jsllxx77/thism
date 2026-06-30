import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react"
import { useReducedMotion } from "framer-motion"
import { Activity, ArrowDown, ArrowUp, ChevronsUpDown, Clock, RadioTower } from "lucide-react"
import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"
import { api, type AvailabilityReport, type NodeAvailabilityReport } from "../lib/api"
import { useLanguage } from "../i18n/language"
import { NodeTagChips } from "../components/NodeTagChips"
import { MotionSection } from "../motion/transitions"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "../components/ui/dialog"

const RANGE_OPTIONS = [
  { key: "24h", seconds: 24 * 60 * 60, labelKey: "reportsPage.range24h" },
  { key: "7d", seconds: 7 * 24 * 60 * 60, labelKey: "reportsPage.range7d" },
  { key: "30d", seconds: 30 * 24 * 60 * 60, labelKey: "reportsPage.range30d" },
] as const

type RangeKey = (typeof RANGE_OPTIONS)[number]["key"]
type SortKey = "name" | "availability" | "offline" | "outages" | "p95Latency" | "recentOutage" | "lastSeen"
type SortDirection = "asc" | "desc"
type AbnormalFilter = "all" | "offline" | "below99" | "outages" | "highP95"
type SortState = {
  key: SortKey
  direction: SortDirection
}

const OFFLINE_GRACE_SECONDS = 2 * 60
const HIGH_P95_LATENCY_MS = 300

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
  const remainingSeconds = Math.floor(seconds % 60)
  if (days > 0) return hours > 0 ? `${days}d ${hours}h` : `${days}d`
  if (hours > 0) return minutes > 0 ? `${hours}h ${minutes}m` : `${hours}h`
  if (minutes > 0 && remainingSeconds > 0) return `${minutes}m ${remainingSeconds}s`
  return `${Math.max(1, minutes)}m`
}

function formatLatency(value?: number | null) {
  return typeof value === "number" ? `${value.toFixed(1)} ms` : "—"
}

function getRecentOutageTimestamp(row: NodeAvailabilityReport) {
  return row.last_outage_end ?? row.last_outage_start ?? null
}

function getRecentOutageDuration(row: NodeAvailabilityReport) {
  if (typeof row.last_outage_start !== "number" || typeof row.last_outage_end !== "number") {
    return null
  }
  return Math.max(0, row.last_outage_end - row.last_outage_start)
}

function isCurrentlyOffline(row: NodeAvailabilityReport, reportTo?: number) {
  if (!reportTo || row.last_seen <= 0) return false
  return row.last_seen <= reportTo - OFFLINE_GRACE_SECONDS
}

function hasHighP95(row: NodeAvailabilityReport) {
  return typeof row.latency_p95_ms === "number" && row.latency_p95_ms >= HIGH_P95_LATENCY_MS
}

function sampleCoveragePercent(row: NodeAvailabilityReport) {
  if (row.expected_samples <= 0) return null
  return (row.observed_samples / row.expected_samples) * 100
}

function availabilityColor(value: number) {
  if (value >= 99.9) return "#0f766e"
  if (value >= 99) return "#2563eb"
  return "#d97706"
}

function rowSeverity(row: NodeAvailabilityReport, reportTo?: number) {
  if (isCurrentlyOffline(row, reportTo) || row.availability_percent < 99) return "critical"
  if (row.outage_count > 0 || hasHighP95(row)) return "warning"
  return "stable"
}

function compareNullableValues(left: string | number | null, right: string | number | null, direction: SortDirection) {
  if (left == null && right == null) return 0
  if (left == null) return 1
  if (right == null) return -1

  const result = typeof left === "string" && typeof right === "string"
    ? left.localeCompare(right)
    : Number(left) - Number(right)

  return direction === "asc" ? result : -result
}

function sortValue(row: NodeAvailabilityReport, key: SortKey): string | number | null {
  switch (key) {
    case "name":
      return row.name
    case "availability":
      return row.availability_percent
    case "offline":
      return row.offline_duration_seconds
    case "outages":
      return row.outage_count
    case "p95Latency":
      return row.latency_p95_ms ?? null
    case "recentOutage":
      return getRecentOutageTimestamp(row)
    case "lastSeen":
      return row.last_seen > 0 ? row.last_seen : null
  }
}

function sortRows(rows: NodeAvailabilityReport[], sort: SortState) {
  return [...rows].sort((left, right) => {
    const result = compareNullableValues(sortValue(left, sort.key), sortValue(right, sort.key), sort.direction)
    return result || left.name.localeCompare(right.name)
  })
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
    <section aria-label={title} className={`motion-chart-panel panel-card enterprise-surface rounded-[24px] p-4 ${className}`.trim()}>
      <h2 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{title}</h2>
      {children}
    </section>
  )
}

function ChartLoadingSkeleton() {
  return (
    <div className="chart-skeleton mt-4 rounded-2xl bg-slate-100/80 p-4 dark:bg-slate-900/70" aria-hidden="true">
      <div className="chart-skeleton__axis" />
      <div className="chart-skeleton__bar chart-skeleton__bar--one" />
      <div className="chart-skeleton__bar chart-skeleton__bar--two" />
      <div className="chart-skeleton__bar chart-skeleton__bar--three" />
    </div>
  )
}

function StatusBadge({ children, tone = "neutral" }: { children: ReactNode; tone?: "critical" | "warning" | "stable" | "neutral" }) {
  const toneClass = {
    critical: "border-red-200 bg-red-50 text-red-700 dark:border-red-900/60 dark:bg-red-950/35 dark:text-red-200",
    warning: "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200",
    stable: "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-200",
    neutral: "border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-300",
  }[tone]

  return (
    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold leading-5 ${toneClass}`}>
      {children}
    </span>
  )
}

function getStatusBadge(row: NodeAvailabilityReport, reportTo: number | undefined, t: (key: string, params?: Record<string, string | number | undefined>) => string) {
  if (isCurrentlyOffline(row, reportTo)) {
    return { label: t("reportsPage.statusOffline"), tone: "critical" as const }
  }
  if (row.last_outage_end && reportTo && row.last_outage_end >= reportTo - 3600) {
    return { label: t("reportsPage.statusRecentlyRecovered"), tone: "warning" as const }
  }
  if (row.outage_count > 0) {
    return { label: t("reportsPage.statusRecentOutage"), tone: "warning" as const }
  }
  if (row.availability_percent >= 99.9 && row.outage_count === 0) {
    return { label: t("reportsPage.statusStable"), tone: "stable" as const }
  }
  return { label: t("reportsPage.statusWatch"), tone: "neutral" as const }
}

function SortableTableHeader({
  label,
  sortKey,
  sort,
  onSort,
  className = "",
}: {
  label: string
  sortKey: SortKey
  sort: SortState
  onSort: (key: SortKey) => void
  className?: string
}) {
  const active = sort.key === sortKey
  const Icon = active ? (sort.direction === "asc" ? ArrowUp : ArrowDown) : ChevronsUpDown

  return (
    <th aria-sort={active ? (sort.direction === "asc" ? "ascending" : "descending") : "none"} className={`pb-2 pr-3 font-medium ${className}`.trim()}>
      <button
        type="button"
        onClick={() => onSort(sortKey)}
        className="inline-flex items-center gap-1.5 rounded-md text-left transition-colors hover:text-slate-800 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:hover:text-slate-100"
      >
        <span>{label}</span>
        <Icon className={`h-3.5 w-3.5 ${active ? "text-slate-700 dark:text-slate-100" : "text-slate-400 dark:text-slate-500"}`} aria-hidden="true" />
      </button>
    </th>
  )
}

function AbnormalQuickFilters({
  value,
  counts,
  onChange,
  t,
  language,
}: {
  value: AbnormalFilter
  counts: Record<AbnormalFilter, number>
  onChange: (value: AbnormalFilter) => void
  t: (key: string, params?: Record<string, string | number | undefined>) => string
  language: string
}) {
  const options: Array<{ value: AbnormalFilter; label: string }> = [
    { value: "all", label: t("reportsPage.filterAll") },
    { value: "offline", label: t("reportsPage.filterOffline") },
    { value: "below99", label: t("reportsPage.filterBelow99") },
    { value: "outages", label: t("reportsPage.filterOutages") },
    { value: "highP95", label: t("reportsPage.filterHighP95") },
  ]

  return (
    <div className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
      <span>{t("reportsPage.abnormalFilterLabel")}</span>
      <div role="group" aria-label={t("reportsPage.abnormalFilterLabel")} className="flex flex-wrap gap-2">
        {options.map((option) => {
          const active = value === option.value
          return (
            <button
              key={option.value}
              type="button"
              aria-label={option.label}
              aria-pressed={active}
              onClick={() => onChange(option.value)}
              className={`h-10 cursor-pointer rounded-xl border px-3 text-xs font-semibold transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 ${
                active
                  ? "border-slate-300 bg-slate-900 text-white shadow-sm dark:border-white/10 dark:bg-slate-50 dark:text-slate-950"
                  : "border-slate-200 bg-white/70 text-slate-700 hover:border-slate-300 hover:bg-white dark:border-slate-800 dark:bg-slate-950/60 dark:text-slate-200 dark:hover:border-slate-700"
              }`}
            >
              <span>{option.label}</span>
              <span className={`ml-2 rounded-full px-1.5 py-0.5 text-[10px] ${active ? "bg-white/15" : "bg-slate-100 text-slate-500 dark:bg-slate-800 dark:text-slate-300"}`}>
                {counts[option.value].toLocaleString(language)}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}

function AvailabilityRankingChart({ rows, title, reportTo }: { rows: NodeAvailabilityReport[]; title: string; reportTo?: number }) {
  const reduceMotion = useReducedMotion()
  const data = useMemo(
    () =>
      [...rows]
        .sort((left, right) => right.availability_percent - left.availability_percent || left.name.localeCompare(right.name))
        .slice(0, 8)
        .map((row) => ({
          nodeID: row.node_id,
          name: row.name,
          availability: Number(row.availability_percent.toFixed(2)),
          severity: rowSeverity(row, reportTo),
        })),
    [reportTo, rows],
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
              wrapperStyle={{ outline: "none", transition: "opacity 160ms ease, transform 160ms ease" }}
              contentStyle={chartTooltipStyle()}
              itemStyle={{ color: "hsl(var(--popover-foreground))" }}
              labelStyle={{ color: "hsl(var(--popover-foreground))" }}
              formatter={(value) => [formatPercent(Number(value)), title]}
            />
            <Bar
              dataKey="availability"
              radius={[0, 8, 8, 0]}
              isAnimationActive={!reduceMotion}
              animationDuration={420}
              animationEasing="ease-out"
            >
              {data.map((row) => (
                <Cell key={row.nodeID} fill={row.severity === "critical" ? "#dc2626" : availabilityColor(row.availability)} />
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
  reportTo,
}: {
  rows: NodeAvailabilityReport[]
  title: string
  outageLabel: string
  language: string
  reportTo?: number
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
            <div key={row.node_id} className={`space-y-1.5 rounded-xl p-2 ${isCurrentlyOffline(row, reportTo) ? "bg-red-50/80 dark:bg-red-950/25" : ""}`}>
              <div className="flex items-center justify-between gap-3 text-xs">
                <span className="min-w-0 truncate font-medium text-slate-700 dark:text-slate-200">{row.name}</span>
                <span className="shrink-0 text-slate-500 dark:text-slate-400">
                  {formatSeconds(row.offline_duration_seconds)} · {row.outage_count.toLocaleString(language)} {outageLabel}
                </span>
              </div>
              <div className="h-2 rounded-full bg-slate-100 dark:bg-slate-800">
                <div className={`motion-report-bar h-full rounded-full ${isCurrentlyOffline(row, reportTo) ? "bg-red-500" : "bg-amber-500"}`} style={{ width: `${width}%` }} />
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
              className="motion-report-segment h-full"
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

function ReportCharts({ rows, reportTo }: { rows: NodeAvailabilityReport[]; reportTo?: number }) {
  const { language, t } = useLanguage()

  return (
    <section className="grid grid-cols-1 gap-4 xl:grid-cols-3">
      <AvailabilityRankingChart rows={rows} title={t("reportsPage.chartAvailabilityRanking")} reportTo={reportTo} />
      <div className="grid grid-cols-1 gap-4">
        <OfflineImpactChart rows={rows} title={t("reportsPage.chartOfflineImpact")} outageLabel={t("reportsPage.outages")} language={language} reportTo={reportTo} />
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

function DetailStat({ label, value, tone = "neutral" }: { label: string; value: string; tone?: "critical" | "warning" | "neutral" }) {
  const valueClass = tone === "critical"
    ? "text-red-700 dark:text-red-200"
    : tone === "warning"
      ? "text-amber-700 dark:text-amber-200"
      : "text-slate-900 dark:text-slate-50"

  return (
    <div className="rounded-2xl border border-slate-200 bg-white/70 p-3 dark:border-slate-800 dark:bg-slate-950/50">
      <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">{label}</p>
      <p className={`mt-2 text-base font-semibold ${valueClass}`}>{value}</p>
    </div>
  )
}

function NodeReportDetailDrawer({
  row,
  reportTo,
  nowMs,
  open,
  onOpenChange,
}: {
  row: NodeAvailabilityReport | null
  reportTo?: number
  nowMs: number
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t, formatRelativeLastSeen } = useLanguage()

  if (!row) return null

  const status = getStatusBadge(row, reportTo, t)
  const recentOutageDuration = getRecentOutageDuration(row)
  const coverage = sampleCoveragePercent(row)
  const lastOutageWindow = row.last_outage_start && row.last_outage_end
    ? `${formatRelativeLastSeen(row.last_outage_start, nowMs)} - ${formatRelativeLastSeen(row.last_outage_end, nowMs)}`
    : "—"

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="right-0 top-0 h-full max-h-screen w-full max-w-[520px] translate-x-0 translate-y-0 rounded-none border-y-0 border-r-0 p-0 sm:right-4 sm:top-4 sm:h-[calc(100vh-2rem)] sm:rounded-2xl"
        aria-label={t("reportsPage.detailDialogAria", { name: row.name })}
      >
        <div className="flex h-full flex-col">
          <DialogHeader className="border-b border-slate-200 p-5 pr-12 dark:border-slate-800">
            <div className="flex flex-wrap items-center gap-2">
              <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
              <span className="text-xs text-slate-500 dark:text-slate-400">{t("reportsPage.detailWindow")}</span>
            </div>
            <DialogTitle className="sr-only">{t("reportsPage.detailDialogAria", { name: row.name })}</DialogTitle>
            <div className="mt-3 text-xl font-semibold text-slate-900 dark:text-slate-50">{row.name}</div>
            <DialogDescription>
              {t("reportsPage.samples", { observed: row.observed_samples, expected: row.expected_samples })}
            </DialogDescription>
          </DialogHeader>

          <div className="min-h-0 flex-1 overflow-y-auto p-5">
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              <DetailStat label={t("reportsPage.availability")} value={formatPercent(row.availability_percent)} tone={row.availability_percent < 99 ? "critical" : "neutral"} />
              <DetailStat label={t("reportsPage.offline")} value={formatSeconds(row.offline_duration_seconds)} tone={row.offline_duration_seconds > 0 ? "warning" : "neutral"} />
              <DetailStat label={t("reportsPage.outages")} value={String(row.outage_count)} tone={row.outage_count > 0 ? "warning" : "neutral"} />
              <DetailStat label={t("reportsPage.sampleCoverage")} value={coverage == null ? "—" : formatPercent(coverage)} tone={coverage != null && coverage < 95 ? "warning" : "neutral"} />
              <DetailStat label={t("reportsPage.p50Latency")} value={formatLatency(row.latency_p50_ms)} />
              <DetailStat label={t("reportsPage.p95Latency")} value={formatLatency(row.latency_p95_ms)} tone={hasHighP95(row) ? "warning" : "neutral"} />
            </div>

            <section className="mt-5 space-y-3 rounded-2xl border border-slate-200 bg-slate-50/70 p-4 dark:border-slate-800 dark:bg-slate-900/40">
              <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("reportsPage.outageSummary")}</h3>
              <div className="space-y-3 text-sm">
                <div className="flex items-start gap-3">
                  <Clock className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
                  <div>
                    <p className="font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.lastOutageWindow")}</p>
                    <p className="text-slate-500 dark:text-slate-400">{lastOutageWindow}</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <Activity className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
                  <div>
                    <p className="font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.lastOutageDuration")}</p>
                    <p className="text-slate-500 dark:text-slate-400">{recentOutageDuration == null ? "—" : formatSeconds(recentOutageDuration)}</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <RadioTower className="mt-0.5 h-4 w-4 shrink-0 text-slate-400" aria-hidden="true" />
                  <div>
                    <p className="font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.lastSeen")}</p>
                    <p className="text-slate-500 dark:text-slate-400">{row.last_seen > 0 ? formatRelativeLastSeen(row.last_seen, nowMs) : "—"}</p>
                  </div>
                </div>
              </div>
            </section>

            <section className="mt-5 rounded-2xl border border-slate-200 bg-white/70 p-4 dark:border-slate-800 dark:bg-slate-950/50">
              <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("reportsPage.tags")}</h3>
              <div className="mt-3">
                <NodeTagChips tags={row.tags} emptyLabel="—" />
              </div>
            </section>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

export function Reports() {
  const { language, t, formatRelativeLastSeen } = useLanguage()
  const [rangeKey, setRangeKey] = useState<RangeKey>("24h")
  const [tagFilter, setTagFilter] = useState("all")
  const [abnormalFilter, setAbnormalFilter] = useState<AbnormalFilter>("all")
  const [sort, setSort] = useState<SortState>({ key: "availability", direction: "asc" })
  const [selectedNodeID, setSelectedNodeID] = useState<string | null>(null)
  const [nowMs] = useState(() => Date.now())
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

  const rows = useMemo(() => report?.nodes ?? [], [report])
  const abnormalCounts = useMemo<Record<AbnormalFilter, number>>(() => {
    const reportTo = report?.range.to
    return {
      all: rows.length,
      offline: rows.filter((row) => isCurrentlyOffline(row, reportTo)).length,
      below99: rows.filter((row) => row.availability_percent < 99).length,
      outages: rows.filter((row) => row.outage_count > 0).length,
      highP95: rows.filter(hasHighP95).length,
    }
  }, [report?.range.to, rows])
  const filteredRows = useMemo(() => {
    const reportTo = report?.range.to
    return rows.filter((row) => {
      switch (abnormalFilter) {
        case "offline":
          return isCurrentlyOffline(row, reportTo)
        case "below99":
          return row.availability_percent < 99
        case "outages":
          return row.outage_count > 0
        case "highP95":
          return hasHighP95(row)
        case "all":
          return true
      }
    })
  }, [abnormalFilter, report?.range.to, rows])
  const sortedRows = useMemo(() => sortRows(filteredRows, sort), [filteredRows, sort])
  const selectedNode = useMemo(() => rows.find((row) => row.node_id === selectedNodeID) ?? null, [rows, selectedNodeID])
  const reportMotionKey = [
    rangeKey,
    tagFilter,
    rows.map((row) => row.node_id).join("|"),
  ].join(":")
  const handleSort = useCallback((key: SortKey) => {
    setSort((current) => {
      if (current.key !== key) {
        return { key, direction: key === "name" ? "asc" : "desc" }
      }
      return { key, direction: current.direction === "asc" ? "desc" : "asc" }
    })
  }, [])

  return (
    <MotionSection className="mx-auto max-w-[1440px] space-y-6" testId="motion-reports-root" variant="page">
      <section className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t("reportsPage.title")}</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">{t("reportsPage.subtitle")}</p>
      </section>

      <section className="motion-filter-panel panel-card enterprise-surface rounded-[24px] p-4">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-3 md:flex-row md:items-start">
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
          <AbnormalQuickFilters value={abnormalFilter} counts={abnormalCounts} onChange={setAbnormalFilter} t={t} language={language} />
        </div>
      </section>

      {loading ? (
        <section className="panel-card rounded-2xl border border-slate-200 p-6 dark:border-slate-700">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.loading")}</p>
          <div className="mt-4 space-y-3">
            <div className="content-skeleton h-4 w-1/3">
              <div className="content-skeleton__line h-full w-full" />
            </div>
            <ChartLoadingSkeleton />
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
          <section key={`metrics:${reportMotionKey}`} className="motion-results-region grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <ReportMetric label={t("reportsPage.averageAvailability")} value={formatPercent(report.overview.average_availability_percent)} />
            <ReportMetric label={t("reportsPage.nodesBelow99")} value={String(report.overview.nodes_below_99)} />
            <ReportMetric label={t("reportsPage.totalOffline")} value={formatSeconds(report.overview.total_offline_duration_seconds)} />
            <ReportMetric label={t("reportsPage.highestP95")} value={formatLatency(report.overview.highest_latency_p95_ms)} />
          </section>

          {rows.length === 0 ? (
            <section key={`empty:${reportMotionKey}`} className="motion-empty-state panel-card rounded-2xl border border-slate-200 px-6 py-14 text-center dark:border-slate-700">
              <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("reportsPage.empty")}</p>
            </section>
          ) : (
            <div key={`results:${reportMotionKey}`} className="motion-results-region space-y-6">
              <ReportCharts rows={filteredRows} reportTo={report.range.to} />
              <section className="panel-card enterprise-surface overflow-x-auto rounded-[24px] p-4">
                <div className="mb-3 flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
                  <p className="text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("reportsPage.tableTitle")}</p>
                  <p className="text-xs text-slate-500 dark:text-slate-400">{t("reportsPage.filteredCount", { count: filteredRows.length, total: rows.length })}</p>
                </div>
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-[0.18em] text-slate-500 dark:border-slate-700 dark:text-slate-400">
                      <SortableTableHeader className="pl-3" label={t("reportsPage.node")} sortKey="name" sort={sort} onSort={handleSort} />
                      <th className="pb-2 pr-3 font-medium">{t("reportsPage.tags")}</th>
                      <SortableTableHeader label={t("reportsPage.availability")} sortKey="availability" sort={sort} onSort={handleSort} />
                      <SortableTableHeader label={t("reportsPage.offline")} sortKey="offline" sort={sort} onSort={handleSort} />
                      <SortableTableHeader label={t("reportsPage.outages")} sortKey="outages" sort={sort} onSort={handleSort} />
                      <SortableTableHeader label={t("reportsPage.p95Latency")} sortKey="p95Latency" sort={sort} onSort={handleSort} />
                      <SortableTableHeader label={t("reportsPage.recentOutage")} sortKey="recentOutage" sort={sort} onSort={handleSort} />
                      <SortableTableHeader label={t("reportsPage.lastSeen")} sortKey="lastSeen" sort={sort} onSort={handleSort} />
                    </tr>
                  </thead>
                  <tbody className="motion-table-body">
                    {sortedRows.map((row) => {
                      const recentOutageTimestamp = getRecentOutageTimestamp(row)
                      const recentOutageDuration = getRecentOutageDuration(row)
                      const status = getStatusBadge(row, report.range.to, t)
                      const severity = rowSeverity(row, report.range.to)

                      return (
                        <tr
                          key={row.node_id}
                          className={`motion-table-row border-b border-slate-100 transition-colors hover:bg-slate-50/70 dark:border-slate-800 dark:hover:bg-slate-900/45 ${
                            severity === "critical" ? "bg-red-50/50 dark:bg-red-950/20" : severity === "warning" ? "bg-amber-50/35 dark:bg-amber-950/15" : ""
                          }`}
                        >
                          <td className="py-2.5 pl-3 pr-3 text-slate-900 dark:text-slate-100">
                            <button
                              type="button"
                              onClick={() => setSelectedNodeID(row.node_id)}
                              className="cursor-pointer rounded-md text-left font-medium text-slate-900 transition-colors hover:text-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:text-slate-100 dark:hover:text-blue-300"
                              aria-label={t("reportsPage.openNodeDetails", { name: row.name })}
                            >
                              {row.name}
                            </button>
                            <div className="mt-1">
                              <StatusBadge tone={status.tone}>{status.label}</StatusBadge>
                            </div>
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
                          <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">
                            {recentOutageTimestamp ? (
                              <>
                                <div>{formatRelativeLastSeen(recentOutageTimestamp, nowMs)}</div>
                                {recentOutageDuration != null ? (
                                  <div className="text-xs text-slate-500 dark:text-slate-400">{formatSeconds(recentOutageDuration)}</div>
                                ) : null}
                              </>
                            ) : "—"}
                          </td>
                          <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{row.last_seen > 0 ? formatRelativeLastSeen(row.last_seen, nowMs) : "—"}</td>
                        </tr>
                      )
                    })}
                    {sortedRows.length === 0 ? (
                      <tr>
                        <td colSpan={8} className="py-10 text-center text-sm text-slate-500 dark:text-slate-400">
                          {t("reportsPage.noAbnormalMatches")}
                        </td>
                      </tr>
                    ) : null}
                  </tbody>
                </table>
              </section>
              <NodeReportDetailDrawer
                row={selectedNode}
                reportTo={report.range.to}
                nowMs={nowMs}
                open={selectedNode != null}
                onOpenChange={(open) => {
                  if (!open) setSelectedNodeID(null)
                }}
              />
            </div>
          )}
        </>
      ) : null}
    </MotionSection>
  )
}
