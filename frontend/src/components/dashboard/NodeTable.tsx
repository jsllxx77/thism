import { useMemo, useState, type ReactNode } from "react"
import { ArrowUpDown, ChevronUp } from "lucide-react"
import { useLanguage } from "../../i18n/language"
import type { Node } from "../../lib/api"
import { formatBytes, formatCompactBytes, formatCompactBytesPerSecond } from "../../lib/units"
import { CountryFlag } from "../../components/CountryFlag"

type SortKey = "name" | "status"

type LiveTableMetrics = Record<string, {
  cpu?: number
  memUsed?: number
  memTotal?: number
  netRxSpeed?: number
  netTxSpeed?: number
}>

type Props = {
  nodes: Node[]
  liveMetrics?: LiveTableMetrics
  onSelectNode: (id: string) => void
}

function SortButton({
  active,
  ascending,
  children,
  onClick,
}: {
  active: boolean
  ascending: boolean
  children: ReactNode
  onClick: () => void
}) {
  const Icon = active ? ChevronUp : ArrowUpDown

  return (
    <button
      type="button"
      onClick={onClick}
      data-active={active}
      className="motion-sort-button text-left hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:hover:text-slate-200"
    >
      <span>{children}</span>
      <Icon className={`motion-sort-icon h-3.5 w-3.5 ${active && !ascending ? "rotate-180" : ""}`} />
    </button>
  )
}

function StatusPill({ online, label }: { online: boolean; label: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-semibold ${
        online
          ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
          : "border-[hsl(var(--destructive)/0.35)] bg-[hsl(var(--destructive)/0.12)] text-[hsl(var(--destructive))]"
      }`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${
          online ? "bg-emerald-500" : "bg-[hsl(var(--destructive))]"
        }`}
      />
      {label}
    </span>
  )
}

function clampPercent(value: number) {
  return Math.min(100, Math.max(0, value))
}

function metricTone(value: number | null) {
  if (value === null) return "bg-slate-200 dark:bg-slate-700"
  if (value >= 85) return "bg-[hsl(var(--destructive))]"
  if (value >= 70) return "bg-amber-500"
  return "bg-emerald-500"
}

function MetricBar({ value, className = "" }: { value: number | null; className?: string }) {
  const width = value === null ? 0 : clampPercent(value)
  return (
    <div className={`h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-slate-800 ${className}`}>
      <div className={`h-full rounded-full transition-[width] duration-300 ${metricTone(value)}`} style={{ width: `${width}%` }} />
    </div>
  )
}

function percentLabel(value: number | null, unavailable: string) {
  return value === null ? unavailable : `${value.toFixed(1)}%`
}

function platformLabel(node: Node, unavailable: string) {
  return [node.os, node.arch].filter(Boolean).join("/") || unavailable
}

function metricSnapshot(node: Node, liveMetrics?: LiveTableMetrics[string]) {
  const snapshot = node.latest_metrics
  const cpu = typeof liveMetrics?.cpu === "number"
    ? liveMetrics.cpu
    : typeof snapshot?.cpu === "number"
      ? snapshot.cpu
      : null

  const memUsed = typeof liveMetrics?.memUsed === "number" ? liveMetrics.memUsed : snapshot?.mem_used
  const memTotal = typeof liveMetrics?.memTotal === "number" ? liveMetrics.memTotal : snapshot?.mem_total
  const memory = typeof memUsed === "number" && typeof memTotal === "number" && memTotal > 0
    ? {
        percent: (memUsed / memTotal) * 100,
        detail: `${formatCompactBytes(memUsed)} / ${formatCompactBytes(memTotal)}`,
      }
    : null

  const disk = snapshot && snapshot.disk_total > 0
    ? {
        percent: (snapshot.disk_used / snapshot.disk_total) * 100,
        detail: `${formatCompactBytes(snapshot.disk_used)} / ${formatCompactBytes(snapshot.disk_total)}`,
      }
    : null

  const hasNetSpeed = typeof liveMetrics?.netRxSpeed === "number" || typeof liveMetrics?.netTxSpeed === "number"
  const network = hasNetSpeed
    ? {
        primary: `↓ ${typeof liveMetrics?.netRxSpeed === "number" ? formatCompactBytesPerSecond(liveMetrics.netRxSpeed) : "—"}`,
        secondary: `↑ ${typeof liveMetrics?.netTxSpeed === "number" ? formatCompactBytesPerSecond(liveMetrics.netTxSpeed) : "—"}`,
      }
    : snapshot
      ? {
          primary: `↓ ${formatCompactBytes(snapshot.net_rx)}`,
          secondary: `↑ ${formatCompactBytes(snapshot.net_tx)}`,
          title: `${formatBytes(snapshot.net_rx)} received / ${formatBytes(snapshot.net_tx)} sent`,
        }
      : null

  return { cpu, memory, disk, network }
}

function MetricCell({ label, value, detail }: { label: string; value: number | null; detail?: string }) {
  const { t } = useLanguage()
  return (
    <div className="grid min-w-[6.5rem] grid-rows-[1.25rem_0.375rem_1rem] gap-y-1">
      <div className="flex items-baseline justify-between gap-2 leading-none">
        <span className="sr-only">{label}</span>
        <span className="tabular-nums text-sm font-semibold text-slate-900 dark:text-slate-100">{percentLabel(value, t("common.unavailable"))}</span>
      </div>
      <MetricBar value={value} />
      <p className="truncate text-[11px] leading-4 text-slate-500 dark:text-slate-400" aria-hidden={detail ? undefined : true}>
        {detail ?? "\u00a0"}
      </p>
    </div>
  )
}

export function NodeTable({ nodes, liveMetrics = {}, onSelectNode }: Props) {
  const { t } = useLanguage()
  const [sortKey, setSortKey] = useState<SortKey>("status")
  const [sortAsc, setSortAsc] = useState(true)

  const sorted = useMemo(() => {
    const list = [...nodes]
    list.sort((left, right) => {
      let value = 0
      if (sortKey === "name") {
        value = left.name.localeCompare(right.name)
      } else {
        value = Number(right.online) - Number(left.online)
      }
      return sortAsc ? value : -value
    })
    return list
  }, [nodes, sortAsc, sortKey])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc((current) => !current)
      return
    }
    setSortKey(key)
    setSortAsc(true)
  }

  return (
    <div className="motion-results-region panel-card enterprise-surface rounded-[24px] p-4">
      <p className="mb-3 text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("dashboard.table.title")}</p>
      <div className="hidden overflow-x-auto md:block">
        <table className="min-w-[920px] w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-[0.18em] text-slate-500 dark:border-slate-700 dark:text-slate-400">
              <th className="pb-2 pr-3 font-medium" aria-sort={sortKey === "name" ? (sortAsc ? "ascending" : "descending") : "none"}>
                <SortButton active={sortKey === "name"} ascending={sortAsc} onClick={() => toggleSort("name")}>
                  {t("dashboard.table.nodeName")}
                </SortButton>
              </th>
              <th className="pb-2 pr-3 font-medium">{t("dashboard.table.os")}</th>
              <th className="pb-2 pr-3 font-medium" aria-sort={sortKey === "status" ? (sortAsc ? "ascending" : "descending") : "none"}>
                <SortButton active={sortKey === "status"} ascending={sortAsc} onClick={() => toggleSort("status")}>
                  {t("dashboard.table.status")}
                </SortButton>
              </th>
              <th className="pb-2 pr-3 font-medium">{t("dashboard.table.cpu")}</th>
              <th className="pb-2 pr-3 font-medium">{t("dashboard.table.memory")}</th>
              <th className="pb-2 pr-3 font-medium">{t("dashboard.table.disk")}</th>
              <th className="pb-2 pr-3 font-medium">{t("dashboard.table.network")}</th>
            </tr>
          </thead>
          <tbody className="motion-table-body">
            {sorted.map((node) => {
              const metrics = metricSnapshot(node, liveMetrics[node.id])
              return (
                <tr
                  key={node.id}
                  className={`motion-table-row border-b border-slate-100 dark:border-slate-800 ${
                    !node.online ? "bg-[hsl(var(--destructive)/0.05)]" : ""
                  }`}
                >
                  <td className="py-3 pr-3 text-slate-900 dark:text-slate-100">
                    <button
                      type="button"
                      onClick={() => onSelectNode(node.id)}
                      aria-label={t("dashboard.openNodeAria", { name: node.name })}
                      className="inline-flex max-w-[14rem] items-center rounded-sm text-left font-medium text-slate-900 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-slate-100 dark:hover:text-slate-200 dark:focus-visible:ring-offset-slate-950"
                    >
                      <CountryFlag countryCode={node.country_code} className="mr-1" />
                      <span className="truncate">{node.name}</span>
                    </button>
                  </td>
                  <td className="py-3 pr-3 font-mono text-xs text-slate-600 dark:text-slate-300">{platformLabel(node, t("common.unavailable"))}</td>
                  <td className="py-3 pr-3 text-slate-600 dark:text-slate-300">
                    <StatusPill online={node.online} label={node.online ? t("common.online") : t("common.offline")} />
                  </td>
                  <td className="py-3 pr-3 text-slate-600 dark:text-slate-300">
                    <MetricCell label={t("dashboard.table.cpu")} value={metrics.cpu} />
                  </td>
                  <td className="py-3 pr-3 text-slate-600 dark:text-slate-300">
                    <MetricCell label={t("dashboard.table.memory")} value={metrics.memory?.percent ?? null} detail={metrics.memory?.detail} />
                  </td>
                  <td className="py-3 pr-3 text-slate-600 dark:text-slate-300">
                    <MetricCell label={t("dashboard.table.disk")} value={metrics.disk?.percent ?? null} detail={metrics.disk?.detail} />
                  </td>
                  <td className="py-3 pr-3 text-slate-600 dark:text-slate-300" title={metrics.network?.title}>
                    {metrics.network ? (
                      <div className="space-y-1 font-mono text-xs tabular-nums text-slate-700 dark:text-slate-200">
                        <p>{metrics.network.primary}</p>
                        <p>{metrics.network.secondary}</p>
                      </div>
                    ) : (
                      t("common.unavailable")
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <div className="grid gap-3 md:hidden">
        {sorted.map((node) => {
          const metrics = metricSnapshot(node, liveMetrics[node.id])
          return (
            <button
              key={node.id}
              type="button"
              onClick={() => onSelectNode(node.id)}
              aria-label={t("dashboard.openNodeAria", { name: node.name })}
              className={`enterprise-inner-surface rounded-2xl p-3 text-left transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 ${
                !node.online ? "border border-[hsl(var(--destructive)/0.28)] bg-[hsl(var(--destructive)/0.05)]" : ""
              }`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="flex min-w-0 items-center text-sm font-semibold text-slate-900 dark:text-slate-100">
                    <CountryFlag countryCode={node.country_code} className="mr-1.5" />
                    <span className="truncate">{node.name}</span>
                  </p>
                  <p className="mt-1 truncate font-mono text-[11px] text-slate-500 dark:text-slate-400">{platformLabel(node, t("common.unavailable"))}</p>
                </div>
                <StatusPill online={node.online} label={node.online ? t("common.online") : t("common.offline")} />
              </div>
              <div className="mt-3 grid grid-cols-3 gap-2">
                {[
                  { label: t("dashboard.table.cpu"), value: metrics.cpu, detail: undefined },
                  { label: t("dashboard.table.memory"), value: metrics.memory?.percent ?? null, detail: metrics.memory?.detail },
                  { label: t("dashboard.table.disk"), value: metrics.disk?.percent ?? null, detail: metrics.disk?.detail },
                ].map((item) => (
                  <div key={item.label} className="rounded-xl border border-slate-200/70 bg-white/55 p-2 dark:border-slate-700/70 dark:bg-slate-950/20">
                    <p className="text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500 dark:text-slate-400">{item.label}</p>
                    <p className="mt-1 tabular-nums text-sm font-semibold text-slate-900 dark:text-slate-100">{percentLabel(item.value, t("common.unavailable"))}</p>
                    <MetricBar value={item.value} className="mt-1.5" />
                  </div>
                ))}
              </div>
              <div className="mt-2 flex items-center justify-between rounded-xl border border-slate-200/70 bg-white/55 px-2.5 py-2 font-mono text-xs text-slate-700 dark:border-slate-700/70 dark:bg-slate-950/20 dark:text-slate-200">
                <span className="font-sans text-[10px] font-semibold uppercase tracking-[0.12em] text-slate-500 dark:text-slate-400">{t("dashboard.table.network")}</span>
                <span>{metrics.network ? `${metrics.network.primary}  ${metrics.network.secondary}` : t("common.unavailable")}</span>
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
