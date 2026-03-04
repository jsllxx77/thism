import { useState } from "react"
import { Layout } from "./components/Layout"
import { Dashboard } from "./pages/Dashboard"

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
          <p className="text-white/50">Node detail for {selectedNode} — coming in Task 11</p>
        </div>
      ) : (
        <Dashboard onSelectNode={setSelectedNode} />
      )}
    </Layout>
  )
}
