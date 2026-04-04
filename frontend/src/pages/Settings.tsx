import { Suspense, lazy, useCallback, useEffect, useMemo, useState, type FormEvent } from "react"
import { Plus } from "lucide-react"
import { AgentAutoUpdateCard } from "../components/settings/AgentAutoUpdateCard"
import { DashboardVisibilityCard } from "../components/settings/DashboardVisibilityCard"
import { LatencyMonitorsCard } from "../components/settings/LatencyMonitorsCard"
import { MetricsRetentionCard } from "../components/settings/MetricsRetentionCard"
import { NodesTable } from "../components/settings/NodesTable"
import { NotificationsCard } from "../components/settings/NotificationsCard"
import { Button } from "../components/ui/button"
import { Input } from "../components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs"
import { useLanguage } from "../i18n/language"
import { api } from "../lib/api"
import type { Node, VersionMeta } from "../lib/api"
import { withEffectiveNodeStatus } from "../lib/node-status"
import { MotionSection } from "../motion/transitions"

type Props = {
  refreshNonce?: number
}

const AddNodeModal = lazy(async () => ({ default: (await import("../components/AddNodeModal")).AddNodeModal }))

const settingsSections = ["nodes", "agent", "monitoring", "alerts", "security"] as const

type SettingsSection = (typeof settingsSections)[number]

function isSettingsSection(value: string | null): value is SettingsSection {
  return value !== null && settingsSections.includes(value as SettingsSection)
}

function resolveSettingsSection(search: string): { section: SettingsSection; normalizedSearch: string } {
  const params = new URLSearchParams(search)
  const rawSection = params.get("section")
  const section = isSettingsSection(rawSection) ? rawSection : "nodes"

  params.delete("section")
  params.set("section", section)

  return { section, normalizedSearch: `?${params.toString()}` }
}

function replaceSettingsSearch(search: string) {
  if (typeof window === "undefined") return
  window.history.replaceState(window.history.state, "", `${window.location.pathname}${search}${window.location.hash}`)
}

function pushSettingsSearch(search: string) {
  if (typeof window === "undefined") return
  window.history.pushState(window.history.state, "", `${window.location.pathname}${search}${window.location.hash}`)
}

export function Settings({ refreshNonce = 0 }: Props) {
  const { t, translateApiError } = useLanguage()
  const [nodes, setNodes] = useState<Node[]>([])
  const [showModal, setShowModal] = useState(false)
  const [loadingNodes, setLoadingNodes] = useState(true)
  const [nodesError, setNodesError] = useState<string | null>(null)
  const [currentPassword, setCurrentPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [changingPassword, setChangingPassword] = useState(false)
  const [loggingOut, setLoggingOut] = useState(false)
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSuccess, setPasswordSuccess] = useState<string | null>(null)
  const [versionMeta, setVersionMeta] = useState<VersionMeta | null>(null)
  const [versionError, setVersionError] = useState<string | null>(null)
  const [nowMs, setNowMs] = useState(() => Date.now())
  const [activeSection, setActiveSection] = useState<SettingsSection>(() => {
    if (typeof window === "undefined") return "nodes"
    return resolveSettingsSection(window.location.search).section
  })

  useEffect(() => {
    if (typeof window === "undefined") return

    const syncSectionFromLocation = () => {
      const { section, normalizedSearch } = resolveSettingsSection(window.location.search)
      if (window.location.search !== normalizedSearch) {
        replaceSettingsSearch(normalizedSearch)
      }
      setActiveSection(section)
    }

    syncSectionFromLocation()
    window.addEventListener("popstate", syncSectionFromLocation)

    return () => {
      window.removeEventListener("popstate", syncSectionFromLocation)
    }
  }, [])

  const fetchNodes = useCallback(async () => {
    setLoadingNodes(true)
    setNodesError(null)

    try {
      const response = await api.nodes()
      setNodes(response.nodes ?? [])
    } catch {
      setNodesError(t("We couldn't load settings data. Please try again."))
    } finally {
      setLoadingNodes(false)
    }
  }, [t])

  useEffect(() => {
    void fetchNodes()
  }, [fetchNodes])

  useEffect(() => {
    if (refreshNonce === 0) return
    void fetchNodes()
  }, [fetchNodes, refreshNonce])

  useEffect(() => {
    const intervalId = window.setInterval(() => {
      setNowMs(Date.now())
    }, 1000)

    return () => {
      window.clearInterval(intervalId)
    }
  }, [])

  const effectiveNodes = useMemo(() => nodes.map((node) => withEffectiveNodeStatus(node, nowMs)), [nodes, nowMs])

  const sectionTabs = useMemo(
    () =>
      [
        { value: "nodes", label: t("settingsPage.sectionNodes") },
        { value: "agent", label: t("settingsPage.sectionAgent") },
        { value: "monitoring", label: t("settingsPage.sectionMonitoring") },
        { value: "alerts", label: t("settingsPage.sectionAlerts") },
        { value: "security", label: t("settingsPage.sectionSecurity") },
      ] satisfies Array<{ value: SettingsSection; label: string }>,
    [t],
  )

  const fetchVersionMeta = useCallback(async () => {
    setVersionError(null)
    try {
      const versionMetaRequest = (api as { versionMeta?: () => Promise<VersionMeta> }).versionMeta
      if (!versionMetaRequest) {
        setVersionError(t("Version metadata is currently unavailable."))
        return
      }
      const response = await versionMetaRequest()
      setVersionMeta(response)
    } catch {
      setVersionError(t("Version metadata is currently unavailable."))
    }
  }, [t])

  useEffect(() => {
    void fetchVersionMeta()
  }, [fetchVersionMeta])

  useEffect(() => {
    if (refreshNonce === 0) return
    void fetchVersionMeta()
  }, [fetchVersionMeta, refreshNonce])

  const handleChangePassword = useCallback(async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setPasswordError(null)
    setPasswordSuccess(null)

    if (!currentPassword || !newPassword || !confirmPassword) {
      setPasswordError(t("All password fields are required."))
      return
    }
    if (newPassword !== confirmPassword) {
      setPasswordError(t("New password and confirmation do not match."))
      return
    }
    if (newPassword === currentPassword) {
      setPasswordError(t("New password must be different from the current password."))
      return
    }

    setChangingPassword(true)
    try {
      await api.changePassword(currentPassword, newPassword)
      setCurrentPassword("")
      setNewPassword("")
      setConfirmPassword("")
      setPasswordSuccess(t("Password updated successfully."))
    } catch (error) {
      const message = error instanceof Error ? translateApiError(error.message) : t("Failed to update password.")
      setPasswordError(message)
    } finally {
      setChangingPassword(false)
    }
  }, [confirmPassword, currentPassword, newPassword, t, translateApiError])

  const handleLogout = useCallback(async () => {
    setPasswordError(null)
    setPasswordSuccess(null)
    setLoggingOut(true)
    try {
      await api.logout()
      window.location.href = "/login"
    } catch (error) {
      const message = error instanceof Error ? translateApiError(error.message) : t("Failed to sign out.")
      setPasswordError(message)
      setLoggingOut(false)
    }
  }, [t, translateApiError])

  const handleSectionChange = useCallback((nextValue: string) => {
    if (!isSettingsSection(nextValue) || nextValue === activeSection || typeof window === "undefined") {
      return
    }

    const { normalizedSearch } = resolveSettingsSection(`?section=${nextValue}`)
    pushSettingsSearch(normalizedSearch)
    setActiveSection(nextValue)
    window.scrollTo({ top: 0, left: 0, behavior: "auto" })
  }, [activeSection])

  return (
    <MotionSection className="mx-auto max-w-[1440px] space-y-6" delay={0.03}>
      <section className="panel-card enterprise-hero rounded-[28px] px-5 py-5 sm:px-6">
        <div className="flex flex-col gap-4 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="enterprise-kicker text-[11px] font-semibold uppercase tracking-[0.24em]">{t("Control Plane")}</p>
            <h2 className="mt-2 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100 md:text-[2rem]">{t("Settings")}</h2>
            <p className="mt-2 max-w-2xl text-sm text-slate-600 dark:text-slate-400">
              {t("Manage node enrollment, registry actions, and administrator credentials from the same engineering-passport shell used across the dashboard.")}
            </p>
          </div>
          <div className="flex flex-wrap gap-2 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
            <span className="enterprise-chip rounded-full px-3 py-1.5">{t("Node registry")}</span>
            <span className="enterprise-chip rounded-full px-3 py-1.5">{t("Provisioning")}</span>
            <span className="enterprise-chip rounded-full px-3 py-1.5">{t("Security")}</span>
          </div>
        </div>
      </section>

      <Tabs value={activeSection} onValueChange={handleSectionChange} className="space-y-6">
        <TabsList
          aria-label={t("settingsPage.sectionTabsAriaLabel")}
          className="flex h-auto w-full justify-start gap-2 overflow-x-auto rounded-[24px] border border-slate-200/80 bg-white/75 p-2 dark:border-white/10 dark:bg-slate-950/70"
        >
          {sectionTabs.map((section) => (
            <TabsTrigger
              key={section.value}
              value={section.value}
              className="enterprise-chip h-10 rounded-full border border-transparent px-4 text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500 data-[state=active]:border-slate-300 data-[state=active]:bg-slate-900 data-[state=active]:text-white dark:text-slate-300 dark:data-[state=active]:border-white/10 dark:data-[state=active]:bg-slate-100 dark:data-[state=active]:text-slate-950"
            >
              {section.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="nodes" forceMount hidden={activeSection !== "nodes"} className="space-y-6">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-200">{t("Node Management")}</h3>
              <Button
                onClick={() => setShowModal(true)}
                className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
              >
                <Plus className="h-3.5 w-3.5" />
                {t("Add Node")}
              </Button>
            </div>

            {loadingNodes ? (
              <div className="panel-card rounded-2xl border border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                {t("Loading node registry...")}
              </div>
            ) : nodesError ? (
              <div
                role="alert"
                className="panel-card rounded-2xl border border-red-200 bg-red-50/80 px-4 py-5 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300"
              >
                <p>{nodesError}</p>
                <button
                  type="button"
                  onClick={() => void fetchNodes()}
                  className="mt-3 rounded-md border border-red-300 bg-white px-3 py-1.5 text-xs font-medium text-red-700 transition-colors hover:bg-red-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500 dark:border-red-800 dark:bg-red-950/50 dark:text-red-200 dark:hover:bg-red-900/40"
                >
                  {t("Retry")}
                </button>
              </div>
            ) : nodes.length === 0 ? (
              <div className="panel-card rounded-2xl border border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
                {t("No nodes registered yet")}
              </div>
            ) : (
              <NodesTable nodes={effectiveNodes} onUpdated={fetchNodes} />
            )}
          </div>
        </TabsContent>

        <TabsContent value="agent" forceMount hidden={activeSection !== "agent"} className="space-y-6">
          <AgentAutoUpdateCard nodes={effectiveNodes} />
          <LatencyMonitorsCard nodes={effectiveNodes} />
        </TabsContent>

        <TabsContent value="monitoring" forceMount hidden={activeSection !== "monitoring"} className="space-y-6">
          <MetricsRetentionCard />
          <DashboardVisibilityCard />
        </TabsContent>

        <TabsContent value="alerts" forceMount hidden={activeSection !== "alerts"} className="space-y-6">
          <NotificationsCard />
        </TabsContent>

        <TabsContent value="security" forceMount hidden={activeSection !== "security"} className="space-y-6">
          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-200">{t("Version")}</h3>
            <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
              {versionError ? (
                <p role="status" className="text-xs text-slate-500 dark:text-slate-400">{versionError}</p>
              ) : !versionMeta ? (
                <p role="status" className="text-xs text-slate-500 dark:text-slate-400">{t("Loading version metadata...")}</p>
              ) : (
                <dl className="grid gap-3 text-xs sm:grid-cols-3">
                  <div>
                    <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("Server version")}</dt>
                    <dd className="mt-1 font-mono text-slate-800 dark:text-slate-100">{versionMeta.version}</dd>
                  </div>
                  <div>
                    <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("Commit")}</dt>
                    <dd className="mt-1 font-mono text-slate-800 dark:text-slate-100">{versionMeta.commit}</dd>
                  </div>
                  <div>
                    <dt className="font-medium uppercase tracking-[0.08em] text-slate-500 dark:text-slate-400">{t("Build time")}</dt>
                    <dd className="mt-1 text-slate-800 dark:text-slate-100">{versionMeta.build_time}</dd>
                  </div>
                </dl>
              )}
            </section>
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-200">{t("Security")}</h3>
            <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
              <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("Change Password")}</h4>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {t("Update the administrator password used on the login page.")}
              </p>

              <div className="mt-4 flex flex-wrap items-center gap-3 border-b border-slate-200 pb-4 dark:border-white/10">
                <Button type="button" onClick={() => void handleLogout()} disabled={loggingOut} className="h-10 rounded-xl px-4 text-sm font-medium">
                  {loggingOut ? t("Signing out...") : t("Sign out")}
                </Button>
                <p className="text-xs text-slate-500 dark:text-slate-400">{t("Use this to proactively end the current admin session on this browser.")}</p>
              </div>

              <form className="mt-4 space-y-3" onSubmit={handleChangePassword}>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("Current password")}
                  <Input
                    type="password"
                    autoComplete="current-password"
                    aria-label={t("Current password")}
                    value={currentPassword}
                    onChange={(event) => setCurrentPassword(event.target.value)}
                    className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
                  />
                </label>

                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("New password")}
                  <Input
                    type="password"
                    autoComplete="new-password"
                    aria-label={t("New password")}
                    value={newPassword}
                    onChange={(event) => setNewPassword(event.target.value)}
                    className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
                  />
                </label>

                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("Confirm new password")}
                  <Input
                    type="password"
                    autoComplete="new-password"
                    aria-label={t("Confirm new password")}
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
                  />
                </label>

                <Button
                  type="submit"
                  disabled={changingPassword}
                  className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
                >
                  {changingPassword ? t("Updating...") : t("Update Password")}
                </Button>

                {passwordError && (
                  <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{passwordError}</p>
                )}
                {passwordSuccess && (
                  <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{passwordSuccess}</p>
                )}
              </form>
            </section>
          </div>
        </TabsContent>
      </Tabs>

      {showModal && (
        <Suspense
          fallback={
            <div className="panel-card rounded-2xl border border-slate-200 px-4 py-6 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
              {t("Loading")}...
            </div>
          }
        >
          <AddNodeModal
            onClose={() => setShowModal(false)}
            onCreated={fetchNodes}
          />
        </Suspense>
      )}
    </MotionSection>
  )
}
