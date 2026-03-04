import { useState } from "react"
import { Layout } from "./components/Layout"

export default function App() {
  const [selectedNode, setSelectedNode] = useState<string | null>(null)

  return (
    <Layout>
      {selectedNode ? (
        <div>
          <button
            onClick={() => setSelectedNode(null)}
            className="text-xs text-white/40 hover:text-white mb-4 flex items-center gap-1"
          >
            ← Back
          </button>
          <p className="text-white/50">Node: {selectedNode} (detail page coming in Task 11)</p>
        </div>
      ) : (
        <p className="text-white/50">Dashboard coming in Task 10</p>
      )}
    </Layout>
  )
}
