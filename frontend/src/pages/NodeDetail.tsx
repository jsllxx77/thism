import { useEffect, useState } from "react"
import { api } from "../lib/api"
import type { Node, MetricsRow, Process, ServiceCheck } from "../lib/api"
import { MetricsChart } from "../components/MetricsChart"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { cn } from "../lib/utils"

const RANGES = [
  { label: "1h", seconds: 3600 },
  { label: "6h", seconds: 21600 },
  { label: "24h", seconds: 86400 },
  { label: "7d", seconds: 604800 },
] as const

type Props = {
  nodeId: string
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

export function NodeDetail({ nodeId }: Props) {
  const [node, setNode] = useState<Node | null>(null)
  const [metrics, setMetrics] = useState<MetricsRow[]>([])
  const [processes, setProcesses] = useState<Process[]>([])
  const [services, setServices] = useState<ServiceCheck[]>([])
  const [range, setRange] = useState(3600)

  // Fetch node info
  useEffect(() => {
    api.nodes().then((r) => {
      const found = r.nodes?.find((n) => n.id === nodeId) ?? null
      setNode(found)
    })
  }, [nodeId])

  // Fetch historical metrics when range changes
  useEffect(() => {
    const to = Math.floor(Date.now() / 1000)
    const from = to - range
    api.metrics(nodeId, from, to).then((r) => setMetrics(r.metrics ?? []))
    api.processes(nodeId).then((r) => setProcesses(Array.isArray(r) ? r : []))
    api.services(nodeId).then((r) => setServices(r.services ?? []))
  }, [nodeId, range])

  // Live metrics via WebSocket
  useEffect(() => {
    const token = (import.meta.env.VITE_ADMIN_TOKEN as string) ?? ""
    const ws = getDashboardWS(token)
    const handler = (msg: WSMessage) => {
      if (msg.type === "metrics") {
        const { node_id, data } = msg.payload as { node_id: string; data: MetricsRow & { mem: { used: number; total: number }; net: { rx_bytes: number; tx_bytes: number } } }
        if (node_id !== nodeId) return
        const point: MetricsRow = {
          ts: data.ts ?? Math.floor(Date.now() / 1000),
          cpu: data.cpu,
          mem_used: data.mem?.used ?? 0,
          mem_total: data.mem?.total ?? 0,
          disk_used: data.disk_used ?? 0,
          disk_total: data.disk_total ?? 0,
          net_rx: data.net?.rx_bytes ?? 0,
          net_tx: data.net?.tx_bytes ?? 0,
        }
        setMetrics((prev) => [...prev.slice(-719), point]) // keep max 720 points
      }
    }
    ws.on(handler)
    return () => ws.off(handler)
  }, [nodeId])

  const cpuData = metrics.map((m) => ({ ts: m.ts, value: m.cpu }))
  const memData = metrics.map((m) => ({
    ts: m.ts,
    value: m.mem_total > 0 ? (m.mem_used / m.mem_total) * 100 : 0,
  }))
  const netRxData = metrics.map((m) => ({ ts: m.ts, value: m.net_rx / 1024 }))
  const netTxData = metrics.map((m) => ({ ts: m.ts, value: m.net_tx / 1024 }))

  return (
    <div className="space-y-6 max-w-5xl">
      {/* Node header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold">{node?.name ?? nodeId}</h2>
          <p className="text-xs text-white/40 mt-0.5">
            {node?.ip || "—"} · {node?.os}/{node?.arch}
          </p>
        </div>
        {/* Time range selector */}
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <button
              key={r.label}
              onClick={() => setRange(r.seconds)}
              className={cn(
                "text-xs px-3 py-1.5 rounded-lg border transition-all",
                range === r.seconds
                  ? "bg-white/10 border-white/20 text-white"
                  : "border-white/10 text-white/40 hover:text-white/70 hover:border-white/20"
              )}
            >
              {r.label}
            </button>
          ))}
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <MetricsChart data={cpuData} label="CPU" color="#10b981" />
        <MetricsChart data={memData} label="Memory" color="#3b82f6" />
        <MetricsChart
          data={netRxData}
          label="Network RX"
          color="#a855f7"
          unit=" KB"
          domain={[0, "auto"]}
        />
        <MetricsChart
          data={netTxData}
          label="Network TX"
          color="#f59e0b"
          unit=" KB"
          domain={[0, "auto"]}
        />
      </div>

      {/* Processes */}
      {processes.length > 0 && (
        <div className="bg-white/[0.03] border border-white/10 rounded-xl overflow-hidden">
          <div className="px-4 py-3 border-b border-white/10">
            <p className="text-xs text-white/40 font-medium">Top Processes</p>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-white/30 border-b border-white/5">
                  <th className="text-left px-4 py-2 font-normal">Name</th>
                  <th className="text-right px-4 py-2 font-normal">PID</th>
                  <th className="text-right px-4 py-2 font-normal">CPU%</th>
                  <th className="text-right px-4 py-2 font-normal">Memory</th>
                </tr>
              </thead>
              <tbody>
                {processes.slice(0, 15).map((p) => (
                  <tr key={p.pid} className="border-b border-white/5 hover:bg-white/[0.02]">
                    <td className="px-4 py-2 text-white/80 truncate max-w-[180px]">{p.name}</td>
                    <td className="px-4 py-2 text-right text-white/40">{p.pid}</td>
                    <td className="px-4 py-2 text-right text-white/70">{p.cpu.toFixed(1)}%</td>
                    <td className="px-4 py-2 text-right text-white/70">{formatBytes(p.mem)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Services */}
      {services.length > 0 && (
        <div className="bg-white/[0.03] border border-white/10 rounded-xl overflow-hidden">
          <div className="px-4 py-3 border-b border-white/10">
            <p className="text-xs text-white/40 font-medium">Services</p>
          </div>
          <div className="p-4 flex flex-wrap gap-2">
            {services.map((s) => (
              <div
                key={s.name}
                className={cn(
                  "text-xs px-3 py-1.5 rounded-lg border flex items-center gap-1.5",
                  s.status === "running"
                    ? "bg-emerald-500/10 border-emerald-500/20 text-emerald-400"
                    : s.status === "dead" || s.status === "failed"
                    ? "bg-red-500/10 border-red-500/20 text-red-400"
                    : "bg-white/5 border-white/10 text-white/40"
                )}
              >
                <span
                  className={cn(
                    "w-1.5 h-1.5 rounded-full",
                    s.status === "running" ? "bg-emerald-400" : s.status === "dead" || s.status === "failed" ? "bg-red-400" : "bg-white/30"
                  )}
                />
                {s.name}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
