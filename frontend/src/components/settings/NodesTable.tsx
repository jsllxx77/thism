import { useMemo, useState } from "react"
import type { Node } from "../../lib/api"

type Props = {
  nodes: Node[]
}

export function NodesTable({ nodes }: Props) {
  const [statusFilter, setStatusFilter] = useState<"all" | "online" | "offline">("all")
  const [nameAsc, setNameAsc] = useState(false)

  const rows = useMemo(() => {
    const filtered = nodes.filter((node) => {
      if (statusFilter === "online") return node.online
      if (statusFilter === "offline") return !node.online
      return true
    })
    return filtered.sort((left, right) => {
      const value = left.name.localeCompare(right.name)
      return nameAsc ? value : -value
    })
  }, [nameAsc, nodes, statusFilter])

  return (
    <section className="glass-panel rounded-xl p-4 space-y-3">
      <div className="flex flex-col md:flex-row gap-3 md:items-center md:justify-between">
        <label className="text-xs text-white/60 flex items-center gap-2">
          Settings status filter
          <select
            aria-label="Settings status filter"
            value={statusFilter}
            onChange={(event) => setStatusFilter(event.target.value as "all" | "online" | "offline")}
            className="bg-white/10 border border-white/20 rounded-md px-2 py-1.5 text-white"
          >
            <option value="all">All</option>
            <option value="online">Online</option>
            <option value="offline">Offline</option>
          </select>
        </label>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-white/45 border-b border-white/15">
              <th className="py-2 pr-3 font-medium">
                <button onClick={() => setNameAsc((current) => !current)} className="hover:text-white/85">
                  Name
                </button>
              </th>
              <th className="py-2 pr-3 font-medium">IP</th>
              <th className="py-2 pr-3 font-medium">OS / Arch</th>
              <th className="py-2 pr-3 font-medium">Status</th>
              <th className="py-2 pr-3 font-medium">Created</th>
              <th className="py-2 pr-3 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((node) => (
              <tr key={node.id} className="border-b border-white/10 hover:bg-white/5">
                <td className="py-2 pr-3 text-white">{node.name}</td>
                <td className="py-2 pr-3 text-white/70">{node.ip || "—"}</td>
                <td className="py-2 pr-3 text-white/70">
                  {node.os && node.arch ? `${node.os}/${node.arch}` : "—"}
                </td>
                <td className="py-2 pr-3 text-white/70">{node.online ? "Online" : "Offline"}</td>
                <td className="py-2 pr-3 text-white/70">
                  {node.created_at ? new Date(node.created_at * 1000).toLocaleDateString() : "—"}
                </td>
                <td className="py-2 pr-3">
                  <div className="flex gap-1.5">
                    <button className="px-2 py-1 rounded border border-white/20 text-xs text-white/75">View</button>
                    <button className="px-2 py-1 rounded border border-white/20 text-xs text-white/75">Copy</button>
                    <button className="px-2 py-1 rounded border border-red-400/30 text-xs text-red-300">Remove</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}
