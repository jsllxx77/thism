import { useEffect, useState } from "react"
import { api } from "../lib/api"
import type { Node } from "../lib/api"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeCard } from "../components/NodeCard"
import { Activity } from "lucide-react"

type LiveMetrics = Record<string, { cpu: number; memUsed: number; memTotal: number }>

type Props = {
  onSelectNode: (id: string) => void
}

export function Dashboard({ onSelectNode }: Props) {
  const [nodes, setNodes] = useState<Node[]>([])
  const [live, setLive] = useState<LiveMetrics>({})

  useEffect(() => {
    api.nodes().then((r) => setNodes(r.nodes ?? []))

    const token = (import.meta.env.VITE_ADMIN_TOKEN as string) ?? ""
    const ws = getDashboardWS(token)

    const handler = (msg: WSMessage) => {
      if (msg.type === "metrics") {
        const { node_id, data } = msg.payload as {
          node_id: string
          data: { cpu: number; mem: { used: number; total: number } }
        }
        setLive((prev) => ({
          ...prev,
          [node_id]: { cpu: data.cpu, memUsed: data.mem.used, memTotal: data.mem.total },
        }))
      }
      if (msg.type === "node_status") {
        const { node_id, online } = msg.payload as { node_id: string; online: boolean }
        setNodes((prev) => prev.map((n) => (n.id === node_id ? { ...n, online } : n)))
      }
    }

    ws.on(handler)
    return () => ws.off(handler)
  }, [])

  const onlineCount = nodes.filter((n) => n.online).length
  const liveValues = Object.values(live)
  const avgCPU = liveValues.length > 0
    ? (liveValues.reduce((a, b) => a + b.cpu, 0) / liveValues.length).toFixed(1)
    : "—"

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Summary bar */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-white/[0.03] border border-white/10 rounded-xl px-4 py-3">
          <p className="text-xs text-white/40">Nodes</p>
          <p className="text-2xl font-semibold mt-1 tracking-tight">
            {onlineCount}
            <span className="text-white/30 text-base font-normal"> / {nodes.length}</span>
          </p>
        </div>
        <div className="bg-white/[0.03] border border-white/10 rounded-xl px-4 py-3">
          <p className="text-xs text-white/40">Avg CPU</p>
          <p className="text-2xl font-semibold mt-1 tracking-tight">
            {avgCPU}{typeof avgCPU === "string" && avgCPU !== "—" ? "%" : ""}
          </p>
        </div>
      </div>

      {/* Node grid */}
      {nodes.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-white/20">
          <Activity className="w-10 h-10 mb-3" />
          <p className="text-sm">No nodes registered yet</p>
          <p className="text-xs mt-1">Register a node using the API, then start thisM-agent</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {nodes.map((node) => (
            <NodeCard
              key={node.id}
              node={node}
              cpu={live[node.id]?.cpu}
              memUsed={live[node.id]?.memUsed}
              memTotal={live[node.id]?.memTotal}
              onClick={() => onSelectNode(node.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
