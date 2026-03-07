import { useEffect, useState } from "react"
import { Cpu, MemoryStick } from "lucide-react"
import { useLanguage } from "../i18n/language"
import type { Node } from "../lib/api"
import { Badge } from "./ui/badge"
import { Card, CardContent } from "./ui/card"

type Props = {
  node: Node
  cpu?: number
  memUsed?: number
  memTotal?: number
  showIP?: boolean
  onClick?: () => void
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
        className="h-1.5 rounded-full transition-all duration-200"
        style={{ width: `${safeValue}%`, backgroundColor: metricColor(value) }}
      />
    </div>
  )
}

export function NodeCard({ node, cpu, memUsed, memTotal, showIP = true, onClick }: Props) {
  const { t, formatRelativeLastSeen } = useLanguage()
  const hasCpu = typeof cpu === "number"
  const hasMemory = typeof memUsed === "number" && typeof memTotal === "number" && memTotal > 0
  const memPct = hasMemory ? (memUsed / memTotal) * 100 : null
  const [nowMs, setNowMs] = useState(() => Date.now())
  const platformLabel = [node.os, node.arch].filter(Boolean).join("/") || t("common.unavailable")
  const subtitle = showIP ? `${node.ip || t("common.unavailable")} · ${platformLabel}` : platformLabel

  useEffect(() => {
    if (node.last_seen <= 0) {
      return undefined
    }

    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [node.last_seen])

  const formattedLastSeen = formatRelativeLastSeen(node.last_seen, nowMs)

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={node.name}
      className={`w-full rounded-2xl border-0 bg-transparent p-0 text-left transition-transform hover:-translate-y-0.5 ${
        !node.online ? "opacity-80" : ""
      }`}
    >
      <Card className="panel-card panel-card-hover enterprise-surface rounded-[24px]">
        <CardContent className="p-4">
          <div className="mb-4 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h3 className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{node.name}</h3>
              <p className="mt-1 truncate text-[11px] text-slate-500 dark:text-slate-400">{subtitle}</p>
            </div>
            <Badge
              variant={node.online ? "secondary" : "outline"}
              className={
                node.online
                  ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
                  : "border-slate-300 bg-slate-50 text-slate-600 dark:border-white/10 dark:bg-slate-900 dark:text-slate-300"
              }
            >
              {node.online ? t("common.online") : t("common.offline")}
            </Badge>
          </div>

          <div className="enterprise-inner-surface space-y-3 rounded-2xl p-3 text-xs">
            <div>
              <div className="mb-1 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
                  <Cpu className="h-3 w-3" /> {t("dashboard.nodeCard.cpu")}
                </span>
                <span className="text-slate-700 dark:text-slate-200">{hasCpu ? `${cpu.toFixed(1)}%` : t("common.unavailable")}</span>
              </div>
              <MetricBar value={hasCpu ? Number(cpu.toFixed(1)) : 0} hasValue={hasCpu} />
            </div>

            <div>
              <div className="mb-1 flex items-center justify-between">
                <span className="flex items-center gap-1.5 text-slate-500 dark:text-slate-400">
                  <MemoryStick className="h-3 w-3" /> {t("dashboard.nodeCard.memory")}
                </span>
                <span className="text-slate-700 dark:text-slate-200">{memPct === null ? t("common.unavailable") : `${memPct.toFixed(1)}%`}</span>
              </div>
              <MetricBar value={memPct === null ? 0 : Number(memPct.toFixed(1))} hasValue={memPct !== null} />
            </div>
          </div>

          <p className="mt-3 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">{t("dashboard.nodeCard.lastSeen", { value: formattedLastSeen })}</p>
        </CardContent>
      </Card>
    </button>
  )
}
