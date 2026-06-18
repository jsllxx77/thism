import { memo, useEffect, useRef, useState } from "react"
import { Cpu, MemoryStick } from "lucide-react"
import { useLanguage } from "../i18n/language"
import type { Node } from "../lib/api"
import { formatBytesPerSecond } from "../lib/units"
import { CountryFlag } from "./CountryFlag"
import { NodeTagChips } from "./NodeTagChips"
import { Badge } from "./ui/badge"
import { Card, CardContent } from "./ui/card"

type Props = {
  node: Node
  cpu?: number
  memUsed?: number
  memTotal?: number
  netRxSpeed?: number
  netTxSpeed?: number
  showIP?: boolean
  onClick?: () => void
  onSelectNode?: (id: string) => void
}

function isDarkModeEnabled(): boolean {
  return typeof document !== "undefined" && document.documentElement.classList.contains("dark")
}

function metricColor(value: number): string {
  if (value > 80) return isDarkModeEnabled() ? "#c55f5f" : "#c16f6f"
  if (value > 60) return "#335f9f"
  return isDarkModeEnabled() ? "#1f7a52" : "#7db58c"
}

function MetricBar({ value, hasValue = true }: { value: number; hasValue?: boolean }) {
  const safeValue = hasValue ? Math.min(100, Math.max(0, value)) : 0
  return (
    <div className="h-1.5 rounded-full bg-slate-100 dark:bg-[#20242a]">
      <div
        className="h-1.5 rounded-full transition-[width,background-color] duration-300 ease-out"
        style={{ width: `${safeValue}%`, backgroundColor: metricColor(value) }}
      />
    </div>
  )
}

function useValueFlash(value: string | number | null | undefined) {
  const previous = useRef(value)
  const [flash, setFlash] = useState(false)

  useEffect(() => {
    if (previous.current === value) {
      return undefined
    }

    const hadPreviousValue = previous.current !== null && previous.current !== undefined
    previous.current = value

    if (!hadPreviousValue || value === null || value === undefined) {
      return undefined
    }

    const startId = window.setTimeout(() => {
      setFlash(true)
    }, 0)
    const endId = window.setTimeout(() => {
      setFlash(false)
    }, 360)

    return () => {
      window.clearTimeout(startId)
      window.clearTimeout(endId)
    }
  }, [value])

  return flash
}

function metricValueClass(flash: boolean, className = "") {
  return `metric-value tabular-nums text-slate-700 dark:text-slate-200 ${flash ? "metric-value--flash" : ""} ${className}`.trim()
}

function RelativeLastSeenLabel({ lastSeen, offline = false }: { lastSeen: number; offline?: boolean }) {
  const { t, formatRelativeLastSeen } = useLanguage()
  const [nowMs, setNowMs] = useState(() => Date.now())

  useEffect(() => {
    if (lastSeen <= 0) {
      return undefined
    }

    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [lastSeen])

  return (
    <p
      className={`mt-3 text-[11px] font-medium uppercase tracking-[0.16em] ${
        offline ? "font-semibold text-[hsl(var(--destructive))]" : "text-slate-500 dark:text-slate-400"
      }`}
    >
      {t("dashboard.nodeCard.lastSeen", { value: formatRelativeLastSeen(lastSeen, nowMs) })}
    </p>
  )
}

export const NodeCard = memo(function NodeCard({ node, cpu, memUsed, memTotal, netRxSpeed, netTxSpeed, showIP = true, onClick, onSelectNode }: Props) {
  const { t } = useLanguage()
  const hasCpu = typeof cpu === "number"
  const hasMemory = typeof memUsed === "number" && typeof memTotal === "number" && memTotal > 0
  const memPct = hasMemory ? (memUsed / memTotal) * 100 : null
  const showNetSpeed = node.online
  const hasNetRxSpeed = showNetSpeed && typeof netRxSpeed === "number" && Number.isFinite(netRxSpeed) && netRxSpeed >= 0
  const hasNetTxSpeed = showNetSpeed && typeof netTxSpeed === "number" && Number.isFinite(netTxSpeed) && netTxSpeed >= 0
  const netRxLabel = hasNetRxSpeed ? formatBytesPerSecond(netRxSpeed) : "—"
  const netTxLabel = hasNetTxSpeed ? formatBytesPerSecond(netTxSpeed) : "—"
  const cpuLabel = hasCpu ? `${cpu.toFixed(1)}%` : t("common.unavailable")
  const memLabel = memPct === null ? t("common.unavailable") : `${memPct.toFixed(1)}%`
  const cpuFlash = useValueFlash(hasCpu ? Number(cpu.toFixed(1)) : null)
  const memFlash = useValueFlash(memPct === null ? null : Number(memPct.toFixed(1)))
  const platformLabel = [node.os, node.arch].filter(Boolean).join("/") || t("common.unavailable")
  const subtitle = showIP ? `${node.ip || t("common.unavailable")} · ${platformLabel}` : platformLabel
  const tags = node.tags ?? []
  const handleClick = onClick ?? (onSelectNode ? () => onSelectNode(node.id) : undefined)

  return (
    <button
      type="button"
      onClick={handleClick}
      aria-label={node.name}
      className={`group/card h-full w-full rounded-2xl border-0 bg-transparent p-0 text-left transition-[opacity,transform] duration-200 ease-out hover:-translate-y-1 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white active:translate-y-0 dark:focus-visible:ring-offset-slate-950`}
    >
      <Card
        className={`node-card-shell theme-dashboard-card panel-card panel-card-hover enterprise-surface h-full rounded-[24px] ${
          !node.online
            ? "border-l-4 border-l-[hsl(var(--destructive))] bg-[hsl(var(--destructive)/0.04)] dark:bg-[hsl(var(--destructive)/0.08)]"
            : ""
        }`}
      >
        <CardContent className="relative z-[1] p-4">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h3 className="flex min-w-0 items-center text-sm font-semibold text-slate-900 dark:text-slate-100">
                <CountryFlag countryCode={node.country_code} className="mr-1" />
                <span className="truncate">{node.name}</span>
              </h3>
              <p className="mt-1 truncate text-[11px] text-slate-500 dark:text-slate-400">{subtitle}</p>
              <div data-node-card-tag-slot className="mt-2 min-h-[22px]">
                {tags.length > 0 && <NodeTagChips tags={tags} />}
              </div>
            </div>
            <Badge
              variant={node.online ? "secondary" : "outline"}
              className={
                node.online
                  ? "gap-1.5 border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
                  : "gap-1.5 border-[hsl(var(--destructive)/0.35)] bg-[hsl(var(--destructive)/0.12)] font-semibold text-[hsl(var(--destructive))]"
              }
            >
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  node.online
                    ? "bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.12)]"
                    : "bg-[hsl(var(--destructive))] shadow-[0_0_0_3px_hsl(var(--destructive)/0.18)]"
                }`}
              />
              {node.online ? t("common.online") : t("common.offline")}
            </Badge>
          </div>

          <div className="enterprise-inner-surface space-y-3 rounded-2xl p-3 text-xs">
            <div>
              <div className="mb-1 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
                  <Cpu className="h-3 w-3" /> {t("dashboard.nodeCard.cpu")}
                </span>
                <span className={metricValueClass(cpuFlash)}>{cpuLabel}</span>
              </div>
              <MetricBar value={hasCpu ? Number(cpu.toFixed(1)) : 0} hasValue={hasCpu} />
            </div>

            <div>
              <div className="mb-1 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
                  <MemoryStick className="h-3 w-3" /> {t("dashboard.nodeCard.memory")}
                </span>
                <span className={metricValueClass(memFlash)}>{memLabel}</span>
              </div>
              <MetricBar value={memPct === null ? 0 : Number(memPct.toFixed(1))} hasValue={memPct !== null} />
            </div>

            <div className={`space-y-1 ${showNetSpeed ? "" : "opacity-60"}`}>
              <div className="flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">{t("dashboard.nodeCard.inboundSpeed")}</span>
                <span className={metricValueClass(false, "dashboard-net-speed-value")}>↓ {netRxLabel}</span>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-slate-500 dark:text-slate-400">{t("dashboard.nodeCard.outboundSpeed")}</span>
                <span className={metricValueClass(false, "dashboard-net-speed-value")}>↑ {netTxLabel}</span>
              </div>
            </div>
          </div>

          <RelativeLastSeenLabel lastSeen={node.last_seen} offline={!node.online} />
        </CardContent>
      </Card>
    </button>
  )
})
