import { Card, Progress, Tag } from "antd"
import { Cpu, MemoryStick } from "lucide-react"
import type { Node } from "../lib/api"

type Props = {
  node: Node
  cpu?: number
  memUsed?: number
  memTotal?: number
  onClick?: () => void
}

function metricColor(value: number): string {
  if (value > 80) return "#f43f5e"
  if (value > 60) return "#f59e0b"
  return "#34d399"
}

export function NodeCard({ node, cpu = 0, memUsed = 0, memTotal = 0, onClick }: Props) {
  const memPct = memTotal > 0 ? (memUsed / memTotal) * 100 : 0

  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={node.name}
      className={`w-full text-left rounded-xl border-0 p-0 bg-transparent transition-transform hover:-translate-y-0.5 ${
        !node.online ? "opacity-70" : ""
      }`}
    >
      <Card className="glass-panel glass-panel-hover !border-white/15 !rounded-xl [&_.ant-card-body]:p-4">
        <div className="flex items-start justify-between mb-4 gap-3">
          <div className="min-w-0">
            <h3 className="font-medium text-sm text-white truncate">{node.name}</h3>
            <p className="text-xs text-white/45 mt-0.5 truncate">
              {node.ip || "—"} · {node.os}/{node.arch}
            </p>
          </div>
          <Tag
            color={node.online ? "green" : "default"}
            className="!m-0 !font-medium !text-[11px] !px-2 !py-0.5"
          >
            {node.online ? "Online" : "Offline"}
          </Tag>
        </div>

        <div className="space-y-3 text-xs">
          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="flex items-center gap-1.5 text-white/55">
                <Cpu className="w-3 h-3" /> CPU
              </span>
              <span className="text-white/85">{cpu.toFixed(1)}%</span>
            </div>
            <Progress
              percent={Number(cpu.toFixed(1))}
              showInfo={false}
              size={["100%", 6]}
              strokeColor={metricColor(cpu)}
              trailColor="rgba(255,255,255,0.12)"
            />
          </div>

          <div>
            <div className="flex items-center justify-between mb-1">
              <span className="flex items-center gap-1.5 text-white/55">
                <MemoryStick className="w-3 h-3" /> MEM
              </span>
              <span className="text-white/85">{memPct.toFixed(1)}%</span>
            </div>
            <Progress
              percent={Number(memPct.toFixed(1))}
              showInfo={false}
              size={["100%", 6]}
              strokeColor={metricColor(memPct)}
              trailColor="rgba(255,255,255,0.12)"
            />
          </div>
        </div>

        {!node.online && (
          <p className="mt-3 text-[11px] text-white/45">
            Last seen {new Date(node.last_seen * 1000).toLocaleString()}
          </p>
        )}
      </Card>
    </button>
  )
}
