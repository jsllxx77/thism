import { useMemo, type ReactNode } from "react"
import { useLanguage } from "../../i18n/language"
import { MetricsChart, type DataPoint } from "../MetricsChart"

type ValueFormatter = (value: number) => string

type Props = {
  range: number
  cpuData: DataPoint[]
  memData: DataPoint[]
  netRxData: DataPoint[]
  netTxData: DataPoint[]
  netRxSpeedData: DataPoint[]
  netTxSpeedData: DataPoint[]
  diskData: DataPoint[]
  diskReadSpeedData: DataPoint[]
  diskWriteSpeedData: DataPoint[]
  netValueFormatter?: ValueFormatter
  netAxisTickFormatter?: ValueFormatter
  netSpeedFormatter?: ValueFormatter
  netSpeedAxisTickFormatter?: ValueFormatter
  networkSummary?: ReactNode
}

export function MetricTabs({
  range,
  cpuData,
  memData,
  netRxData,
  netTxData,
  netRxSpeedData,
  netTxSpeedData,
  diskData,
  diskReadSpeedData,
  diskWriteSpeedData,
  netValueFormatter,
  netAxisTickFormatter,
  netSpeedFormatter,
  netSpeedAxisTickFormatter,
  networkSummary,
}: Props) {
  const { t, language } = useLanguage()

  const showDateInTimeLabels = range >= 86400
  const xAxisTickFormatter = useMemo(() => {
    if (showDateInTimeLabels) {
      return (value: number) =>
        new Date(value * 1000).toLocaleString(language, { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" })
    }

    return (value: number) =>
      new Date(value * 1000).toLocaleTimeString(language, { hour: "2-digit", minute: "2-digit" })
  }, [language, showDateInTimeLabels])
  const tooltipLabelFormatter = useMemo(() => {
    if (showDateInTimeLabels) {
      return (value: number) =>
        new Date(value * 1000).toLocaleString(language, {
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })
    }

    return (value: number) => new Date(value * 1000).toLocaleTimeString(language)
  }, [language, showDateInTimeLabels])
  return (
    <section className="space-y-5">
      <section className="space-y-3" aria-labelledby="resource-usage-heading">
        <div className="flex items-center justify-between">
          <h3 id="resource-usage-heading" className="text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t("nodeDetail.resourceUsage")}
          </h3>
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <MetricsChart data={cpuData} label={t("nodeDetail.cpuUsage")} color="#4f78bf" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />
          <MetricsChart data={memData} label={t("nodeDetail.memoryUsage")} color="#4b8b6a" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />
          <MetricsChart data={diskData} label={t("nodeDetail.diskUsage")} color="#8b6d3f" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />
        </div>
      </section>

      <section className="space-y-3" aria-labelledby="throughput-heading">
        <div className="flex items-center justify-between">
          <h3 id="throughput-heading" className="text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t("nodeDetail.throughputTraffic")}
          </h3>
        </div>
        <div className="space-y-4">
          <section className="space-y-4" aria-label={t("nodeDetail.networkTraffic")}>
            {networkSummary}
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <MetricsChart
                data={netRxData}
                label={t("nodeDetail.inboundTraffic")}
                color="#4f78bf"
                domain={[0, "auto"]}
                valueFormatter={netValueFormatter}
                axisTickFormatter={netAxisTickFormatter}
                xAxisTickFormatter={xAxisTickFormatter}
                tooltipLabelFormatter={tooltipLabelFormatter}
              />
              <MetricsChart
                data={netTxData}
                label={t("nodeDetail.outboundTraffic")}
                color="#4b8b6a"
                domain={[0, "auto"]}
                valueFormatter={netValueFormatter}
                axisTickFormatter={netAxisTickFormatter}
                xAxisTickFormatter={xAxisTickFormatter}
                tooltipLabelFormatter={tooltipLabelFormatter}
              />
              <MetricsChart
                data={netRxSpeedData}
                label={t("nodeDetail.inboundSpeed")}
                color="#4f78bf"
                domain={[0, "auto"]}
                valueFormatter={netSpeedFormatter}
                axisTickFormatter={netSpeedAxisTickFormatter}
                xAxisTickFormatter={xAxisTickFormatter}
                tooltipLabelFormatter={tooltipLabelFormatter}
              />
              <MetricsChart
                data={netTxSpeedData}
                label={t("nodeDetail.outboundSpeed")}
                color="#4b8b6a"
                domain={[0, "auto"]}
                valueFormatter={netSpeedFormatter}
                axisTickFormatter={netSpeedAxisTickFormatter}
                xAxisTickFormatter={xAxisTickFormatter}
                tooltipLabelFormatter={tooltipLabelFormatter}
              />
            </div>
          </section>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2" aria-label={t("nodeDetail.diskIo")}>
            <MetricsChart
              data={diskReadSpeedData}
              label={t("nodeDetail.diskReadSpeed")}
              color="#8b6d3f"
              domain={[0, "auto"]}
              valueFormatter={netSpeedFormatter}
              axisTickFormatter={netSpeedAxisTickFormatter}
              xAxisTickFormatter={xAxisTickFormatter}
              tooltipLabelFormatter={tooltipLabelFormatter}
            />
            <MetricsChart
              data={diskWriteSpeedData}
              label={t("nodeDetail.diskWriteSpeed")}
              color="#b45d4f"
              domain={[0, "auto"]}
              valueFormatter={netSpeedFormatter}
              axisTickFormatter={netSpeedAxisTickFormatter}
              xAxisTickFormatter={xAxisTickFormatter}
              tooltipLabelFormatter={tooltipLabelFormatter}
            />
          </div>
        </div>
      </section>
    </section>
  )
}
