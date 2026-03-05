import { useEffect, useMemo, useState } from "react"
import { api } from "../lib/api"
import type { Node } from "../lib/api"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeCard } from "../components/NodeCard"
import { Activity } from "lucide-react"
import { OverviewStats } from "../components/dashboard/OverviewStats"
import { NodeFilters } from "../components/dashboard/NodeFilters"
import { ViewModeToggle } from "../components/dashboard/ViewModeToggle"
import { NodeTable } from "../components/dashboard/NodeTable"
import { MotionSection } from "../motion/transitions"

type LiveMetrics = Record<string, { cpu: number; memUsed: number; memTotal: number }>

type Props = {
  onSelectNode: (id: string) => void
}

export function Dashboard({ onSelectNode }: Props) {
  const [nodes, setNodes] = useState<Node[]>([])
  const [live, setLive] = useState<LiveMetrics>({})
  const [statusFilter, setStatusFilter] = useState<"all" | "online" | "offline">("all")
  const [searchFilter, setSearchFilter] = useState("")
  const [viewMode, setViewMode] = useState<"cards" | "table">("cards")

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
  const offlineCount = nodes.length - onlineCount
  const liveValues = Object.values(live)
  const avgCPU = liveValues.length > 0
    ? (liveValues.reduce((a, b) => a + b.cpu, 0) / liveValues.length).toFixed(1)
    : "—"
  const avgMemory = liveValues.length > 0
    ? (liveValues.reduce((sum, point) => {
      if (point.memTotal <= 0) return sum
      return sum + (point.memUsed / point.memTotal) * 100
    }, 0) / liveValues.length)
    : null
  const filteredNodes = useMemo(() => {
    return nodes.filter((node) => {
      if (statusFilter === "online" && !node.online) return false
      if (statusFilter === "offline" && node.online) return false
      if (searchFilter.trim() && !node.name.toLowerCase().includes(searchFilter.trim().toLowerCase())) {
        return false
      }
      return true
    })
  }, [nodes, searchFilter, statusFilter])

  return (
    <MotionSection className="space-y-6 max-w-7xl mx-auto" testId="motion-dashboard-root">
      <OverviewStats
        onlineNodes={onlineCount}
        totalNodes={nodes.length}
        avgCpu={avgCPU === "—" ? null : Number(avgCPU)}
        avgMemory={avgMemory}
        alertCount={offlineCount}
        heartbeatLatencyMs={null}
      />
      <MotionSection
        className="flex flex-col xl:flex-row gap-3 xl:items-center xl:justify-between"
        testId="motion-dashboard-content"
        delay={0.04}
      >
        <NodeFilters
          status={statusFilter}
          search={searchFilter}
          onStatusChange={setStatusFilter}
          onSearchChange={setSearchFilter}
          onReset={() => {
            setStatusFilter("all")
            setSearchFilter("")
          }}
        />
        <ViewModeToggle mode={viewMode} onChange={setViewMode} />
      </MotionSection>

      {/* Node grid */}
      {nodes.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-24 text-white/20">
          <Activity className="w-10 h-10 mb-3" />
          <p className="text-sm">No nodes registered yet</p>
          <p className="text-xs mt-1">Register a node using the API, then start thisM-agent</p>
        </div>
      ) : viewMode === "table" ? (
        <NodeTable nodes={filteredNodes} onSelectNode={onSelectNode} />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filteredNodes.map((node) => (
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
    </MotionSection>
  )
}
