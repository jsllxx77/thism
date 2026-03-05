import { Card } from "antd"

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
}

export function OverviewStats({
  onlineNodes,
  totalNodes,
  avgCpu,
  avgMemory,
  alertCount,
  heartbeatLatencyMs,
}: Props) {
  const items: StatItem[] = [
    { label: "Online Nodes", value: `${onlineNodes} / ${totalNodes}` },
    { label: "Avg CPU", value: avgCpu === null ? "—" : `${avgCpu.toFixed(1)}%` },
    { label: "Avg Memory", value: avgMemory === null ? "—" : `${avgMemory.toFixed(1)}%` },
    { label: "Alerts", value: `${alertCount}` },
    { label: "Heartbeat", value: heartbeatLatencyMs === null ? "—" : `${heartbeatLatencyMs} ms` },
  ]

  return (
    <section className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-5 gap-3">
      {items.map((item) => (
        <Card key={item.label} className="glass-panel !border-white/15 !rounded-xl">
          <p className="text-xs text-white/55">{item.label}</p>
          <p className="text-xl font-semibold mt-2 text-white tracking-tight">{item.value}</p>
        </Card>
      ))}
    </section>
  )
}
