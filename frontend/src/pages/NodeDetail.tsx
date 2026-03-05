import { useEffect, useState } from "react"
import { api } from "../lib/api"
import type { Node, MetricsRow, Process, ServiceCheck } from "../lib/api"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeHero } from "../components/node-detail/NodeHero"
import { MetricTabs } from "../components/node-detail/MetricTabs"
import { ProcessTable } from "../components/node-detail/ProcessTable"
import { ServiceStatusList } from "../components/node-detail/ServiceStatusList"
import { MotionSection } from "../motion/transitions"

type Props = {
  nodeId: string
}

const DESKTOP_BREAKPOINT_QUERY = "(min-width: 768px)"

function isDesktopViewport() {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false
  }

  return window.matchMedia(DESKTOP_BREAKPOINT_QUERY).matches
}

export function NodeDetail({ nodeId }: Props) {
  const [node, setNode] = useState<Node | null>(null)
  const [metrics, setMetrics] = useState<MetricsRow[]>([])
  const [processes, setProcesses] = useState<Process[]>([])
  const [services, setServices] = useState<ServiceCheck[]>([])
  const [range, setRange] = useState(3600)
  const [desktopSectionsOpen, setDesktopSectionsOpen] = useState(isDesktopViewport)

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return
    }

    const mediaQuery = window.matchMedia(DESKTOP_BREAKPOINT_QUERY)
    const onMediaQueryChange = (event: MediaQueryListEvent) => {
      setDesktopSectionsOpen(event.matches)
    }

    setDesktopSectionsOpen(mediaQuery.matches)
    mediaQuery.addEventListener("change", onMediaQueryChange)

    return () => {
      mediaQuery.removeEventListener("change", onMediaQueryChange)
    }
  }, [])

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
  const diskData = metrics.map((m) => ({
    ts: m.ts,
    value: m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
  }))

  return (
    <MotionSection className="space-y-6 max-w-5xl" delay={0.02}>
      <NodeHero node={node} />
      <MetricTabs
        range={range}
        onRangeChange={setRange}
        cpuData={cpuData}
        memData={memData}
        netRxData={netRxData}
        netTxData={netTxData}
        diskData={diskData}
      />
      <ProcessTable processes={processes} defaultOpen={desktopSectionsOpen} />
      <ServiceStatusList services={services} defaultOpen={desktopSectionsOpen} />
    </MotionSection>
  )
}
