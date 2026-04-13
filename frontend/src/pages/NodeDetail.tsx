import { useCallback, useEffect, useMemo, useState } from "react"
import { api, type AccessMode, type DockerSnapshot, type LatencyMonitor, type LatencyMonitorResult, type MetricsRow, type Node, type Process, type ServiceCheck } from "../lib/api"
import { NetworkSummary } from "../components/node-detail/NetworkSummary"
import { formatBytes, formatBytesPerSecond } from "../lib/units"
import { appendLiveMetricPoint, buildNodeDetailMetricSeries } from "../lib/metric-series"
import { getDashboardWS } from "../lib/ws"
import type { WSMessage } from "../lib/ws"
import { NodeHero } from "../components/node-detail/NodeHero"
import { HardwarePassport } from "../components/node-detail/HardwarePassport"
import { LatencyMonitorChart } from "../components/node-detail/LatencyMonitorChart"
import { MetricTabs } from "../components/node-detail/MetricTabs"
import { ProcessTable } from "../components/node-detail/ProcessTable"
import { ServiceStatusList } from "../components/node-detail/ServiceStatusList"
import { DockerContainerTable } from "../components/node-detail/DockerContainerTable"
import { MotionSection } from "../motion/transitions"
import { useLanguage } from "../i18n/language"

type Props = {
  nodeId: string
  refreshNonce?: number
  accessMode?: AccessMode
}

type LiveDiskStats = {
  used?: number
  total?: number
}

type LiveMetricsMessage = Partial<MetricsRow> & {
  cpu: number
  mem?: { used?: number; total?: number }
  net?: { rx_bytes?: number; tx_bytes?: number }
  disk?: LiveDiskStats[]
}

const DESKTOP_BREAKPOINT_QUERY = "(min-width: 768px)"
const DEFAULT_METRICS_RETENTION_DAYS = 7
const SEVEN_DAYS_SECONDS = 604800
const THIRTY_DAYS_SECONDS = 2592000

function isDesktopViewport() {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false
  }

  return window.matchMedia(DESKTOP_BREAKPOINT_QUERY).matches
}

function getLatestValidValue(points: ReadonlyArray<{ value: number | null }>): number | undefined {
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

function isFiniteNumber(value: unknown): value is number {
  return typeof value === "number" && Number.isFinite(value)
}

function aggregateLiveDiskTotals(disks?: ReadonlyArray<LiveDiskStats>): { diskUsed?: number; diskTotal?: number } {
  if (!Array.isArray(disks) || disks.length === 0) {
    return {}
  }

  let diskUsed = 0
  let diskTotal = 0
  let hasValue = false

  for (const disk of disks) {
    if (isFiniteNumber(disk?.used)) {
      diskUsed += disk.used
      hasValue = true
    }
    if (isFiniteNumber(disk?.total)) {
      diskTotal += disk.total
      hasValue = true
    }
  }

  if (!hasValue) {
    return {}
  }

  return { diskUsed, diskTotal }
}

function resolveLiveDiskTotals(data: LiveMetricsMessage, previous?: MetricsRow): { diskUsed: number; diskTotal: number } {
  if (isFiniteNumber(data.disk_used) && isFiniteNumber(data.disk_total)) {
    return { diskUsed: data.disk_used, diskTotal: data.disk_total }
  }

  const aggregated = aggregateLiveDiskTotals(data.disk)
  if (isFiniteNumber(aggregated.diskUsed) && isFiniteNumber(aggregated.diskTotal) && aggregated.diskTotal > 0) {
    return { diskUsed: aggregated.diskUsed, diskTotal: aggregated.diskTotal }
  }

  if (previous) {
    return { diskUsed: previous.disk_used, diskTotal: previous.disk_total }
  }

  return { diskUsed: 0, diskTotal: 0 }
}

export function NodeDetail({ nodeId, refreshNonce = 0, accessMode = "admin" }: Props) {
  const { t } = useLanguage()
  const [node, setNode] = useState<Node | null>(null)
  const [metrics, setMetrics] = useState<MetricsRow[]>([])
  const [latencyMonitors, setLatencyMonitors] = useState<LatencyMonitor[]>([])
  const [latencyResults, setLatencyResults] = useState<LatencyMonitorResult[]>([])
  const [processes, setProcesses] = useState<Process[]>([])
  const [services, setServices] = useState<ServiceCheck[]>([])
  const [dockerSnapshot, setDockerSnapshot] = useState<DockerSnapshot | null>(null)
  const [range, setRange] = useState(3600)
  const [metricsRetentionDays, setMetricsRetentionDays] = useState(DEFAULT_METRICS_RETENTION_DAYS)
  const [desktopSectionsOpen, setDesktopSectionsOpen] = useState(isDesktopViewport)
  const [loadingDetail, setLoadingDetail] = useState(true)
  const [detailError, setDetailError] = useState<string | null>(null)
  const maxRange = metricsRetentionDays >= 30 ? THIRTY_DAYS_SECONDS : SEVEN_DAYS_SECONDS
  const effectiveRange = Math.min(range, maxRange)
  const ranges = [
    { label: t("nodeDetail.range1h"), seconds: 3600 },
    { label: t("nodeDetail.range6h"), seconds: 21600 },
    { label: t("nodeDetail.range24h"), seconds: 86400 },
    { label: t("nodeDetail.range7d"), seconds: 604800 },
    ...(metricsRetentionDays >= 30 ? [{ label: t("nodeDetail.range30d"), seconds: 2592000 }] : []),
  ]

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

  useEffect(() => {
    if (accessMode === "guest") {
      return
    }

    let cancelled = false

    const loadMetricsRetention = async () => {
      try {
        const response = await api.metricsRetention()
        if (!cancelled && typeof response.retention_days === "number" && Number.isFinite(response.retention_days)) {
          setMetricsRetentionDays(response.retention_days)
        }
      } catch {
        if (!cancelled) {
          setMetricsRetentionDays(DEFAULT_METRICS_RETENTION_DAYS)
        }
      }
    }

    void loadMetricsRetention()

    return () => {
      cancelled = true
    }
  }, [accessMode, refreshNonce])

  useEffect(() => {
    if (range > maxRange) {
      setRange(maxRange)
    }
  }, [maxRange, range])

  const loadNodeDetail = useCallback(async () => {
    setLoadingDetail(true)
    setDetailError(null)

    try {
      const nodeResponse = await api.node(nodeId)
      setNode(nodeResponse.node ?? null)

      if (accessMode === "guest") {
        setMetrics([])
        setLatencyMonitors([])
        setLatencyResults([])
        setProcesses([])
        setServices([])
        setDockerSnapshot(null)
        return
      }

      const to = Math.floor(Date.now() / 1000)
      const from = to - effectiveRange
      const latencyResultsRequest = (api as { latencyResults?: typeof api.latencyResults }).latencyResults
      const [metricsResponse, latencyResponse, processesResponse, servicesResponse, dockerResponse] = await Promise.all([
        api.metrics(nodeId, from, to),
        latencyResultsRequest ? latencyResultsRequest(nodeId, from, to) : Promise.resolve({ monitors: [], results: [] }),
        api.processes(nodeId),
        api.services(nodeId),
        api.docker(nodeId).catch(() => ({ docker_available: false, containers: [] })),
      ])

      setMetrics(metricsResponse.metrics ?? [])
      setLatencyMonitors(latencyResponse.monitors ?? [])
      setLatencyResults(latencyResponse.results ?? [])
      setProcesses(Array.isArray(processesResponse) ? processesResponse : [])
      setServices(servicesResponse.services ?? [])
      setDockerSnapshot(dockerResponse)
    } catch {
      setDetailError(t("nodeDetail.loadError"))
    } finally {
      setLoadingDetail(false)
    }
  }, [accessMode, effectiveRange, nodeId, t])

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
          data: LiveMetricsMessage
        }
        if (node_id !== nodeId) return
        setMetrics((prev) => {
          const previousPoint = prev[prev.length - 1]
          const { diskUsed, diskTotal } = resolveLiveDiskTotals(data, previousPoint)
          const point: MetricsRow = {
            ts: data.ts ?? Math.floor(Date.now() / 1000),
            cpu: data.cpu,
            mem_used: data.mem?.used ?? 0,
            mem_total: data.mem?.total ?? 0,
            disk_used: diskUsed,
            disk_total: diskTotal,
            net_rx: data.net?.rx_bytes ?? 0,
            net_tx: data.net?.tx_bytes ?? 0,
            uptime_seconds: data.uptime_seconds ?? 0,
          }
          return appendLiveMetricPoint(prev, point, effectiveRange)
        })
        return
      }
      if (msg.type === "latency_result") {
        const { node_id, data } = msg.payload as {
          node_id: string
          data: LatencyMonitorResult
        }
        if (node_id !== nodeId) return
        setLatencyResults((prev) => {
          if (!latencyMonitors.some((monitor) => monitor.id === data.monitor_id)) {
            return prev
          }
          const cutoff = Math.floor(Date.now() / 1000) - effectiveRange
          return [...prev.filter((item) => item.ts >= cutoff), data]
        })
      }
    }
    ws.on(handler)
    return () => ws.off(handler)
  }, [accessMode, effectiveRange, latencyMonitors, nodeId])

  const { cpuData, memData, netRxData, netTxData, netRxSpeedData, netTxSpeedData, diskData } = useMemo(
    () => buildNodeDetailMetricSeries(metrics, effectiveRange),
    [effectiveRange, metrics],
  )
  const latestMetricPoint = metrics[metrics.length - 1]
  const heroUptimeSeconds =
    typeof latestMetricPoint?.uptime_seconds === "number" && latestMetricPoint.uptime_seconds > 0
      ? latestMetricPoint.uptime_seconds
      : node?.latest_metrics?.uptime_seconds
  const latestInboundTotal = formatOptionalBytes(latestMetricPoint?.net_rx)
  const latestOutboundTotal = formatOptionalBytes(latestMetricPoint?.net_tx)
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
  const hasProcessSection = processes.length > 0
  const hasServiceSection = services.length > 0
  const hasDockerSection = dockerSnapshot?.docker_available === true
  const showMetrics = accessMode !== "guest"
  const showDetailSections = accessMode !== "guest" && (hasProcessSection || hasServiceSection || hasDockerSection)
  const detailSectionCount = Number(hasProcessSection) + Number(hasServiceSection) + Number(hasDockerSection)
  const detailSectionGridClass =
    detailSectionCount >= 3 ? "xl:grid-cols-2 2xl:grid-cols-3" : detailSectionCount === 2 ? "xl:grid-cols-2" : ""

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
          <NodeHero node={node} showIP={accessMode !== "guest"} uptimeSeconds={heroUptimeSeconds} />
          <HardwarePassport hardware={node?.hardware} os={node?.os} arch={node?.arch} />
          {showMetrics && (
            <>
              <section className="panel-card enterprise-surface rounded-[24px] p-4">
                <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div>
                    <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500 dark:text-slate-400">{t("nodeDetail.rangeLabel")}</p>
                    <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{t("nodeDetail.rangeDescription")}</p>
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {ranges.map((item) => (
                      <button
                        key={item.label}
                        type="button"
                        onClick={() => setRange(item.seconds)}
                        className={`h-10 rounded-lg border px-3 py-1.5 text-xs font-medium ${
                          range === item.seconds
                            ? "border-slate-300 bg-slate-100 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50 dark:shadow-none"
                            : "border-slate-200 bg-white/80 text-slate-600 hover:bg-slate-50 dark:border-white/8 dark:bg-slate-950/80 dark:text-slate-200 dark:hover:bg-slate-900"
                        }`}
                      >
                        {item.label}
                      </button>
                    ))}
                  </div>
                </div>
              </section>
              <MetricTabs
                range={range}
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
              <LatencyMonitorChart monitors={latencyMonitors} results={latencyResults} range={range} />
            </>
          )}
          {showDetailSections && (
            <div className={`grid grid-cols-1 gap-4 ${detailSectionGridClass}`}>
              {hasProcessSection && <ProcessTable processes={processes} defaultOpen={desktopSectionsOpen} />}
              {hasServiceSection && <ServiceStatusList services={services} defaultOpen={desktopSectionsOpen} />}
              {hasDockerSection && <DockerContainerTable containers={dockerSnapshot?.containers ?? []} defaultOpen={desktopSectionsOpen} />}
            </div>
          )}
        </>
      )}
    </MotionSection>
  )
}
