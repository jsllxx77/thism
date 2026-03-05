import { useState } from "react"
import { MetricsChart, type DataPoint } from "../MetricsChart"

type MetricTab = "cpu" | "memory" | "network" | "disk"

const RANGES = [
  { label: "1h", seconds: 3600 },
  { label: "6h", seconds: 21600 },
  { label: "24h", seconds: 86400 },
  { label: "7d", seconds: 604800 },
] as const

type Props = {
  range: number
  onRangeChange: (seconds: number) => void
  cpuData: DataPoint[]
  memData: DataPoint[]
  netRxData: DataPoint[]
  netTxData: DataPoint[]
  diskData: DataPoint[]
}

export function MetricTabs({
  range,
  onRangeChange,
  cpuData,
  memData,
  netRxData,
  netTxData,
  diskData,
}: Props) {
  const [activeTab, setActiveTab] = useState<MetricTab>("cpu")

  return (
    <section className="space-y-4">
      <div className="glass-panel rounded-xl p-3 flex flex-col lg:flex-row lg:items-center gap-3 lg:justify-between">
        <div className="flex gap-2 flex-wrap">
          <button
            onClick={() => setActiveTab("cpu")}
            className={`px-3 py-1.5 rounded-md text-xs ${activeTab === "cpu" ? "bg-emerald-500 text-slate-950 font-medium" : "bg-white/5 text-white/70"}`}
          >
            CPU
          </button>
          <button
            onClick={() => setActiveTab("memory")}
            className={`px-3 py-1.5 rounded-md text-xs ${activeTab === "memory" ? "bg-emerald-500 text-slate-950 font-medium" : "bg-white/5 text-white/70"}`}
          >
            Memory
          </button>
          <button
            onClick={() => setActiveTab("network")}
            className={`px-3 py-1.5 rounded-md text-xs ${activeTab === "network" ? "bg-emerald-500 text-slate-950 font-medium" : "bg-white/5 text-white/70"}`}
          >
            Network
          </button>
          <button
            onClick={() => setActiveTab("disk")}
            className={`px-3 py-1.5 rounded-md text-xs ${activeTab === "disk" ? "bg-emerald-500 text-slate-950 font-medium" : "bg-white/5 text-white/70"}`}
          >
            Disk
          </button>
        </div>

        <div className="flex gap-1 flex-wrap">
          {RANGES.map((item) => (
            <button
              key={item.label}
              onClick={() => onRangeChange(item.seconds)}
              className={`px-3 py-1.5 rounded-md text-xs border ${
                range === item.seconds
                  ? "bg-white/15 border-white/35 text-white"
                  : "border-white/20 text-white/60 hover:text-white/85"
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
      </div>

      {activeTab === "cpu" && <MetricsChart data={cpuData} label="CPU" color="#10b981" />}
      {activeTab === "memory" && <MetricsChart data={memData} label="Memory" color="#3b82f6" />}
      {activeTab === "network" && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <MetricsChart data={netRxData} label="Network RX" color="#a855f7" unit=" KB" domain={[0, "auto"]} />
          <MetricsChart data={netTxData} label="Network TX" color="#f59e0b" unit=" KB" domain={[0, "auto"]} />
        </div>
      )}
      {activeTab === "disk" && <MetricsChart data={diskData} label="Disk" color="#22c55e" />}
    </section>
  )
}
