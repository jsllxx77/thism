import { useState } from "react"
import { Layout } from "./components/Layout"
import { Dashboard } from "./pages/Dashboard"
import { NodeDetail } from "./pages/NodeDetail"
import { Settings } from "./pages/Settings"

type Page = { type: "dashboard" } | { type: "node"; id: string } | { type: "settings" }

export default function App() {
  const [page, setPage] = useState<Page>({ type: "dashboard" })

  return (
    <Layout onSettings={() => setPage({ type: "settings" })}>
      {page.type === "settings" ? (
        <div>
          <button
            onClick={() => setPage({ type: "dashboard" })}
            className="text-xs text-white/40 hover:text-white mb-4 flex items-center gap-1 transition-colors"
          >
            ← Back to Dashboard
          </button>
          <Settings />
        </div>
      ) : page.type === "node" ? (
        <div>
          <button
            onClick={() => setPage({ type: "dashboard" })}
            className="text-xs text-white/40 hover:text-white mb-4 flex items-center gap-1 transition-colors"
          >
            ← Back to Dashboard
          </button>
          <NodeDetail nodeId={page.id} />
        </div>
      ) : (
        <Dashboard onSelectNode={(id) => setPage({ type: "node", id })} />
      )}
    </Layout>
  )
}
