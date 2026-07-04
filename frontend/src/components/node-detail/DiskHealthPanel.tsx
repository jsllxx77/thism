import { AlertTriangle, CheckCircle2, HardDrive, HelpCircle, XCircle } from "lucide-react"
import type { DiskHealthStats, DiskHealthStatus } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { formatBytes } from "../../lib/units"

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

export function DiskHealthPanel({ disks = [] }: Props) {
  const { t } = useLanguage()

  return (
    <section className="panel-card enterprise-surface rounded-[24px] p-4" aria-labelledby="disk-health-heading">
      <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="enterprise-kicker text-[11px] font-semibold uppercase tracking-[0.24em]">{t("nodeDetail.diskHealthEyebrow")}</p>
          <h3 id="disk-health-heading" className="mt-2 text-sm font-semibold text-slate-800 dark:text-slate-100">
            {t("nodeDetail.diskHealthTitle")}
          </h3>
        </div>
        <div className="enterprise-chip inline-flex w-fit items-center gap-2 rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em]">
          <HardDrive className="h-3.5 w-3.5" />
          {t("nodeDetail.diskHealthDeviceCount", { count: disks.length })}
        </div>
      </div>

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
                  </div>
                  <div className="border-b border-slate-100 px-3 py-3 dark:border-white/5">
                    <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-1 text-xs font-medium ${tone.className}`}>
                      <Icon className="h-3.5 w-3.5" />
                      {t(tone.labelKey)}
                    </span>
                    {disk.message ? <p className="mt-2 max-w-[16rem] text-xs text-slate-500 dark:text-slate-400">{disk.message}</p> : null}
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
    </section>
  )
}
