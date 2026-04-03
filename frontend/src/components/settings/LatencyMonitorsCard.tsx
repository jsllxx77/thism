import { useEffect, useState } from "react"
import { Pencil, Plus, Trash2 } from "lucide-react"
import { api, type LatencyMonitor, type LatencyMonitorType, type Node } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { Button } from "../ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "../ui/dialog"
import { Input } from "../ui/input"

type Props = {
  nodes: Node[]
}

type FormState = {
  id: string | null
  name: string
  type: LatencyMonitorType
  target: string
  intervalSeconds: string
  nodeIDs: string[]
}

function typeLabel(type: LatencyMonitorType) {
  return type.toUpperCase()
}

function defaultForm(nodes: Node[]): FormState {
  return {
    id: null,
    name: "",
    type: "tcp",
    target: "",
    intervalSeconds: "60",
    nodeIDs: nodes.map((node) => node.id),
  }
}

export function LatencyMonitorsCard({ nodes }: Props) {
  const { t } = useLanguage()
  const [monitors, setMonitors] = useState<LatencyMonitor[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [open, setOpen] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [form, setForm] = useState<FormState>(() => defaultForm(nodes))

  const loadMonitors = async () => {
    setLoading(true)
    setError(null)
    try {
      const request = (api as { latencyMonitors?: typeof api.latencyMonitors }).latencyMonitors
      if (!request) {
        setMonitors([])
        return
      }
      const response = await request()
      setMonitors(response.monitors ?? [])
    } catch {
      setError(t("settingsPage.latencyMonitorsLoadFailed"))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadMonitors()
  }, [])

  const resetForm = (nextForm?: FormState) => {
    setForm(nextForm ?? defaultForm(nodes))
  }

  const openCreate = () => {
    setError(null)
    setSuccess(null)
    resetForm(defaultForm(nodes))
    setOpen(true)
  }

  const openEdit = (monitor: LatencyMonitor) => {
    setError(null)
    setSuccess(null)
    resetForm({
      id: monitor.id,
      name: monitor.name,
      type: monitor.type,
      target: monitor.target,
      intervalSeconds: String(monitor.interval_seconds),
      nodeIDs: monitor.assigned_node_ids ?? [],
    })
    setOpen(true)
  }

  const toggleNode = (nodeID: string) => {
    setForm((current) => ({
      ...current,
      nodeIDs: current.nodeIDs.includes(nodeID)
        ? current.nodeIDs.filter((candidate) => candidate !== nodeID)
        : [...current.nodeIDs, nodeID],
    }))
  }

  const handleSubmit = async () => {
    setError(null)
    setSuccess(null)

    const intervalSeconds = Number(form.intervalSeconds)
    if (!form.name.trim() || !form.target.trim() || !Number.isFinite(intervalSeconds) || intervalSeconds <= 0) {
      setError(t("settingsPage.latencyMonitorsSaveFailed"))
      return
    }

    setSaving(true)
    try {
      const payload = {
        name: form.name.trim(),
        type: form.type,
        target: form.target.trim(),
        interval_seconds: intervalSeconds,
        auto_assign_new_nodes: true,
        node_ids: form.nodeIDs,
      }
      if (form.id) {
        const request = (api as { updateLatencyMonitor?: typeof api.updateLatencyMonitor }).updateLatencyMonitor
        if (!request) {
          throw new Error("latency monitor update unavailable")
        }
        await request(form.id, payload)
      } else {
        const request = (api as { createLatencyMonitor?: typeof api.createLatencyMonitor }).createLatencyMonitor
        if (!request) {
          throw new Error("latency monitor create unavailable")
        }
        await request(payload)
      }
      await loadMonitors()
      setOpen(false)
      resetForm()
      setSuccess(t("settingsPage.latencyMonitorsSaved"))
    } catch {
      setError(t("settingsPage.latencyMonitorsSaveFailed"))
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (monitorID: string) => {
    setError(null)
    setSuccess(null)
    try {
      const request = (api as { deleteLatencyMonitor?: typeof api.deleteLatencyMonitor }).deleteLatencyMonitor
      if (!request) {
        throw new Error("latency monitor delete unavailable")
      }
      await request(monitorID)
      await loadMonitors()
      setSuccess(t("settingsPage.latencyMonitorsDeleted"))
    } catch {
      setError(t("settingsPage.latencyMonitorsSaveFailed"))
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.latencyMonitorsTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.latencyMonitorsDescription")}</p>
        </div>
        <Button type="button" onClick={openCreate} className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium">
          <Plus className="h-3.5 w-3.5" />
          {t("settingsPage.latencyMonitorsNew")}
        </Button>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("settingsPage.latencyMonitorsLoading")}</p>
      ) : monitors.length === 0 ? (
        <div className="mt-4 rounded-2xl border border-dashed border-slate-300/80 px-4 py-6 text-sm text-slate-500 dark:border-white/10 dark:text-slate-400">
          {t("settingsPage.latencyMonitorsEmpty")}
        </div>
      ) : (
        <div className="mt-4 space-y-3">
          {monitors.map((monitor) => (
            <div key={monitor.id} className="enterprise-inner-surface rounded-2xl border px-4 py-4">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-2">
                  <div className="flex flex-wrap items-center gap-2">
                    <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{monitor.name}</h4>
                    <span className="enterprise-chip rounded-full px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
                      {typeLabel(monitor.type)}
                    </span>
                  </div>
                  <dl className="grid gap-2 text-xs text-slate-500 dark:text-slate-400 md:grid-cols-3">
                    <div>
                      <dt className="font-medium uppercase tracking-[0.14em]">{t("settingsPage.latencyMonitorsTarget")}</dt>
                      <dd className="mt-1 break-all text-slate-700 dark:text-slate-200">{monitor.target}</dd>
                    </div>
                    <div>
                      <dt className="font-medium uppercase tracking-[0.14em]">{t("settingsPage.latencyMonitorsInterval")}</dt>
                      <dd className="mt-1 text-slate-700 dark:text-slate-200">{t("settingsPage.latencyMonitorsIntervalValue", { count: monitor.interval_seconds })}</dd>
                    </div>
                    <div>
                      <dt className="font-medium uppercase tracking-[0.14em]">{t("settingsPage.latencyMonitorsAssignedNodes")}</dt>
                      <dd className="mt-1 text-slate-700 dark:text-slate-200">{monitor.assigned_node_count}</dd>
                    </div>
                  </dl>
                </div>

                <div className="flex items-center gap-2">
                  <Button type="button" variant="outline" onClick={() => openEdit(monitor)} className="h-9 rounded-xl px-3 text-xs font-medium">
                    <Pencil className="h-3.5 w-3.5" />
                    {t("settingsPage.latencyMonitorsEdit")}
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void handleDelete(monitor.id)} className="h-9 rounded-xl px-3 text-xs font-medium text-red-600 hover:text-red-700 dark:text-red-300 dark:hover:text-red-200">
                    <Trash2 className="h-3.5 w-3.5" />
                    {t("settingsPage.latencyMonitorsDelete")}
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {success && <p className="mt-4 text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
      {error && <p role="alert" className="mt-4 text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}

      <Dialog open={open} onOpenChange={(nextOpen) => !saving && setOpen(nextOpen)}>
        <DialogContent className="enterprise-hero max-w-2xl rounded-[28px] border p-6">
          <DialogHeader>
            <DialogTitle>{form.id ? t("settingsPage.latencyMonitorsEdit") : t("settingsPage.latencyMonitorsNew")}</DialogTitle>
            <DialogDescription>{t("settingsPage.latencyMonitorsDialogDescription")}</DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 md:grid-cols-2">
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
              {t("settingsPage.latencyMonitorsName")}
              <Input
                aria-label={t("settingsPage.latencyMonitorsName")}
                value={form.name}
                onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                className="enterprise-outline-control mt-2 rounded-xl border"
              />
            </label>

            <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
              {t("settingsPage.latencyMonitorsType")}
              <select
                aria-label={t("settingsPage.latencyMonitorsType")}
                value={form.type}
                onChange={(event) => setForm((current) => ({ ...current, type: event.target.value as LatencyMonitorType }))}
                className="enterprise-outline-control mt-2 h-10 w-full rounded-xl border bg-white px-3 text-sm text-slate-800 dark:bg-slate-950/90 dark:text-slate-100"
              >
                <option value="icmp">ICMP</option>
                <option value="tcp">TCP</option>
                <option value="http">HTTP</option>
              </select>
            </label>

            <label className="block text-xs font-medium text-slate-600 dark:text-slate-300 md:col-span-2">
              {t("settingsPage.latencyMonitorsTarget")}
              <Input
                aria-label={t("settingsPage.latencyMonitorsTarget")}
                value={form.target}
                onChange={(event) => setForm((current) => ({ ...current, target: event.target.value }))}
                className="enterprise-outline-control mt-2 rounded-xl border"
              />
            </label>

            <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
              {t("settingsPage.latencyMonitorsInterval")}
              <Input
                aria-label={t("settingsPage.latencyMonitorsInterval")}
                type="number"
                min={1}
                value={form.intervalSeconds}
                onChange={(event) => setForm((current) => ({ ...current, intervalSeconds: event.target.value }))}
                className="enterprise-outline-control mt-2 rounded-xl border"
              />
            </label>
          </div>

          <div className="mt-4">
            <p className="text-xs font-medium text-slate-600 dark:text-slate-300">{t("settingsPage.latencyMonitorsNodeScope")}</p>
            <div className="mt-3 grid gap-2 md:grid-cols-2">
              {nodes.map((node) => {
                const checked = form.nodeIDs.includes(node.id)
                return (
                  <label
                    key={node.id}
                    className={`enterprise-inner-surface flex cursor-pointer items-center justify-between rounded-2xl border px-4 py-3 text-sm ${
                      checked
                        ? "border-slate-300 bg-slate-50 text-slate-900 dark:border-white/10 dark:bg-slate-900 dark:text-slate-50"
                        : "border-slate-200 bg-white/80 text-slate-600 dark:border-white/8 dark:bg-slate-950/80 dark:text-slate-200"
                    }`}
                  >
                    <span>{node.name}</span>
                    <input
                      type="checkbox"
                      aria-label={t("settingsPage.latencyMonitorsAssignNode", { name: node.name })}
                      checked={checked}
                      onChange={() => toggleNode(node.id)}
                    />
                  </label>
                )
              })}
            </div>
          </div>

          <div className="mt-6 flex flex-wrap items-center gap-3">
            <Button type="button" onClick={() => void handleSubmit()} disabled={saving} className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium">
              {saving ? t("settingsPage.latencyMonitorsSaving") : t("settingsPage.latencyMonitorsSave")}
            </Button>
            <Button type="button" variant="outline" onClick={() => setOpen(false)} disabled={saving} className="h-10 rounded-xl px-4 text-sm font-medium">
              {t("common.cancel")}
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </section>
  )
}
