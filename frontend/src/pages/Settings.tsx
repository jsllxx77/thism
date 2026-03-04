import { useEffect, useState } from "react"
import { api } from "../lib/api"
import type { Node } from "../lib/api"
import { AddNodeModal } from "../components/AddNodeModal"
import { Plus } from "lucide-react"

export function Settings() {
  const [nodes, setNodes] = useState<Node[]>([])
  const [showModal, setShowModal] = useState(false)

  const fetchNodes = () => {
    api.nodes().then((r) => setNodes(r.nodes ?? []))
  }

  useEffect(() => {
    fetchNodes()
  }, [])

  return (
    <div className="space-y-6 max-w-4xl mx-auto">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Settings</h2>
      </div>

      {/* Node Management */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="text-sm text-white/60 font-medium">Node Management</h3>
          <button
            onClick={() => setShowModal(true)}
            className="flex items-center gap-1.5 bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 rounded-lg px-3 py-1.5 text-xs font-medium hover:bg-emerald-500/30 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            Add Node
          </button>
        </div>

        <div className="bg-white/[0.03] border border-white/10 rounded-xl overflow-hidden">
          {nodes.length === 0 ? (
            <div className="px-4 py-8 text-center text-white/20 text-sm">
              No nodes registered yet
            </div>
          ) : (
            <table className="w-full text-xs">
              <thead>
                <tr className="text-white/30 border-b border-white/5">
                  <th className="text-left px-4 py-2.5 font-normal">Name</th>
                  <th className="text-left px-4 py-2.5 font-normal">IP</th>
                  <th className="text-left px-4 py-2.5 font-normal">OS / Arch</th>
                  <th className="text-left px-4 py-2.5 font-normal">Status</th>
                  <th className="text-left px-4 py-2.5 font-normal">Created</th>
                </tr>
              </thead>
              <tbody>
                {nodes.map((node) => (
                  <tr key={node.id} className="border-b border-white/5 hover:bg-white/[0.02]">
                    <td className="px-4 py-2.5 text-white/80">{node.name}</td>
                    <td className="px-4 py-2.5 text-white/50">{node.ip || "—"}</td>
                    <td className="px-4 py-2.5 text-white/50">
                      {node.os && node.arch ? `${node.os}/${node.arch}` : "—"}
                    </td>
                    <td className="px-4 py-2.5">
                      {node.online ? (
                        <span className="flex items-center gap-1.5 text-emerald-400">
                          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
                          Online
                        </span>
                      ) : (
                        <span className="text-white/30">Offline</span>
                      )}
                    </td>
                    <td className="px-4 py-2.5 text-white/40">
                      {node.created_at ? new Date(node.created_at * 1000).toLocaleDateString() : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {showModal && (
        <AddNodeModal
          onClose={() => setShowModal(false)}
          onCreated={fetchNodes}
        />
      )}
    </div>
  )
}
