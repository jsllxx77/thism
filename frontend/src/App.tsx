import { Navigate, Route, Routes, useNavigate, useParams } from "react-router-dom"
import { AppShell } from "./layout/AppShell"
import { Dashboard } from "./pages/Dashboard"
import { NodeDetail } from "./pages/NodeDetail"
import { Settings } from "./pages/Settings"

function DashboardRoute() {
  const navigate = useNavigate()

  return <Dashboard onSelectNode={(id) => navigate(`/nodes/${id}`)} />
}

function NodeDetailRoute() {
  const { nodeId } = useParams<{ nodeId: string }>()

  if (!nodeId) {
    return <Navigate to="/" replace />
  }

  return <NodeDetail nodeId={nodeId} />
}

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<DashboardRoute />} />
        <Route path="/nodes/:nodeId" element={<NodeDetailRoute />} />
        <Route path="/settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
