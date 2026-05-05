import { useEffect, useMemo, useState } from "react"
import { useLanguage } from "../../i18n/language"
import { api } from "../../lib/api"
import type { Node } from "../../lib/api"
import { copyTextToClipboard } from "../../lib/clipboard"
import { countryCodeToFlagEmoji } from "../../lib/flags"
import { Button } from "../ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog"
import { Input } from "../ui/input"

type Props = {
  nodes: Node[]
  onUpdated?: () => Promise<void> | void
}

const MOBILE_BREAKPOINT_QUERY = "(max-width: 767px)"

function isMobileViewport(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false
  }
  return window.matchMedia(MOBILE_BREAKPOINT_QUERY).matches
}

export function NodesTable({ nodes, onUpdated }: Props) {
  const { language, t, translateError } = useLanguage()
  const [statusFilter, setStatusFilter] = useState<"all" | "online" | "offline">("all")
  const statusOptions: Array<{ value: "all" | "online" | "offline"; label: string }> = [
    { value: "all", label: t("All") },
    { value: "online", label: t("Online") },
    { value: "offline", label: t("Offline") },
  ]
  const [nameAsc, setNameAsc] = useState(false)
  const [busyNodeID, setBusyNodeID] = useState<string | null>(null)
  const [actionError, setActionError] = useState<string | null>(null)
  const [renameTarget, setRenameTarget] = useState<Node | null>(null)
  const [renameName, setRenameName] = useState("")
  const [renameError, setRenameError] = useState<string | null>(null)
  const [scriptTarget, setScriptTarget] = useState<Node | null>(null)
  const [scriptCommand, setScriptCommand] = useState("")
  const [scriptLoading, setScriptLoading] = useState(false)
  const [scriptCopied, setScriptCopied] = useState(false)
  const [scriptError, setScriptError] = useState<string | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Node | null>(null)
  const [mobileView, setMobileView] = useState(isMobileViewport)

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      return
    }

    const mediaQuery = window.matchMedia(MOBILE_BREAKPOINT_QUERY)
    const onMediaQueryChange = (event: MediaQueryListEvent) => {
      setMobileView(event.matches)
    }

    setMobileView(mediaQuery.matches)
    mediaQuery.addEventListener("change", onMediaQueryChange)

    return () => {
      mediaQuery.removeEventListener("change", onMediaQueryChange)
    }
  }, [])


  const rows = useMemo(() => {
    const filtered = nodes.filter((node) => {
      if (statusFilter === "online") return node.online
      if (statusFilter === "offline") return !node.online
      return true
    })
    return filtered.sort((left, right) => {
      const value = left.name.localeCompare(right.name)
      return nameAsc ? value : -value
    })
  }, [nameAsc, nodes, statusFilter])



  const refresh = async () => {
    await onUpdated?.()
  }



  const openRenameDialog = (node: Node) => {
    setRenameTarget(node)
    setRenameName(node.name)
    setRenameError(null)
  }

  const handleRenameConfirm = async () => {
    if (!renameTarget) return

    const trimmed = renameName.trim()
    if (!trimmed) {
      setRenameError(t("Node name is required"))
      return
    }
    if (trimmed === renameTarget.name) {
      setRenameTarget(null)
      setRenameError(null)
      return
    }

    setActionError(null)
    setBusyNodeID(renameTarget.id)
    try {
      await api.renameNode(renameTarget.id, trimmed)
      await refresh()
      setRenameTarget(null)
      setRenameError(null)
    } catch (error) {
      setRenameError(error instanceof Error ? translateError(error.message) : t("Action failed. Please try again."))
    } finally {
      setBusyNodeID(null)
    }
  }

  const handleGetScript = async (node: Node) => {
    setActionError(null)
    setScriptError(null)
    setScriptCopied(false)
    setScriptTarget(node)
    setScriptLoading(true)
    setScriptCommand("")
    setBusyNodeID(node.id)

    try {
      const response = await api.installCommand(node.id)
      setScriptCommand(response.command)
    } catch (error) {
      const message = error instanceof Error ? translateError(error.message) : t("Action failed. Please try again.")
      setScriptError(message)
      setActionError(message)
    } finally {
      setScriptLoading(false)
      setBusyNodeID(null)
    }
  }

  const handleCopyScript = async () => {
    if (!scriptCommand) return

    const result = await copyTextToClipboard(scriptCommand)
    if (result.ok) {
      setScriptCopied(true)
      return
    }

    const message = translateError(result.message)
    setScriptCopied(false)
    setScriptError(message)
  }

  const handleDeleteConfirm = async () => {
    if (!deleteTarget) return

    setActionError(null)
    setBusyNodeID(deleteTarget.id)
    try {
      await api.deleteNode(deleteTarget.id)
      await refresh()
      setDeleteTarget(null)
    } catch (error) {
      setActionError(error instanceof Error ? translateError(error.message) : t("Action failed. Please try again."))
    } finally {
      setBusyNodeID(null)
    }
  }

  const statusLabel = (online: boolean) => online ? t("Online") : t("Offline")
  const formatCreatedDate = (createdAt: number) => createdAt ? new Date(createdAt * 1000).toLocaleDateString(language) : "—"
  const renderNodeName = (node: Node) => {
    const flagEmoji = countryCodeToFlagEmoji(node.country_code)
    return <>
      {flagEmoji ? <span className="mr-1" aria-hidden="true">{flagEmoji}</span> : null}
      <span>{node.name}</span>
    </>
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] p-5 space-y-4">
      <div>
        <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("Node Registry")}</h4>
        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("Manage node lifecycle and access commands.")}</p>
      </div>

      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
          <span>{t("Settings status filter")}</span>
          <div
            role="group"
            aria-label={t("Settings status filter")}
            className="enterprise-inner-surface inline-flex w-full gap-1 rounded-2xl p-1.5 shadow-none md:w-auto md:p-1"
          >
            {statusOptions.map((option) => {
              const active = statusFilter === option.value
              return (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={active}
                  onClick={() => setStatusFilter(option.value)}
                  className={`h-10 flex-1 cursor-pointer rounded-xl border border-transparent px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] transition-all duration-200 focus:outline-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-inset active:translate-y-px active:scale-[0.99] md:flex-initial ${
                    active
                      ? "border border-slate-200/80 bg-slate-50/90 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50 dark:ring-1 dark:ring-inset dark:ring-white/10 dark:shadow-none"
                      : "text-slate-600 hover:bg-white/85 dark:text-slate-200 dark:hover:bg-slate-900"
                  }`}
                >
                  {option.label}
                </button>
              )
            })}
          </div>
        </div>
      </div>

      {actionError && (
        <p role="alert" className="text-xs text-red-600 dark:text-red-300">
          {actionError}
        </p>
      )}

      {rows.length === 0 ? (
        <div className="rounded-xl border border-dashed border-slate-300 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
          {t("No nodes match the active status filter.")}
        </div>
      ) : mobileView ? (
        <div className="space-y-3">
          {rows.map((node) => (
            <article key={node.id} className="enterprise-inner-surface rounded-2xl p-3">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{renderNodeName(node)}</p>
                  <p className="truncate text-xs text-slate-500 dark:text-slate-400">{node.ip || "—"}</p>
                </div>
                <span
                  className={`rounded-md border px-2 py-0.5 text-[11px] font-medium ${
                    node.online
                      ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-300"
                      : "border-slate-300 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300"
                  }`}
                >
                  {statusLabel(node.online)}
                </span>
              </div>

              <dl className="mt-3 grid grid-cols-2 gap-x-3 gap-y-2 text-xs">
                <div>
                  <dt className="text-slate-500 dark:text-slate-400">{t("OS / Arch")}</dt>
                  <dd className="mt-0.5 text-slate-700 dark:text-slate-200">{node.os && node.arch ? `${node.os}/${node.arch}` : "—"}</dd>
                </div>
                <div>
                  <dt className="text-slate-500 dark:text-slate-400">{t("dashboard.table.agent")}</dt>
                  <dd className="mt-0.5 font-mono text-slate-700 dark:text-slate-200">{node.agent_version || "—"}</dd>
                </div>
                <div>
                  <dt className="text-slate-500 dark:text-slate-400">{t("Created")}</dt>
                  <dd className="mt-0.5 text-slate-700 dark:text-slate-200">{formatCreatedDate(node.created_at)}</dd>
                </div>
              </dl>

              <div className="mt-3 grid grid-cols-3 gap-2">
                <button type="button" onClick={() => openRenameDialog(node)} disabled={busyNodeID === node.id} className="enterprise-outline-control h-10 rounded-xl border px-2 text-xs font-medium text-slate-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:bg-slate-950 dark:text-slate-200">
                  {t("Rename")}
                </button>
                <button type="button" onClick={() => void handleGetScript(node)} disabled={busyNodeID === node.id} className="enterprise-outline-control h-10 rounded-xl border px-2 text-xs font-medium text-slate-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:bg-slate-950 dark:text-slate-200">
                  {t("Get Script")}
                </button>
                <button type="button" onClick={() => setDeleteTarget(node)} disabled={busyNodeID === node.id} className="h-10 rounded-xl border border-red-200 bg-red-50 px-2 text-xs font-medium text-red-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300">
                  {t("Delete")}
                </button>
              </div>
            </article>
          ))}
        </div>
      ) : (
        <div className="enterprise-inner-surface overflow-x-auto rounded-2xl px-2 py-2">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-slate-200 text-left text-slate-500 dark:border-slate-700 dark:text-slate-400">
                <th className="py-2 pr-3 font-medium">
                  <button type="button" onClick={() => setNameAsc((current) => !current)} className="hover:text-slate-900 dark:hover:text-slate-200">
                    {t("Name")}
                  </button>
                </th>
                <th className="py-2 pr-3 font-medium">{t("dashboard.table.agent")}</th>
                <th className="py-2 pr-3 font-medium">{t("IP")}</th>
                <th className="py-2 pr-3 font-medium">{t("OS / Arch")}</th>
                <th className="py-2 pr-3 font-medium">{t("Status")}</th>
                <th className="py-2 pr-3 font-medium">{t("Created")}</th>
                <th className="py-2 pr-3 font-medium">{t("Actions")}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((node) => (
                <tr key={node.id} className="border-b border-slate-100 hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-900/70">
                  <td className="py-2 pr-3 text-slate-900 dark:text-slate-100">{renderNodeName(node)}</td>
                  <td className="py-2 pr-3 font-mono text-xs text-slate-600 dark:text-slate-300">{node.agent_version || "—"}</td>
                  <td className="py-2 pr-3 text-slate-600 dark:text-slate-300">{node.ip || "—"}</td>
                  <td className="py-2 pr-3 text-slate-600 dark:text-slate-300">{node.os && node.arch ? `${node.os}/${node.arch}` : "—"}</td>
                  <td className="py-2 pr-3 text-slate-600 dark:text-slate-300">{statusLabel(node.online)}</td>
                  <td className="py-2 pr-3 text-slate-600 dark:text-slate-300">{formatCreatedDate(node.created_at)}</td>
                  <td className="py-2 pr-3">
                    <div className="flex gap-1.5">
                      <button type="button" onClick={() => openRenameDialog(node)} disabled={busyNodeID === node.id} className="enterprise-outline-control rounded-xl border px-2.5 py-1.5 text-xs text-slate-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:bg-slate-950 dark:text-slate-200">
                        {t("Rename")}
                      </button>
                      <button type="button" onClick={() => void handleGetScript(node)} disabled={busyNodeID === node.id} className="enterprise-outline-control rounded-xl border px-2.5 py-1.5 text-xs text-slate-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:bg-slate-950 dark:text-slate-200">
                        {t("Get Script")}
                      </button>
                      <button type="button" onClick={() => setDeleteTarget(node)} disabled={busyNodeID === node.id} className="rounded-xl border border-red-200 bg-red-50 px-2.5 py-1.5 text-xs text-red-700 shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300">
                        {t("Delete")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Dialog
        open={renameTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setRenameTarget(null)
            setRenameError(null)
          }
        }}
      >
        <DialogContent className="enterprise-hero max-w-md rounded-[28px] border p-6">
          <DialogHeader>
            <DialogTitle>{t("Rename Node")}</DialogTitle>
            <DialogDescription>{t("Update the display name for this node.")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <label htmlFor="rename-node-name" className="block text-xs font-medium text-slate-600 dark:text-slate-300">
              {t("Node Name")}
            </label>
            <Input id="rename-node-name" aria-label={t("Node Name")} value={renameName} onChange={(event) => setRenameName(event.target.value)} placeholder={t("Enter a node name")} className="h-10 rounded-xl" />
            {renameError && <p className="text-xs text-red-600 dark:text-red-300">{renameError}</p>}
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setRenameTarget(null)} className="rounded-xl">{t("Cancel")}</Button>
            <Button type="button" onClick={() => void handleRenameConfirm()} disabled={!renameTarget || busyNodeID === renameTarget.id} className="rounded-xl">{t("Save")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={scriptTarget !== null}
        onOpenChange={(open) => {
          if (!open) {
            setScriptTarget(null)
            setScriptCommand("")
            setScriptError(null)
            setScriptCopied(false)
            setScriptLoading(false)
          }
        }}
      >
        <DialogContent className="enterprise-hero max-w-xl rounded-[28px] border p-6">
          <DialogHeader>
            <DialogTitle>{t("Install Command")}</DialogTitle>
            <DialogDescription>{t("Copy this command and run it on the target machine as root.")}</DialogDescription>
          </DialogHeader>
          {scriptLoading ? (
            <p className="text-sm text-slate-500 dark:text-slate-400">{t("Generating command...")}</p>
          ) : scriptError && !scriptCommand ? (
            <p className="text-sm text-red-600 dark:text-red-300">{scriptError}</p>
          ) : (
            <div className="space-y-3">
              <code className="block break-all rounded-2xl border border-slate-200/80 bg-slate-50/90 px-3 py-2 font-mono text-xs text-slate-700 dark:border-white/8 dark:bg-slate-950 dark:text-slate-200">{scriptCommand}</code>
              {scriptError && <p className="text-sm text-red-600 dark:text-red-300">{scriptError}</p>}
            </div>
          )}
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setScriptTarget(null)} className="rounded-xl">{t("Done")}</Button>
            <Button type="button" aria-label={t("Copy command")} onClick={() => void handleCopyScript()} disabled={!scriptCommand || scriptLoading} className="rounded-xl">{scriptCopied ? t("Copied") : t("Copy")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <DialogContent aria-label={t("Delete Node")} className="enterprise-hero max-w-md rounded-[28px] border p-6">
          <DialogHeader>
            <DialogTitle>{t("Delete Node")}</DialogTitle>
            <DialogDescription>{t("settingsTable.deleteNodeDescription", { name: deleteTarget?.name ?? "" })}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setDeleteTarget(null)} className="rounded-xl">{t("Cancel")}</Button>
            <Button type="button" variant="destructive" aria-label={t("Delete Node")} onClick={() => void handleDeleteConfirm()} disabled={!deleteTarget || busyNodeID === deleteTarget.id} className="rounded-xl">{t("Delete Node")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </section>
  )
}
