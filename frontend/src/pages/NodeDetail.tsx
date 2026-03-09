import { useCallback, useEffect, useMemo, useState } from "react"
import { api, type AccessMode, type Node, type MetricsRow, type Process, type ServiceCheck } from "../lib/api"
import { NetworkSummary } from "../components/node-detail/NetworkSummary"
import { formatBytes, formatBytesPerSecond } from "../lib/units"
import { appendLiveMetricPoint, buildMetricChartSeries, buildMetricRateChartSeries } from "../lib/metric-series"
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

export function NodeDetail({ nodeId, refreshNonce = 0, accessMode = "admin" }: Props) {
  const { t } = useLanguage()
  const [node, setNode] = useState<Node | null>(null)
  const [metrics, setMetrics] = useState<MetricsRow[]>([])
  const [processes, setProcesses] = useState<Process[]>([])
  const [services, setServices] = useState<ServiceCheck[]>([])
  const [range, setRange] = useState(3600)
  const [metricsRetentionDays, setMetricsRetentionDays] = useState(DEFAULT_METRICS_RETENTION_DAYS)
  const [desktopSectionsOpen, setDesktopSectionsOpen] = useState(isDesktopViewport)
  const [loadingDetail, setLoadingDetail] = useState(true)
  const [detailError, setDetailError] = useState<string | null>(null)
  const maxRange = metricsRetentionDays >= 30 ? THIRTY_DAYS_SECONDS : SEVEN_DAYS_SECONDS
  const effectiveRange = Math.min(range, maxRange)

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
      const from = to - effectiveRange
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
          uptime_seconds: data.uptime_seconds ?? 0,
        }
        setMetrics((prev) => appendLiveMetricPoint(prev, point, effectiveRange))
      }
    }
    ws.on(handler)
    return () => ws.off(handler)
  }, [accessMode, effectiveRange, nodeId])

  const cpuData = useMemo(() => buildMetricChartSeries(metrics, effectiveRange, (row) => row.cpu, "average"), [effectiveRange, metrics])
  const memData = useMemo(() => buildMetricChartSeries(
    metrics,
    effectiveRange,
    (row) => (row.mem_total > 0 ? (row.mem_used / row.mem_total) * 100 : 0),
    "average",
  ), [effectiveRange, metrics])
  const netRxData = useMemo(() => buildMetricChartSeries(metrics, effectiveRange, (row) => row.net_rx, "last"), [effectiveRange, metrics])
  const netTxData = useMemo(() => buildMetricChartSeries(metrics, effectiveRange, (row) => row.net_tx, "last"), [effectiveRange, metrics])
  const netRxSpeedData = useMemo(() => buildMetricRateChartSeries(metrics, effectiveRange, (row) => row.net_rx), [effectiveRange, metrics])
  const netTxSpeedData = useMemo(() => buildMetricRateChartSeries(metrics, effectiveRange, (row) => row.net_tx), [effectiveRange, metrics])
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
  const diskData = useMemo(() => buildMetricChartSeries(
    metrics,
    effectiveRange,
    (row) => (row.disk_total > 0 ? (row.disk_used / row.disk_total) * 100 : 0),
    "average",
  ), [effectiveRange, metrics])
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
          <NodeHero node={node} showIP={accessMode !== "guest"} uptimeSeconds={heroUptimeSeconds} />
          <HardwarePassport hardware={node?.hardware} os={node?.os} arch={node?.arch} />
          {showMetrics && (
            <MetricTabs
              range={range}
              onRangeChange={setRange}
              retentionDays={metricsRetentionDays}
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
