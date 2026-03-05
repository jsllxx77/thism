import { useEffect, useState } from "react"
import { api } from "../lib/api"
import type { Node } from "../lib/api"
import { AddNodeModal } from "../components/AddNodeModal"
import { Plus } from "lucide-react"
import { NodesTable } from "../components/settings/NodesTable"
import { MotionSection } from "../motion/transitions"

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
    <MotionSection className="space-y-6 max-w-4xl mx-auto" delay={0.03}>
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

        {nodes.length === 0 ? (
          <div className="glass-panel rounded-xl px-4 py-8 text-center text-white/30 text-sm">
            No nodes registered yet
          </div>
        ) : (
          <NodesTable nodes={nodes} />
        )}
      </div>

      {showModal && (
        <AddNodeModal
          onClose={() => setShowModal(false)}
          onCreated={fetchNodes}
        />
      )}
    </MotionSection>
  )
}
