import { useCallback, useEffect, useState } from "react"
import { api, type AccessMode, type Node, type MetricsRow, type Process, type ServiceCheck } from "../lib/api"
import { NetworkSummary } from "../components/node-detail/NetworkSummary"
import { formatBytes, formatBytesPerSecond, deriveRateSeries } from "../lib/units"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeHero } from "../components/node-detail/NodeHero"
import { HardwarePassport } from "../components/node-detail/HardwarePassport"
import { MetricTabs } from "../components/node-detail/MetricTabs"
import { ProcessTable } from "../components/node-detail/ProcessTable"
import { ServiceStatusList } from "../components/node-detail/ServiceStatusList"
import { MotionSection } from "../motion/transitions"
import { useLanguage } from "../i18n/language"

type Props = {
  nodeId: string
  refreshNonce?: number
  accessMode?: AccessMode
}

const DESKTOP_BREAKPOINT_QUERY = "(min-width: 768px)"

function isDesktopViewport() {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false
  }

  return window.matchMedia(DESKTOP_BREAKPOINT_QUERY).matches
}

function getLatestValidValue(points: ReadonlyArray<{ value: number }>): number | undefined {
  for (let index = points.length - 1; index >= 0; index -= 1) {
    const value = points[index]?.value
    if (typeof value === "number" && Number.isFinite(value)) {
      return value
    }
  }

  return undefined
}

function formatOptionalBytes(value: number | undefined): string {
  return typeof value === "number" && Number.isFinite(value) ? formatBytes(value) : "—"
}

function formatOptionalBytesPerSecond(value: number | undefined): string {
  return typeof value === "number" && Number.isFinite(value) ? formatBytesPerSecond(value) : "—"
}

export function NodeDetail({ nodeId, refreshNonce = 0, accessMode = "admin" }: Props) {
  const { t } = useLanguage()
  const [node, setNode] = useState<Node | null>(null)
  const [metrics, setMetrics] = useState<MetricsRow[]>([])
  const [processes, setProcesses] = useState<Process[]>([])
  const [services, setServices] = useState<ServiceCheck[]>([])
  const [range, setRange] = useState(3600)
  const [desktopSectionsOpen, setDesktopSectionsOpen] = useState(isDesktopViewport)
  const [loadingDetail, setLoadingDetail] = useState(true)
  const [detailError, setDetailError] = useState<string | null>(null)

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

  const loadNodeDetail = useCallback(async () => {
    setLoadingDetail(true)
    setDetailError(null)

    try {
      const nodesResponse = await api.nodes()
      const found = nodesResponse.nodes?.find((n) => n.id === nodeId) ?? null
      setNode(found)

      if (accessMode === "guest") {
        setMetrics([])
        setProcesses([])
        setServices([])
        return
      }

      const to = Math.floor(Date.now() / 1000)
      const from = to - range
      const [metricsResponse, processesResponse, servicesResponse] = await Promise.all([
        api.metrics(nodeId, from, to),
        api.processes(nodeId),
        api.services(nodeId),
      ])

      setMetrics(metricsResponse.metrics ?? [])
      setProcesses(Array.isArray(processesResponse) ? processesResponse : [])
      setServices(servicesResponse.services ?? [])
    } catch {
      setDetailError(t("nodeDetail.loadError"))
    } finally {
      setLoadingDetail(false)
    }
  }, [accessMode, nodeId, range, t])

  useEffect(() => {
    void loadNodeDetail()
  }, [loadNodeDetail])

  useEffect(() => {
    if (refreshNonce === 0) return
    void loadNodeDetail()
  }, [loadNodeDetail, refreshNonce])

  useEffect(() => {
    if (accessMode === "guest") {
      return
    }

    const ws = getDashboardWS()
    const handler = (msg: WSMessage) => {
      if (msg.type === "metrics") {
        const { node_id, data } = msg.payload as {
          node_id: string
          data: MetricsRow & {
            mem: { used: number; total: number }
            net: { rx_bytes: number; tx_bytes: number }
          }
        }
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
        setMetrics((prev) => [...prev.slice(-719), point])
      }
    }
    ws.on(handler)
    return () => ws.off(handler)
  }, [accessMode, nodeId])

  const cpuData = metrics.map((m) => ({ ts: m.ts, value: m.cpu }))
  const memData = metrics.map((m) => ({
    ts: m.ts,
    value: m.mem_total > 0 ? (m.mem_used / m.mem_total) * 100 : 0,
  }))
  const netRxData = metrics.map((m) => ({ ts: m.ts, value: m.net_rx }))
  const netTxData = metrics.map((m) => ({ ts: m.ts, value: m.net_tx }))
  const netRxSpeedData = deriveRateSeries(netRxData)
  const netTxSpeedData = deriveRateSeries(netTxData)
  const latestNetworkPoint = metrics[metrics.length - 1]
  const latestInboundTotal = formatOptionalBytes(latestNetworkPoint?.net_rx)
  const latestOutboundTotal = formatOptionalBytes(latestNetworkPoint?.net_tx)
  const latestInboundSpeed = formatOptionalBytesPerSecond(getLatestValidValue(netRxSpeedData))
  const latestOutboundSpeed = formatOptionalBytesPerSecond(getLatestValidValue(netTxSpeedData))
  const networkSummary = (
    <NetworkSummary
      inboundTotal={latestInboundTotal}
      outboundTotal={latestOutboundTotal}
      inboundSpeed={latestInboundSpeed}
      outboundSpeed={latestOutboundSpeed}
    />
  )
  const diskData = metrics.map((m) => ({
    ts: m.ts,
    value: m.disk_total > 0 ? (m.disk_used / m.disk_total) * 100 : 0,
  }))
  const hasProcessSection = processes.length > 0
  const hasServiceSection = services.length > 0
  const showMetrics = accessMode !== "guest"
  const showDetailSections = accessMode !== "guest" && (hasProcessSection || hasServiceSection)

  return (
    <MotionSection className="mx-auto max-w-[1440px] space-y-6" delay={0.02}>
      {loadingDetail ? (
        <section className="panel-card rounded-2xl border border-slate-200 p-6 dark:border-slate-700">
          <p className="text-sm font-medium text-slate-700 dark:text-slate-200">{t("nodeDetail.loading")}</p>
          <div className="mt-4 space-y-3">
            <div className="h-4 w-1/4 animate-pulse rounded bg-slate-200/80 dark:bg-slate-700/70" />
            <div className="h-16 animate-pulse rounded-xl bg-slate-200/70 dark:bg-slate-800/70" />
          </div>
        </section>
      ) : detailError ? (
        <section
          role="alert"
          className="panel-card rounded-2xl border border-red-200 bg-red-50/80 p-5 dark:border-red-900/60 dark:bg-red-950/30"
        >
          <p className="text-sm font-medium text-red-700 dark:text-red-300">{detailError}</p>
          <button
            type="button"
            onClick={() => void loadNodeDetail()}
            className="mt-3 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:border-red-800 dark:bg-red-950/50 dark:text-red-200 dark:hover:bg-red-900/40"
          >
            {t("common.retry")}
          </button>
        </section>
      ) : (
        <>
          <NodeHero node={node} showIP={accessMode !== "guest"} />
          <HardwarePassport hardware={node?.hardware} os={node?.os} arch={node?.arch} />
          {showMetrics && (
            <MetricTabs
              range={range}
              onRangeChange={setRange}
              cpuData={cpuData}
              memData={memData}
              netRxData={netRxData}
              netTxData={netTxData}
              netRxSpeedData={netRxSpeedData}
              netTxSpeedData={netTxSpeedData}
              netValueFormatter={formatBytes}
              netAxisTickFormatter={formatBytes}
              netSpeedFormatter={formatBytesPerSecond}
              netSpeedAxisTickFormatter={formatBytesPerSecond}
              networkSummary={networkSummary}
              diskData={diskData}
            />
          )}
          {showDetailSections && (
            <div className={`grid grid-cols-1 gap-4 ${hasProcessSection && hasServiceSection ? "xl:grid-cols-2" : ""}`}>
              {hasProcessSection && <ProcessTable processes={processes} defaultOpen={desktopSectionsOpen} />}
              {hasServiceSection && <ServiceStatusList services={services} defaultOpen={desktopSectionsOpen} />}
            </div>
          )}
        </>
      )}
    </MotionSection>
  )
}
