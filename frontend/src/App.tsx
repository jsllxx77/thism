import { useState } from "react"
import { Layout } from "./components/Layout"
import { Dashboard } from "./pages/Dashboard"
import { NodeDetail } from "./pages/NodeDetail"

export default function App() {
  const [selectedNode, setSelectedNode] = useState<string | null>(null)

  return (
    <Layout>
      {selectedNode ? (
        <div>
          <button
            onClick={() => setSelectedNode(null)}
            className="text-xs text-white/40 hover:text-white mb-4 flex items-center gap-1 transition-colors"
          >
            ← Back to Dashboard
          </button>
          <NodeDetail nodeId={selectedNode} />
        </div>
      ) : (
        <Dashboard onSelectNode={setSelectedNode} />
      )}
    </Layout>
  )
}
