import { Suspense, lazy } from "react"
import { Navigate, Route, Routes, useNavigate, useOutletContext, useParams } from "react-router-dom"
import type { AccessMode } from "./lib/api"
import { AppShell } from "./layout/AppShell"
import { useThemeRegistry } from "./theme-plugin/runtime"
import { Dashboard, type DashboardCache } from "./pages/Dashboard"

const NodeDetail = lazy(async () => ({ default: (await import("./pages/NodeDetail")).NodeDetail }))
const Settings = lazy(async () => ({ default: (await import("./pages/Settings")).Settings }))
const Reports = lazy(async () => ({ default: (await import("./pages/Reports")).Reports }))
const NotFound = lazy(async () => ({ default: (await import("./pages/NotFound")).NotFound }))

type AppShellOutletContext = {
  refreshNonce: number
  accessMode: AccessMode
  dashboardCache: DashboardCache | null
  setDashboardCache: (cache: DashboardCache) => void
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
  const { refreshNonce, accessMode, dashboardCache, setDashboardCache } = useShellContext()

  return <Dashboard onSelectNode={(id) => navigate(`/nodes/${id}`)} refreshNonce={refreshNonce} accessMode={accessMode} initialCache={dashboardCache} onCacheChange={setDashboardCache} />
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

function ReportsRoute() {
  return (
    <Suspense fallback={<RouteFallback />}>
      <Reports />
    </Suspense>
  )
}

function PluginBackedAppShell() {
  const RootShell = useThemeRegistry().shells.RootShell
  return <RootShell><AppShell /></RootShell>
}

export default function App() {
  return (
    <Routes>
      <Route element={<PluginBackedAppShell />}>
        <Route index element={<DashboardRoute />} />
        <Route path="/nodes/:nodeId" element={<NodeDetailRoute />} />
        <Route path="/reports" element={<ReportsRoute />} />
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
