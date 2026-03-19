import { useEffect, useState, type FormEvent } from "react"
import { useLanguage } from "../../i18n/language"
import { api, type NotificationSettings, type TelegramTarget } from "../../lib/api"
import { Button } from "../ui/button"
import { Input } from "../ui/input"

const defaultTarget = (): TelegramTarget => ({ chat_id: "", topic_id: undefined, name: "" })

const defaultState: NotificationSettings = {
  enabled: false,
  channel: "telegram",
  telegram_bot_token_set: false,
  telegram_bot_token: "",
  telegram_targets: [defaultTarget()],
  cpu_warning_percent: 85,
  cpu_critical_percent: 95,
  mem_warning_percent: 85,
  mem_critical_percent: 95,
  disk_warning_percent: 85,
  disk_critical_percent: 95,
  cooldown_minutes: 30,
  notify_node_offline: true,
  notify_node_online: false,
  node_offline_grace_minutes: 2,
}

export function NotificationsCard() {
  const { t } = useLanguage()
  const [settings, setSettings] = useState<NotificationSettings>(defaultState)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await api.notificationSettings()
        if (cancelled) return
        setSettings({
          ...defaultState,
          ...response,
          telegram_bot_token: "",
          telegram_targets: response.telegram_targets?.length ? response.telegram_targets : [defaultTarget()],
        })
      } catch {
        if (!cancelled) {
          setError(t("settingsPage.notificationsLoadFailed"))
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

  const updateTarget = (index: number, patch: Partial<TelegramTarget>) => {
    setSettings((current: NotificationSettings) => ({
      ...current,
      telegram_targets: current.telegram_targets.map((target: TelegramTarget, idx: number) =>
        idx === index ? { ...target, ...patch } : target,
      ),
    }))
  }

  const buildTargetsPayload = (targets: TelegramTarget[]) =>
    targets
      .filter((target: TelegramTarget) => target.chat_id.trim() !== "")
      .map((target: TelegramTarget) => ({
        name: target.name?.trim() ?? "",
        chat_id: target.chat_id.trim(),
        topic_id: typeof target.topic_id === "number" && Number.isFinite(target.topic_id) ? target.topic_id : undefined,
      }))

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setSaving(true)
    setError(null)
    setSuccess(null)
    try {
      const payload = {
        enabled: settings.enabled,
        channel: settings.channel,
        telegram_bot_token: settings.telegram_bot_token,
        telegram_targets: buildTargetsPayload(settings.telegram_targets),
        cpu_warning_percent: Number(settings.cpu_warning_percent),
        cpu_critical_percent: Number(settings.cpu_critical_percent),
        mem_warning_percent: Number(settings.mem_warning_percent),
        mem_critical_percent: Number(settings.mem_critical_percent),
        disk_warning_percent: Number(settings.disk_warning_percent),
        disk_critical_percent: Number(settings.disk_critical_percent),
        cooldown_minutes: Number(settings.cooldown_minutes),
        notify_node_offline: settings.notify_node_offline,
        notify_node_online: settings.notify_node_online,
        node_offline_grace_minutes: Number(settings.node_offline_grace_minutes),
      }
      const response = await api.updateNotificationSettings(payload)
      setSettings((current: NotificationSettings) => ({
        ...current,
        ...response,
        telegram_bot_token: "",
        telegram_targets: response.telegram_targets?.length ? response.telegram_targets : [defaultTarget()],
      }))
      setSuccess(t("settingsPage.notificationsSaved"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settingsPage.notificationsSaveFailed"))
    } finally {
      setSaving(false)
    }
  }

  const handleSendTest = async () => {
    setTesting(true)
    setError(null)
    setSuccess(null)
    try {
      const targets = buildTargetsPayload(settings.telegram_targets)
      await api.sendTestNotification({
        telegram_bot_token: settings.telegram_bot_token,
        target: targets[0],
      })
      setSuccess(t("settingsPage.notificationsTestSent"))
    } catch (err) {
      setError(err instanceof Error ? err.message : t("settingsPage.notificationsTestFailed"))
    } finally {
      setTesting(false)
    }
  }

  return (
    <section className="panel-card enterprise-surface rounded-[28px] px-5 py-5">
      <div className="flex flex-col gap-2 md:flex-row md:items-start md:justify-between">
        <div>
          <h3 className="text-sm font-semibold text-slate-800 dark:text-slate-100">{t("settingsPage.notificationsTitle")}</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("settingsPage.notificationsDescription")}</p>
        </div>
        <span className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.16em] text-slate-600 dark:text-slate-300">
          Telegram
        </span>
      </div>

      {loading ? (
        <p className="mt-4 text-sm text-slate-500 dark:text-slate-400">{t("Loading")}...</p>
      ) : (
        <form className="mt-4 space-y-4" onSubmit={handleSubmit}>
          <label className="flex items-center gap-3 text-sm font-medium text-slate-700 dark:text-slate-200">
            <input type="checkbox" checked={settings.enabled} onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, enabled: event.target.checked }))} />
            {t("settingsPage.notificationsEnabled")}
          </label>

          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
            {t("settingsPage.telegramBotToken")}
            <Input
              type="password"
              aria-label={t("settingsPage.telegramBotToken")}
              value={settings.telegram_bot_token ?? ""}
              onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, telegram_bot_token: event.target.value }))}
              placeholder={settings.telegram_bot_token_set ? t("settingsPage.telegramBotTokenConfigured") : "123456:ABC..."}
              className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
            />
          </label>

          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium text-slate-600 dark:text-slate-300">{t("settingsPage.telegramTargets")}</p>
              <Button type="button" className="h-9 rounded-xl px-3 text-xs" onClick={() => setSettings((current: NotificationSettings) => ({ ...current, telegram_targets: [...current.telegram_targets, defaultTarget()] }))}>
                {t("settingsPage.addTelegramTarget")}
              </Button>
            </div>
            {settings.telegram_targets.map((target, index) => (
              <div key={index} className="enterprise-inner-surface grid gap-3 rounded-2xl border border-slate-200 px-4 py-4 dark:border-white/10 md:grid-cols-3">
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("settingsPage.targetName")}
                  <Input value={target.name ?? ""} onChange={(event) => updateTarget(index, { name: event.target.value })} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100" />
                </label>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("settingsPage.telegramChatId")}
                  <Input value={target.chat_id} onChange={(event) => updateTarget(index, { chat_id: event.target.value })} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100" />
                </label>
                <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                  {t("settingsPage.telegramTopicId")}
                  <Input value={target.topic_id ?? ""} onChange={(event) => updateTarget(index, { topic_id: event.target.value === "" ? undefined : Number(event.target.value) })} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100" />
                </label>
              </div>
            ))}
          </div>

          <div className="grid gap-3 md:grid-cols-3">
            {[
              ["cpu_warning_percent", "settingsPage.cpuWarningThreshold"],
              ["cpu_critical_percent", "settingsPage.cpuCriticalThreshold"],
              ["mem_warning_percent", "settingsPage.memoryWarningThreshold"],
              ["mem_critical_percent", "settingsPage.memoryCriticalThreshold"],
              ["disk_warning_percent", "settingsPage.diskWarningThreshold"],
              ["disk_critical_percent", "settingsPage.diskCriticalThreshold"],
              ["cooldown_minutes", "settingsPage.notificationCooldownMinutes"],
            ].map(([field, label]) => (
              <label key={field} className="block text-xs font-medium text-slate-600 dark:text-slate-300">
                {t(label)}
                <Input
                  type="number"
                  value={String(settings[field as keyof NotificationSettings] ?? "")}
                  onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, [field]: Number(event.target.value) }))}
                  className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
                />
              </label>
            ))}
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <label className="flex items-center gap-3 text-sm font-medium text-slate-700 dark:text-slate-200">
              <input type="checkbox" checked={settings.notify_node_offline} onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, notify_node_offline: event.target.checked }))} />
              {t("settingsPage.notifyNodeOffline")}
            </label>
            <label className="flex items-center gap-3 text-sm font-medium text-slate-700 dark:text-slate-200">
              <input type="checkbox" checked={settings.notify_node_online} onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, notify_node_online: event.target.checked }))} />
              {t("settingsPage.notifyNodeOnline")}
            </label>
            <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
              {t("settingsPage.nodeOfflineGraceMinutes")}
              <Input
                type="number"
                min="0"
                value={String(settings.node_offline_grace_minutes ?? 0)}
                onChange={(event) => setSettings((current: NotificationSettings) => ({ ...current, node_offline_grace_minutes: Number(event.target.value) }))}
                className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90 dark:text-slate-100"
              />
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" disabled={saving || testing} className="enterprise-accent-button h-10 rounded-xl px-4 text-sm font-medium">
              {saving ? t("settingsPage.notificationsSaving") : t("settingsPage.notificationsSave")}
            </Button>
            <Button type="button" disabled={saving || testing} className="h-10 rounded-xl px-4 text-sm font-medium" onClick={() => void handleSendTest()}>
              {testing ? t("settingsPage.notificationsTesting") : t("settingsPage.notificationsTestSend")}
            </Button>
            {success && <p className="text-xs font-medium text-emerald-600 dark:text-emerald-300">{success}</p>}
            {error && <p role="alert" className="text-xs font-medium text-red-600 dark:text-red-300">{error}</p>}
          </div>
        </form>
      )}
    </section>
  )
}
