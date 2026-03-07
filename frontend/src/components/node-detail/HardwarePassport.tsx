import { Cpu, HardDrive, MemoryStick, ServerCog, Waypoints } from "lucide-react"
import type { NodeHardware } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { formatBytes } from "../../lib/units"

type Props = {
  hardware?: NodeHardware | null
  os?: string
  arch?: string
}

function formatValue(value: string | undefined): string {
  return value && value.trim() ? value : "—"
}

function formatCountPair(cores?: number, threads?: number): string {
  if (!cores && !threads) return "—"
  return `${cores || 0} / ${threads || 0}`
}

function formatVirtualization(hardware?: NodeHardware | null): string {
  if (!hardware) return "—"
  const system = hardware.virtualization_system?.trim()
  const role = hardware.virtualization_role?.trim()

  if (!system && !role) return "—"
  if (system && role) return `${system.toUpperCase()} · ${role}`
  if (system) return system.toUpperCase()
  return role || "—"
}

function formatPlatform(os?: string, arch?: string): string {
  const normalizedOS = os?.trim() || ""
  const normalizedArch = arch?.trim() || ""
  if (!normalizedOS && !normalizedArch) return "—"
  return `${normalizedOS || "—"} / ${normalizedArch || "—"}`
}

function formatHardwareBytes(value?: number): string {
  if (!value) return "—"
  return formatBytes(value)
}

type PassportItemProps = {
  icon: typeof Cpu
  label: string
  value: string
  accent: string
}

function PassportItem({ icon: Icon, label, value, accent }: PassportItemProps) {
  return (
    <div className="enterprise-inner-surface rounded-2xl p-4">
      <div className="flex items-center gap-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
        <span className={`inline-flex h-7 w-7 items-center justify-center rounded-full border ${accent}`}>
          <Icon className="h-3.5 w-3.5" />
        </span>
        {label}
      </div>
      <p className="mt-4 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-50">{value}</p>
    </div>
  )
}

export function HardwarePassport({ hardware, os, arch }: Props) {
  const { t } = useLanguage()
  const cpuModel = formatValue(hardware?.cpu_model)
  const coresThreads = formatCountPair(hardware?.cpu_cores, hardware?.cpu_threads)
  const memoryTotal = formatHardwareBytes(hardware?.memory_total)
  const diskTotal = formatHardwareBytes(hardware?.disk_total)
  const virtualization = formatVirtualization(hardware)
  const platform = formatPlatform(os, arch)

  return (
    <section className="panel-card enterprise-hero rounded-[28px] p-5">
      <div className="space-y-5">
        <div className="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
          <div>
            <p className="enterprise-kicker text-[11px] font-semibold uppercase tracking-[0.24em]">{t("nodeDetail.hardwarePassportEyebrow")}</p>
            <h3 className="mt-3 max-w-3xl text-xl font-semibold tracking-tight text-slate-950 md:text-2xl dark:text-slate-50">{cpuModel}</h3>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">{t("nodeDetail.hardwarePassportDescription")}</p>
          </div>
          <div className="enterprise-chip inline-flex items-center rounded-full px-3 py-1 text-[11px] font-medium uppercase tracking-[0.18em]">
            {t("nodeDetail.assetProfile")}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
          <PassportItem icon={Waypoints} label={t("nodeDetail.coresThreads")} value={coresThreads} accent="border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-200" />
          <PassportItem icon={MemoryStick} label={t("common.memory")} value={memoryTotal} accent="border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300" />
          <PassportItem icon={HardDrive} label={t("common.disk")} value={diskTotal} accent="border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300" />
          <PassportItem icon={ServerCog} label={t("nodeDetail.virtualization")} value={virtualization} accent="border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-200" />
          <PassportItem icon={Cpu} label={t("common.platform")} value={platform} accent="border-slate-200 bg-slate-100 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-200" />
        </div>
      </div>
    </section>
  )
}
