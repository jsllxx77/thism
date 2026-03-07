import { Suspense, lazy } from "react"
import { Navigate, Route, Routes, useNavigate, useOutletContext, useParams } from "react-router-dom"
import type { AccessMode } from "./lib/api"
import { AppShell } from "./layout/AppShell"
import { Dashboard } from "./pages/Dashboard"

const NodeDetail = lazy(async () => ({ default: (await import("./pages/NodeDetail")).NodeDetail }))
const Settings = lazy(async () => ({ default: (await import("./pages/Settings")).Settings }))
const NotFound = lazy(async () => ({ default: (await import("./pages/NotFound")).NotFound }))

type AppShellOutletContext = {
  refreshNonce: number
  accessMode: AccessMode
}

function RouteFallback() {
  return (
    <div className="panel-card rounded-2xl border border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
      Loading...
    </div>
  )
}

function useShellContext() {
  return useOutletContext<AppShellOutletContext>()
}

function useShellRefreshNonce() {
  return useShellContext().refreshNonce
}

function useShellAccessMode() {
  return useShellContext().accessMode
}

function DashboardRoute() {
  const navigate = useNavigate()
  const refreshNonce = useShellRefreshNonce()
  const accessMode = useShellAccessMode()

  return <Dashboard onSelectNode={(id) => navigate(`/nodes/${id}`)} refreshNonce={refreshNonce} accessMode={accessMode} />
}

function NodeDetailRoute() {
  const { nodeId } = useParams<{ nodeId: string }>()
  const refreshNonce = useShellRefreshNonce()
  const accessMode = useShellAccessMode()

  if (!nodeId) {
    return <Navigate to="/" replace />
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <NodeDetail nodeId={nodeId} refreshNonce={refreshNonce} accessMode={accessMode} />
    </Suspense>
  )
}

function SettingsRoute() {
  const refreshNonce = useShellRefreshNonce()
  const accessMode = useShellAccessMode()

  if (accessMode === "guest") {
    return <Navigate to="/" replace />
  }

  return (
    <Suspense fallback={<RouteFallback />}>
      <Settings refreshNonce={refreshNonce} />
    </Suspense>
  )
}

export default function App() {
  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<DashboardRoute />} />
        <Route path="/nodes/:nodeId" element={<NodeDetailRoute />} />
        <Route path="/settings" element={<SettingsRoute />} />
        <Route
          path="*"
          element={
            <Suspense fallback={<RouteFallback />}>
              <NotFound />
            </Suspense>
          }
        />
      </Route>
    </Routes>
  )
}
