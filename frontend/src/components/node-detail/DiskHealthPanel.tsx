import { useId, useState } from "react"
import { AlertTriangle, CheckCircle2, ChevronDown, HardDrive, HelpCircle, ShieldAlert, ShieldCheck, XCircle } from "lucide-react"
import type { DiskHealthStats, DiskHealthStatus, DiskHealthSupportStatus } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { formatBytes } from "../../lib/units"
import { CollapsibleContent } from "./CollapsibleContent"

type Props = {
  disks?: DiskHealthStats[]
}

function formatOptionalPercent(value?: number): string {
  return typeof value === "number" && Number.isFinite(value) ? `${value.toFixed(1)}%` : "—"
}

function formatOptionalTemperature(value?: number): string {
  return typeof value === "number" && Number.isFinite(value) ? `${value.toFixed(1)}°C` : "—"
}

function formatOptionalCount(value?: number): string {
  return typeof value === "number" && Number.isFinite(value) ? String(Math.round(value)) : "—"
}

function formatOptionalBytes(value?: number): string {
  return typeof value === "number" && Number.isFinite(value) && value > 0 ? formatBytes(value) : "—"
}

function formatMounts(disk: DiskHealthStats): string {
  return disk.mounts?.map((mount) => mount.mount).filter(Boolean).join(", ") ?? ""
}

function effectiveSupportStatus(disk: DiskHealthStats): DiskHealthSupportStatus {
  return disk.support_status ?? (disk.status === "unsupported" ? "unsupported" : "unknown")
}

function statusTone(status: DiskHealthStatus) {
  switch (status) {
    case "ok":
      return {
        icon: CheckCircle2,
        labelKey: "nodeDetail.diskHealthStatusOk",
        className: "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300",
      }
    case "warning":
      return {
        icon: AlertTriangle,
        labelKey: "nodeDetail.diskHealthStatusWarning",
        className: "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300",
      }
    case "critical":
      return {
        icon: XCircle,
        labelKey: "nodeDetail.diskHealthStatusCritical",
        className: "border-red-200 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300",
      }
    case "unsupported":
      return {
        icon: HelpCircle,
        labelKey: "nodeDetail.diskHealthStatusUnsupported",
        className: "border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-300",
      }
    default:
      return {
        icon: HelpCircle,
        labelKey: "nodeDetail.diskHealthStatusUnknown",
        className: "border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-300",
      }
  }
}

function supportTone(status?: DiskHealthSupportStatus) {
  switch (status) {
    case "supported":
      return {
        icon: ShieldCheck,
        labelKey: "nodeDetail.diskHealthSupportSupported",
        className: "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-500/20 dark:bg-sky-500/10 dark:text-sky-300",
      }
    case "unsupported":
      return {
        icon: HelpCircle,
        labelKey: "nodeDetail.diskHealthSupportUnsupported",
        className: "border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-300",
      }
    case "degraded":
      return {
        icon: ShieldAlert,
        labelKey: "nodeDetail.diskHealthSupportDegraded",
        className: "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300",
      }
    default:
      return {
        icon: HelpCircle,
        labelKey: "nodeDetail.diskHealthSupportUnknown",
        className: "border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-300",
      }
  }
}

export function DiskHealthPanel({ disks = [] }: Props) {
  const { t } = useLanguage()
  const contentId = useId()
  const [open, setOpen] = useState(false)

  return (
    <section className="panel-card enterprise-surface rounded-[24px] p-4" aria-labelledby="disk-health-heading">
      <button
        type="button"
        className="mb-4 flex w-full cursor-pointer flex-col gap-2 rounded-2xl text-left transition-colors duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500/70 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-sky-300/70 dark:focus-visible:ring-offset-slate-950 md:flex-row md:items-end md:justify-between"
        aria-expanded={open}
        aria-controls={contentId}
        aria-labelledby="disk-health-heading"
        onClick={() => setOpen((value) => !value)}
      >
        <span>
          <span className="enterprise-kicker block text-[11px] font-semibold uppercase tracking-[0.24em]">{t("nodeDetail.diskHealthEyebrow")}</span>
          <span id="disk-health-heading" role="heading" aria-level={3} className="mt-2 block text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t("nodeDetail.diskHealthTitle")}
          </span>
        </span>
        <span className="flex items-center gap-2">
          <span className="enterprise-chip inline-flex w-fit items-center gap-2 rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em]">
            <HardDrive className="h-3.5 w-3.5" />
            {t("nodeDetail.diskHealthDeviceCount", { count: disks.length })}
          </span>
          <ChevronDown className={`h-4 w-4 text-slate-500 transition-transform duration-200 ease-out dark:text-slate-400 ${open ? "rotate-180" : ""}`} />
        </span>
      </button>

      <CollapsibleContent open={open}>
        <div id={contentId}>
          {disks.length === 0 ? (
            <p className="rounded-2xl border border-slate-200 bg-white/70 p-4 text-sm text-slate-600 dark:border-white/10 dark:bg-white/5 dark:text-slate-300">
              {t("nodeDetail.diskHealthEmpty")}
            </p>
          ) : (
            <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white/70 dark:border-white/10 dark:bg-white/5">
              <div className="grid grid-cols-[minmax(9rem,1.4fr)_minmax(7rem,0.9fr)_repeat(5,minmax(6rem,0.7fr))] gap-0 overflow-x-auto">
                <div className="contents text-[11px] font-semibold uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthDisk")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("common.status")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthTemp")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthLifeUsed")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthSpare")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthMediaErrors")}</div>
                  <div className="border-b border-slate-200 px-3 py-2 dark:border-white/10">{t("nodeDetail.diskHealthUnsafeShutdowns")}</div>
                </div>
                {disks.map((disk) => {
                  const tone = statusTone(disk.status)
                  const Icon = tone.icon
                  const supportStatus = effectiveSupportStatus(disk)
                  const support = supportTone(supportStatus)
                  const SupportIcon = support.icon
                  const mounts = formatMounts(disk)
                  const message = supportStatus === "unsupported" && disk.type === "virtual"
                    ? t("nodeDetail.diskHealthVirtualNotice")
                    : disk.message
                  const mediaErrorDetails = [
                    { label: t("nodeDetail.diskHealthReallocated"), value: disk.reallocated_sectors },
                    { label: t("nodeDetail.diskHealthPending"), value: disk.pending_sectors },
                    { label: t("nodeDetail.diskHealthUncorrectable"), value: disk.offline_uncorrectable },
                    { label: t("nodeDetail.diskHealthInterfaceCRC"), value: disk.interface_crc_errors },
                  ].filter((item) => typeof item.value === "number" && Number.isFinite(item.value) && item.value > 0)
                  return (
                    <div key={`${disk.name}-${disk.path ?? ""}`} className="contents text-sm text-slate-700 dark:text-slate-200">
                      <div className="border-b border-slate-100 px-3 py-3 dark:border-white/5">
                        <p className="font-semibold text-slate-900 dark:text-slate-50">{disk.name}</p>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{disk.model || disk.path || disk.type}</p>
                        <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{formatOptionalBytes(disk.size_bytes)}</p>
                        {mounts ? <p className="mt-1 text-xs text-slate-500 dark:text-slate-400">{t("nodeDetail.diskHealthMounts")} {mounts}</p> : null}
                        {mounts || disk.read_only ? (
                          <p className={`mt-1 text-xs font-medium ${disk.read_only ? "text-amber-700 dark:text-amber-300" : "text-emerald-700 dark:text-emerald-300"}`}>
                            {disk.read_only ? t("nodeDetail.diskHealthReadOnly") : t("nodeDetail.diskHealthWritable")}
                          </p>
                        ) : null}
                      </div>
                      <div className="border-b border-slate-100 px-3 py-3 dark:border-white/5">
                        <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-medium ${tone.className}`}>
                          <Icon className="h-3.5 w-3.5" />
                          {t(tone.labelKey)}
                        </span>
                        <span className={`mt-2 inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-medium ${support.className}`}>
                          <SupportIcon className="h-3.5 w-3.5" />
                          {t(support.labelKey)}
                        </span>
                        {message ? <p className="mt-2 max-w-[16rem] text-xs text-slate-500 dark:text-slate-400">{message}</p> : null}
                      </div>
                      <div className="border-b border-slate-100 px-3 py-3 tabular-nums dark:border-white/5">{formatOptionalTemperature(disk.temperature_c)}</div>
                      <div className="border-b border-slate-100 px-3 py-3 tabular-nums dark:border-white/5">{formatOptionalPercent(disk.life_used_percent)}</div>
                      <div className="border-b border-slate-100 px-3 py-3 tabular-nums dark:border-white/5">{formatOptionalPercent(disk.available_spare_percent)}</div>
                      <div className="border-b border-slate-100 px-3 py-3 tabular-nums dark:border-white/5">
                        <p>{formatOptionalCount(disk.media_errors)}</p>
                        {mediaErrorDetails.length > 0 ? (
                          <div className="mt-2 space-y-1 text-[11px] leading-tight text-slate-500 dark:text-slate-400">
                            {mediaErrorDetails.map((item) => (
                              <p key={item.label}>
                                {item.label} {formatOptionalCount(item.value)}
                              </p>
                            ))}
                          </div>
                        ) : null}
                      </div>
                      <div className="border-b border-slate-100 px-3 py-3 tabular-nums dark:border-white/5">{formatOptionalCount(disk.unsafe_shutdowns)}</div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </CollapsibleContent>
    </section>
  )
}
