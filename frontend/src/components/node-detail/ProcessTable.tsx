import { useEffect, useState } from "react"
import { Collapse } from "antd"
import type { Process } from "../../lib/api"

type Props = {
  processes: Process[]
  defaultOpen?: boolean
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB"]
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

const PANEL_KEY = "processes"

function normalizeActiveKey(key: string | string[]): string[] {
  return Array.isArray(key) ? key : [key]
}

export function ProcessTable({ processes, defaultOpen = false }: Props) {
  const [activeKey, setActiveKey] = useState<string[]>(defaultOpen ? [PANEL_KEY] : [])

  useEffect(() => {
    setActiveKey(defaultOpen ? [PANEL_KEY] : [])
  }, [defaultOpen])

  if (processes.length === 0) {
    return null
  }

  return (
    <Collapse
      activeKey={activeKey}
      destroyOnHidden
      onChange={(key) => setActiveKey(normalizeActiveKey(key))}
      className="glass-panel !border-white/15 !bg-transparent"
      items={[
        {
          key: PANEL_KEY,
          label: "Top Processes",
          children: (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-white/35 border-b border-white/10">
                    <th className="text-left px-2 py-2 font-normal">Name</th>
                    <th className="text-right px-2 py-2 font-normal">PID</th>
                    <th className="text-right px-2 py-2 font-normal">CPU%</th>
                    <th className="text-right px-2 py-2 font-normal">Memory</th>
                  </tr>
                </thead>
                <tbody>
                  {processes.slice(0, 15).map((process) => (
                    <tr key={process.pid} className="border-b border-white/10">
                      <td className="px-2 py-2 text-white/85 truncate max-w-[180px]">{process.name}</td>
                      <td className="px-2 py-2 text-right text-white/55">{process.pid}</td>
                      <td className="px-2 py-2 text-right text-white/75">{process.cpu.toFixed(1)}%</td>
                      <td className="px-2 py-2 text-right text-white/75">{formatBytes(process.mem)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ),
        },
      ]}
    />
  )
}
