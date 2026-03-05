import { useMemo, useState } from "react"
import type { Node } from "../../lib/api"

type SortKey = "name" | "status"

type Props = {
  nodes: Node[]
  onSelectNode: (id: string) => void
}

export function NodeTable({ nodes, onSelectNode }: Props) {
  const [sortKey, setSortKey] = useState<SortKey>("status")
  const [sortAsc, setSortAsc] = useState(true)

  const sorted = useMemo(() => {
    const list = [...nodes]
    list.sort((left, right) => {
      let value = 0
      if (sortKey === "name") {
        value = left.name.localeCompare(right.name)
      } else {
        value = Number(right.online) - Number(left.online)
      }
      return sortAsc ? value : -value
    })
    return list
  }, [nodes, sortAsc, sortKey])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc((current) => !current)
      return
    }
    setSortKey(key)
    setSortAsc(true)
  }

  return (
    <div className="glass-panel rounded-xl p-4 overflow-x-auto">
      <p className="text-sm text-white/70 mb-3">Node table view</p>
      <table className="w-full text-sm">
        <thead>
          <tr className="text-left text-white/45 border-b border-white/15">
            <th className="pb-2 pr-3 font-medium">
              <button onClick={() => toggleSort("name")} className="text-left hover:text-white/85">
                Name
              </button>
            </th>
            <th className="pb-2 pr-3 font-medium">IP</th>
            <th className="pb-2 pr-3 font-medium">
              <button onClick={() => toggleSort("status")} className="text-left hover:text-white/85">
                Status
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((node) => (
            <tr
              key={node.id}
              className="border-b border-white/10 hover:bg-white/5 cursor-pointer"
              onClick={() => onSelectNode(node.id)}
            >
              <td className="py-2 pr-3 text-white">{node.name}</td>
              <td className="py-2 pr-3 text-white/70">{node.ip || "—"}</td>
              <td className="py-2 pr-3 text-white/70">{node.online ? "Online" : "Offline"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
