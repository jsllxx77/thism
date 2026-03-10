import { Suspense, lazy, useCallback, useEffect, useMemo, useState } from "react"
import { Activity } from "lucide-react"
import { api, type AccessMode, type Node } from "../lib/api"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeCard } from "../components/NodeCard"
import { OverviewStats } from "../components/dashboard/OverviewStats"
import { NodeFilters } from "../components/dashboard/NodeFilters"
import { ViewModeToggle } from "../components/dashboard/ViewModeToggle"
import { MotionSection } from "../motion/transitions"
import { useLanguage } from "../i18n/language"

type LiveNetSample = { ts: number; rxBytes: number; txBytes: number }
type LiveMetricsSample = {
  cpu: number
  memUsed: number
  memTotal: number
  netRxSpeed?: number
  netTxSpeed?: number
  lastNet?: LiveNetSample
}
type LiveMetrics = Record<string, LiveMetricsSample>
const ONLINE_GRACE_PERIOD_SECONDS = 15
const NodeTable = lazy(async () => ({ default: (await import("../components/dashboard/NodeTable")).NodeTable }))

function snapshotToLive(node: Node): LiveMetrics[string] | null {
  const snapshot = node.latest_metrics
  if (!snapshot) return null

  const entry: LiveMetrics[string] = {
    cpu: snapshot.cpu,
    memUsed: snapshot.mem_used,
    memTotal: snapshot.mem_total,
  }

  if (typeof snapshot.ts === "number" && typeof (snapshot as any).net_rx === "number" && typeof (snapshot as any).net_tx === "number") {
    entry.lastNet = { ts: snapshot.ts, rxBytes: (snapshot as any).net_rx, txBytes: (snapshot as any).net_tx }
  }

  return entry
}

function isNodeEffectivelyOnline(node: Node, nowMs: number): boolean {
  if (node.online) return true
  if (node.last_seen <= 0) return false

  return Math.max(0, Math.floor(nowMs / 1000) - node.last_seen) <= ONLINE_GRACE_PERIOD_SECONDS
}

type Props = {
  onSelectNode: (id: string) => void
  refreshNonce?: number
  accessMode?: AccessMode
}

export function Dashboard({ onSelectNode, refreshNonce = 0, accessMode = "admin" }: Props) {
  const { t } = useLanguage()
  const [nodes, setNodes] = useState<Node[]>([])
  const [live, setLive] = useState<LiveMetrics>({})
  const [nowMs, setNowMs] = useState(() => Date.now())
  const [statusFilter, setStatusFilter] = useState<"all" | "online" | "offline">("all")
  const [searchFilter, setSearchFilter] = useState("")
  const [viewMode, setViewMode] = useState<"cards" | "table">("cards")
  const [loadingNodes, setLoadingNodes] = useState(true)
  const [nodesError, setNodesError] = useState<string | null>(null)

  const loadNodes = useCallback(async () => {
    setLoadingNodes(true)
    setNodesError(null)

    try {
      const response = await api.nodes()
      const nextNodes = response.nodes ?? []
      setNodes(nextNodes)
      setLive((prev) => {
        const next: LiveMetrics = {}
        const validNodeIDs = new Set(nextNodes.map((node) => node.id))

        for (const [nodeID, metrics] of Object.entries(prev)) {
          if (validNodeIDs.has(nodeID)) {
            next[nodeID] = metrics
          }
        }

        for (const node of nextNodes) {
          if (next[node.id]) continue
          const metrics = snapshotToLive(node)
          if (metrics) {
            next[node.id] = metrics
          }
        }

        return next
      })
    } catch {
      setNodesError(t("dashboard.loadError"))
    } finally {
      setLoadingNodes(false)
    }
  }, [t])

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [])

  useEffect(() => {
    void loadNodes()

    const ws = getDashboardWS()

    const handler = (msg: WSMessage) => {
      if (msg.type === "metrics") {
        const { node_id, last_seen, data } = msg.payload as {
          node_id: string
          last_seen?: number
          data: { cpu: number; mem: { used: number; total: number }; ts?: number; net?: { rx_bytes: number; tx_bytes: number } }
        }
        setLive((prev) => {
          const existing = prev[node_id]
          const next: LiveMetrics[string] = existing
            ? { ...existing, cpu: data.cpu, memUsed: data.mem.used, memTotal: data.mem.total }
            : { cpu: data.cpu, memUsed: data.mem.used, memTotal: data.mem.total }

          if (typeof data.ts === "number" && data.net && typeof data.net.rx_bytes === "number" && typeof data.net.tx_bytes === "number") {
            const ts = data.ts
            const rxBytes = data.net.rx_bytes
            const txBytes = data.net.tx_bytes

            if (existing?.lastNet && ts > existing.lastNet.ts) {
              const deltaTs = ts - existing.lastNet.ts
              const deltaRx = rxBytes - existing.lastNet.rxBytes
              const deltaTx = txBytes - existing.lastNet.txBytes
              next.netRxSpeed = deltaTs > 0 && deltaRx >= 0 ? deltaRx / deltaTs : undefined
              next.netTxSpeed = deltaTs > 0 && deltaTx >= 0 ? deltaTx / deltaTs : undefined
            }

            next.lastNet = { ts, rxBytes, txBytes }
          }

          return {
            ...prev,
            [node_id]: next,
          }
        })
        if (typeof last_seen === "number") {
          setNodes((prev) => prev.map((n) => (n.id === node_id ? { ...n, last_seen } : n)))
        }
      }
      if (msg.type === "node_status") {
        const { node_id, online } = msg.payload as { node_id: string; online: boolean }
        setNodes((prev) => prev.map((n) => (n.id === node_id ? { ...n, online } : n)))
      }
    }

    ws.on(handler)
    return () => ws.off(handler)
  }, [loadNodes])

  const effectiveNodes = useMemo(
    () => nodes.map((node) => ({ ...node, online: isNodeEffectivelyOnline(node, nowMs) })),
    [nodes, nowMs],
  )

  const onlineCount = effectiveNodes.filter((n) => n.online).length
  const offlineCount = effectiveNodes.length - onlineCount
  const liveValues = Object.values(live)
  const avgCPU = liveValues.length > 0
    ? (liveValues.reduce((a, b) => a + b.cpu, 0) / liveValues.length).toFixed(1)
    : "—"
  const validMemorySamples = liveValues.filter((point) => point.memTotal > 0)
  const avgMemory = validMemorySamples.length > 0
    ? (validMemorySamples.reduce((sum, point) => sum + (point.memUsed / point.memTotal) * 100, 0) / validMemorySamples.length)
    : null

  useEffect(() => {
    if (refreshNonce === 0) return
    void loadNodes()
  }, [loadNodes, refreshNonce])

  const filteredNodes = useMemo(() => {
    return effectiveNodes.filter((node) => {
      if (statusFilter === "online" && !node.online) return false
      if (statusFilter === "offline" && node.online) return false
      if (searchFilter.trim() && !node.name.toLowerCase().includes(searchFilter.trim().toLowerCase())) {
        return false
      }
      return true
    })
  }, [effectiveNodes, searchFilter, statusFilter])

  const showTable = accessMode !== "guest" && viewMode === "table"

  return (
    <MotionSection className="mx-auto max-w-[1440px] space-y-6" testId="motion-dashboard-root">
      <section className="space-y-2">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t("dashboard.title")}</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400">{t("dashboard.subtitle")}</p>
      </section>

      {loadingNodes ? (
        <section className="panel-card rounded-2xl border border-slate-200 p-6 dark:border-slate-700">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("dashboard.loadingInventory")}</p>
          <div className="mt-4 space-y-3">
            <div className="h-4 w-1/3 animate-pulse rounded bg-slate-200/80 dark:bg-slate-700/70" />
            <div className="h-16 animate-pulse rounded-xl bg-slate-200/70 dark:bg-slate-800/70" />
            <div className="h-16 animate-pulse rounded-xl bg-slate-200/70 dark:bg-slate-800/70" />
          </div>
        </section>
      ) : nodesError ? (
        <section
          role="alert"
          className="panel-card rounded-2xl border border-red-200 bg-red-50/80 p-5 dark:border-red-900/60 dark:bg-red-950/30"
        >
          <p className="text-sm font-medium text-red-700 dark:text-red-300">{nodesError}</p>
          <button
            type="button"
            onClick={() => void loadNodes()}
            className="mt-3 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:border-red-800 dark:bg-red-950/50 dark:text-red-200 dark:hover:bg-red-900/40"
          >
            {t("common.retry")}
          </button>
        </section>
      ) : (
        <>
          <OverviewStats
            onlineNodes={onlineCount}
            totalNodes={effectiveNodes.length}
            avgCpu={avgCPU === "—" ? null : Number(avgCPU)}
            avgMemory={avgMemory}
            alertCount={offlineCount}
            heartbeatLatencyMs={null}
          />
          <MotionSection
            className="space-y-3"
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
            {accessMode !== "guest" && (
              <div className="flex justify-stretch md:justify-end">
                <ViewModeToggle mode={viewMode} onChange={setViewMode} />
              </div>
            )}
          </MotionSection>

          {effectiveNodes.length === 0 ? (
            <div className="panel-card flex flex-col items-center justify-center rounded-2xl border border-slate-200 py-24 text-slate-400 dark:border-slate-700 dark:text-slate-500">
              <Activity className="mb-3 h-10 w-10" />
              <p className="text-sm">{t("dashboard.noNodesRegistered")}</p>
              <p className="mt-1 text-xs">{t("dashboard.registrationHint")}</p>
            </div>
          ) : filteredNodes.length === 0 ? (
            <section className="panel-card rounded-2xl border border-slate-200 px-6 py-14 text-center dark:border-slate-700">
              <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("dashboard.noNodesMatch")}</p>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("dashboard.adjustFilters")}</p>
            </section>
          ) : showTable ? (
            <Suspense
              fallback={
                <div className="panel-card rounded-[24px] border border-slate-200/80 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                  {t("common.loading")}...
                </div>
              }
            >
              <NodeTable nodes={filteredNodes} onSelectNode={onSelectNode} />
            </Suspense>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {filteredNodes.map((node) => (
                <NodeCard
                  key={node.id}
                  node={node}
                  cpu={live[node.id]?.cpu}
                  memUsed={live[node.id]?.memUsed}
                  memTotal={live[node.id]?.memTotal}
                  netRxSpeed={live[node.id]?.netRxSpeed}
                  netTxSpeed={live[node.id]?.netTxSpeed}
                  showIP={accessMode !== "guest"}
                  onClick={() => onSelectNode(node.id)}
                />
              ))}
            </div>
          )}
        </>
      )}
    </MotionSection>
  )
}
