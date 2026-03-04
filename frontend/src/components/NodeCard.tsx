import type { Node } from "../lib/api"
import { Badge } from "./ui/badge"
import { cn } from "../lib/utils"
import { Cpu, MemoryStick } from "lucide-react"

type Props = {
  node: Node
  cpu?: number
  memUsed?: number
  memTotal?: number
  onClick?: () => void
}

function ProgressBar({ value, max, colorClass }: { value: number; max: number; colorClass: string }) {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0
  return (
    <div className="w-full bg-white/5 rounded-full h-1.5 overflow-hidden">
      <div
        className={cn("h-full rounded-full transition-all duration-500", colorClass)}
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}

function cpuColor(cpu: number) {
  if (cpu > 80) return "bg-red-500"
  if (cpu > 60) return "bg-amber-500"
  return "bg-emerald-500"
}

function memColor(pct: number) {
  if (pct > 80) return "bg-red-500"
  if (pct > 60) return "bg-amber-500"
  return "bg-blue-500"
}

export function NodeCard({ node, cpu = 0, memUsed = 0, memTotal = 0, onClick }: Props) {
  const memPct = memTotal > 0 ? Math.round((memUsed / memTotal) * 100) : 0

  return (
    <div
      onClick={onClick}
      className={cn(
        "relative overflow-hidden cursor-pointer rounded-xl p-4",
        "bg-white/[0.03] border border-white/10",
        "hover:border-white/20 hover:bg-white/[0.05] transition-all duration-200",
        !node.online && "opacity-50 grayscale"
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-4">
        <div className="min-w-0 flex-1 pr-2">
          <h3 className="font-medium text-sm truncate">{node.name}</h3>
          <p className="text-xs text-white/40 mt-0.5 truncate">
            {node.ip || "—"} · {node.os}/{node.arch}
          </p>
        </div>
        {node.online ? (
          <Badge className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 text-[10px] px-2 py-0.5 shrink-0 flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
            Online
          </Badge>
        ) : (
          <Badge className="bg-white/5 text-white/30 border border-white/10 text-[10px] px-2 py-0.5 shrink-0">
            Offline
          </Badge>
        )}
      </div>

      {/* Metrics */}
      <div className="space-y-2.5 text-xs">
        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="flex items-center gap-1.5 text-white/50">
              <Cpu className="w-3 h-3" /> CPU
            </span>
            <span className={cpu > 80 ? "text-red-400" : "text-white/70"}>{cpu.toFixed(1)}%</span>
          </div>
          <ProgressBar value={cpu} max={100} colorClass={cpuColor(cpu)} />
        </div>

        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="flex items-center gap-1.5 text-white/50">
              <MemoryStick className="w-3 h-3" /> MEM
            </span>
            <span className={memPct > 80 ? "text-red-400" : "text-white/70"}>{memPct}%</span>
          </div>
          <ProgressBar value={memUsed} max={memTotal} colorClass={memColor(memPct)} />
        </div>
      </div>

      {/* Last seen for offline nodes */}
      {!node.online && (
        <p className="absolute bottom-2 right-3 text-[10px] text-white/20">
          {new Date(node.last_seen * 1000).toLocaleDateString()}
        </p>
      )}
    </div>
  )
}
