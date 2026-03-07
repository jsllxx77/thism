import { Search } from "lucide-react"
import { useLanguage } from "../../i18n/language"
import { Input } from "../ui/input"

type StatusFilter = "all" | "online" | "offline"

type Props = {
  status: StatusFilter
  search: string
  onStatusChange: (value: StatusFilter) => void
  onSearchChange: (value: string) => void
  onReset: () => void
}

export function NodeFilters({ status, search, onStatusChange, onSearchChange, onReset }: Props) {
  const { t } = useLanguage()

  return (
    <section className="panel-card enterprise-surface rounded-[24px] p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-end">
        <label className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
          {t("dashboard.filters.status")}
          <select
            aria-label={t("dashboard.filters.status")}
            value={status}
            onChange={(event) => onStatusChange(event.target.value as StatusFilter)}
            className="enterprise-outline-control h-11 rounded-xl border px-3 py-1.5 text-sm shadow-none focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-slate-950 dark:text-slate-200 md:h-10"
          >
            <option value="all">{t("dashboard.filters.all")}</option>
            <option value="online">{t("dashboard.filters.online")}</option>
            <option value="offline">{t("dashboard.filters.offline")}</option>
          </select>
        </label>

        <label className="flex-1 flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
          {t("dashboard.filters.search")}
          <div className="relative">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400 dark:text-slate-500" />
            <Input
              aria-label={t("dashboard.filters.search")}
              value={search}
              onChange={(event) => onSearchChange(event.target.value)}
              placeholder={t("dashboard.filters.searchPlaceholder")}
              className="enterprise-outline-control h-11 rounded-xl border pl-9 text-slate-800 placeholder:text-slate-400 shadow-none dark:bg-slate-950 dark:text-slate-100 dark:placeholder:text-slate-500 md:h-10"
            />
          </div>
        </label>

        <button
          type="button"
          onClick={onReset}
          className="enterprise-outline-control h-11 w-full rounded-xl border px-4 text-xs font-medium uppercase tracking-[0.18em] text-slate-700 transition-colors hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 active:translate-y-px active:scale-[0.99] dark:bg-slate-950 dark:text-slate-200 dark:hover:bg-slate-900 dark:focus-visible:ring-offset-slate-950 md:h-10 md:w-auto"
        >
          {t("dashboard.filters.reset")}
        </button>
      </div>
    </section>
  )
}
