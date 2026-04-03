import { useMemo, useState, type ReactNode } from "react"
import { useLanguage } from "../../i18n/language"
import { MetricsChart, type DataPoint } from "../MetricsChart"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs"

type MetricTab = "cpu" | "memory" | "network" | "disk"

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
  netValueFormatter,
  netAxisTickFormatter,
  netSpeedFormatter,
  netSpeedAxisTickFormatter,
  networkSummary,
}: Props) {
  const { t, language } = useLanguage()
  const [activeTab, setActiveTab] = useState<MetricTab>("cpu")

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
    <section className="space-y-4">
      <div className="panel-card enterprise-surface rounded-[24px] p-4">
        <div className="flex flex-col gap-3">
          <Tabs
            value={activeTab}
            onValueChange={(value) => setActiveTab(value as MetricTab)}
            className="w-full"
          >
            <div className="-mx-1 overflow-x-auto pb-1 md:mx-0 md:pb-0" data-testid="metric-tabs-scroll">
              <TabsList className="enterprise-inner-surface h-11 w-max min-w-full rounded-2xl p-1.5 shadow-none md:h-10 md:w-auto md:min-w-0">
                <TabsTrigger className="h-9 shrink-0 rounded-xl border border-transparent px-4 text-xs text-slate-600 data-[state=active]:border data-[state=active]:border-slate-200/80 data-[state=active]:bg-slate-50/90 data-[state=active]:text-slate-900 data-[state=active]:shadow-sm dark:text-slate-300 dark:data-[state=active]:border-white/10 dark:data-[state=active]:bg-slate-900 dark:data-[state=active]:text-slate-50 dark:data-[state=active]:ring-1 dark:data-[state=active]:ring-inset dark:data-[state=active]:ring-white/10 dark:data-[state=active]:shadow-none md:h-8 md:px-3 md:text-sm" value="cpu">{t("nodeDetail.cpuUsage")}</TabsTrigger>
                <TabsTrigger className="h-9 shrink-0 rounded-xl border border-transparent px-4 text-xs text-slate-600 data-[state=active]:border data-[state=active]:border-slate-200/80 data-[state=active]:bg-slate-50/90 data-[state=active]:text-slate-900 data-[state=active]:shadow-sm dark:text-slate-300 dark:data-[state=active]:border-white/10 dark:data-[state=active]:bg-slate-900 dark:data-[state=active]:text-slate-50 dark:data-[state=active]:ring-1 dark:data-[state=active]:ring-inset dark:data-[state=active]:ring-white/10 dark:data-[state=active]:shadow-none md:h-8 md:px-3 md:text-sm" value="memory">{t("nodeDetail.memoryUsage")}</TabsTrigger>
                <TabsTrigger className="h-9 shrink-0 rounded-xl border border-transparent px-4 text-xs text-slate-600 data-[state=active]:border data-[state=active]:border-slate-200/80 data-[state=active]:bg-slate-50/90 data-[state=active]:text-slate-900 data-[state=active]:shadow-sm dark:text-slate-300 dark:data-[state=active]:border-white/10 dark:data-[state=active]:bg-slate-900 dark:data-[state=active]:text-slate-50 dark:data-[state=active]:ring-1 dark:data-[state=active]:ring-inset dark:data-[state=active]:ring-white/10 dark:data-[state=active]:shadow-none md:h-8 md:px-3 md:text-sm" value="network">{t("nodeDetail.networkTraffic")}</TabsTrigger>
                <TabsTrigger className="h-9 shrink-0 rounded-xl border border-transparent px-4 text-xs text-slate-600 data-[state=active]:border data-[state=active]:border-slate-200/80 data-[state=active]:bg-slate-50/90 data-[state=active]:text-slate-900 data-[state=active]:shadow-sm dark:text-slate-300 dark:data-[state=active]:border-white/10 dark:data-[state=active]:bg-slate-900 dark:data-[state=active]:text-slate-50 dark:data-[state=active]:ring-1 dark:data-[state=active]:ring-inset dark:data-[state=active]:ring-white/10 dark:data-[state=active]:shadow-none md:h-8 md:px-3 md:text-sm" value="disk">{t("nodeDetail.diskUsage")}</TabsTrigger>
              </TabsList>
            </div>
            <TabsContent value="cpu" className="hidden" />
            <TabsContent value="memory" className="hidden" />
            <TabsContent value="network" className="hidden" />
            <TabsContent value="disk" className="hidden" />
          </Tabs>
        </div>
      </div>

      {activeTab === "cpu" && <MetricsChart data={cpuData} label={t("nodeDetail.cpuUsage")} color="#4f78bf" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />}
      {activeTab === "memory" && <MetricsChart data={memData} label={t("nodeDetail.memoryUsage")} color="#4b8b6a" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />}
      {activeTab === "network" && (
        <div className="space-y-4">
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
        </div>
      )}
      {activeTab === "disk" && <MetricsChart data={diskData} label={t("nodeDetail.diskUsage")} color="#8b6d3f" xAxisTickFormatter={xAxisTickFormatter} tooltipLabelFormatter={tooltipLabelFormatter} />}
    </section>
  )
}
