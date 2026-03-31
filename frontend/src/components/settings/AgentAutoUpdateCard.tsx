import { useEffect, useMemo, useState } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type AgentReleaseManifest, type Node } from "../../lib/api"
import { Button } from "../ui/button"

type ReleaseState = {
  amd64: AgentReleaseManifest | null
  arm64: AgentReleaseManifest | null
}

type SupportedArch = keyof ReleaseState

type Props = {
  nodes: Node[]
}

const supportedArchOrder: SupportedArch[] = ["amd64", "arm64"]

function formatInterval(t: (key: string, params?: Record<string, string | number | undefined>) => string, seconds: number) {
  const minutes = Math.max(1, Math.floor(seconds / 60))
  return t("settingsPage.autoUpdateChecksEveryMinutes", { count: minutes })
}

function normalizeArch(arch?: string): SupportedArch | null {
  const normalized = arch?.trim().toLowerCase()
  if (normalized === "amd64" || normalized === "arm64") {
    return normalized
  }
  return null
}

function needsUpdate(node: Node, manifest: AgentReleaseManifest): boolean {
  const targetVersion = manifest.target_version.trim()
  const reportedVersion = node.agent_version?.trim() ?? ""

  if (targetVersion === "") {
    return true
  }
  if (reportedVersion === "") {
    return true
  }

  return reportedVersion !== targetVersion
}

export function AgentAutoUpdateCard({ nodes }: Props) {
  const { t, translateApiError } = useLanguage()
  const [releaseState, setReleaseState] = useState<ReleaseState>({ amd64: null, arm64: null })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)
  const [actionMessage, setActionMessage] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const [amd64, arm64] = await Promise.all([
          api.agentRelease("linux", "amd64"),
          api.agentRelease("linux", "arm64"),
        ])
        if (!cancelled) {
          setReleaseState({ amd64, arm64 })
        }
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.autoUpdateUnavailable"))
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [t])

  const intervalLabel = useMemo(() => {
    const seconds = releaseState.amd64?.check_interval_seconds ?? releaseState.arm64?.check_interval_seconds
    return seconds ? formatInterval(t, seconds) : null
  }, [releaseState, t])

  const eligibleGroups = useMemo(() => supportedArchOrder.flatMap((arch) => {
    const manifest = releaseState[arch]
    if (!manifest) {
      return []
    }

    const nodeIDs = nodes
      .filter((node) => node.online && node.os === "linux" && normalizeArch(node.arch) === arch && needsUpdate(node, manifest))
      .map((node) => node.id)

    if (nodeIDs.length === 0) {
      return []
    }

    return [{ arch, manifest, nodeIDs }]
  }), [nodes, releaseState])

  const eligibleNodeCount = useMemo(
    () => eligibleGroups.reduce((sum, group) => sum + group.nodeIDs.length, 0),
    [eligibleGroups],
  )

  const handleUpdateNow = async () => {
    setActionError(null)
    setActionMessage(null)

    if (eligibleGroups.length === 0) {
      setActionError(t("settingsPage.autoUpdateNoEligibleNodes"))
      return
    }

    setSubmitting(true)
    try {
      const results = await Promise.allSettled(
        eligibleGroups.map(({ nodeIDs, manifest }) =>
          api.createAgentUpdateJob(nodeIDs, manifest.target_version, manifest.download_url, manifest.sha256),
        ),
      )

      let succeededNodes = 0
      let failedGroups = 0
      let firstFailureMessage: string | null = null

      results.forEach((result, index) => {
        if (result.status === "fulfilled") {
          succeededNodes += eligibleGroups[index].nodeIDs.length
          return
        }

        failedGroups += 1
        if (!firstFailureMessage) {
          const reason = result.reason
          firstFailureMessage = reason instanceof Error ? translateApiError(reason.message) : t("settingsPage.autoUpdateDispatchFailed")
        }
      })

      if (succeededNodes === 0) {
        setActionError(firstFailureMessage ?? t("settingsPage.autoUpdateDispatchFailed"))
        return
      }

      if (failedGroups === 0) {
        setActionMessage(t("settingsPage.autoUpdateDispatchSuccess", { count: succeededNodes }))
        return
      }

      setActionMessage(t("settingsPage.autoUpdateDispatchPartial", { count: succeededNodes, failed: failedGroups }))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.autoUpdateTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.autoUpdateDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-emerald-700 dark:text-emerald-300">
          {t("settingsPage.autoUpdateEnabled")}
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-300">{error}</p>
      ) : (
        <div className="mt-4 space-y-3">
          {intervalLabel && (
            <p className="text-xs font-medium text-slate-600 dark:text-slate-300">{intervalLabel}</p>
          )}
          <div className="enterprise-inner-surface flex flex-col gap-3 rounded-2xl px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <p className="text-[11px] font-medium uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
                {t("settingsPage.autoUpdateReadyLabel")}
              </p>
              <p className="text-sm font-medium text-slate-800 dark:text-slate-100">
                {t("settingsPage.autoUpdateReadyCount", { count: eligibleNodeCount })}
              </p>
            </div>
            <Button
              type="button"
              onClick={() => void handleUpdateNow()}
              disabled={submitting || eligibleNodeCount === 0}
              className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium"
            >
              {submitting ? t("settingsPage.autoUpdateUpdatingNow") : t("settingsPage.autoUpdateUpdateNow")}
            </Button>
          </div>
          <div className="grid gap-3 md:grid-cols-2">
            {releaseState.amd64 && (
              <div className="enterprise-inner-surface rounded-2xl px-4 py-3 text-sm">
                <p className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">linux/amd64</p>
                <p className="mt-1 font-medium text-slate-800 dark:text-slate-100">linux/amd64 · {releaseState.amd64.target_version}</p>
              </div>
            )}
            {releaseState.arm64 && (
              <div className="enterprise-inner-surface rounded-2xl px-4 py-3 text-sm">
                <p className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">linux/arm64</p>
                <p className="mt-1 font-medium text-slate-800 dark:text-slate-100">linux/arm64 · {releaseState.arm64.target_version}</p>
              </div>
            )}
          </div>
          {actionError && (
            <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{actionError}</p>
          )}
          {actionMessage && (
            <p role="status" className="text-xs font-medium text-emerald-700 dark:text-emerald-300">{actionMessage}</p>
          )}
        </div>
      )}
    </section>
  )
}
