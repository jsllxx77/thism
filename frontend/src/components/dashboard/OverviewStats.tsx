import { useLanguage } from "../../i18n/language"
import { Card, CardContent } from "../ui/card"

type Props = {
  onlineNodes: number
  totalNodes: number
  avgCpu: number | null
  avgMemory: number | null
  alertCount: number
  heartbeatLatencyMs: number | null
}

type StatItem = {
  label: string
  value: string
  accent?: "neutral" | "green" | "amber"
}

export function OverviewStats({
  onlineNodes,
  totalNodes,
  avgCpu,
  avgMemory,
  alertCount,
  heartbeatLatencyMs,
}: Props) {
  const { t } = useLanguage()
  const offlineAlerts = alertCount
  const items: StatItem[] = [
    { label: t("dashboard.stats.totalNodes"), value: `${totalNodes}`, accent: "neutral" },
    { label: t("dashboard.stats.onlineNodes"), value: t("dashboard.stats.onlineValue", { online: onlineNodes, total: totalNodes }), accent: "green" },
    { label: t("dashboard.stats.avgCpu"), value: avgCpu === null ? t("common.unavailable") : `${avgCpu.toFixed(1)}%`, accent: "neutral" },
    { label: t("dashboard.stats.avgMemory"), value: avgMemory === null ? t("common.unavailable") : `${avgMemory.toFixed(1)}%`, accent: "neutral" },
    { label: t("dashboard.stats.offlineAlerts"), value: `${offlineAlerts}`, accent: offlineAlerts > 0 ? "amber" : "neutral" },
  ]

  const accentClass = (accent: StatItem["accent"]) => {
    if (accent === "green") return "text-emerald-700 dark:text-emerald-300"
    if (accent === "amber") return "text-amber-700 dark:text-amber-300"
    return "text-slate-900 dark:text-slate-50"
  }

  const _heartbeat = heartbeatLatencyMs
  void _heartbeat

  return (
    <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-5">
      {items.map((item) => (
        <Card key={item.label} className="panel-card enterprise-surface rounded-[24px]">
          <CardContent className="p-5">
            <p className="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500 dark:text-slate-400">{item.label}</p>
            <p className={`mt-3 text-2xl font-semibold tracking-tight ${accentClass(item.accent)}`}>{item.value}</p>
          </CardContent>
        </Card>
      ))}
    </section>
  )
}
