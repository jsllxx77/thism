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
  const statusOptions: Array<{ value: StatusFilter; label: string }> = [
    { value: "all", label: t("dashboard.filters.all") },
    { value: "online", label: t("dashboard.filters.online") },
    { value: "offline", label: t("dashboard.filters.offline") },
  ]

  return (
    <section className="panel-card enterprise-surface rounded-[24px] p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-end">
        <div className="flex flex-col gap-1.5 text-xs font-medium text-slate-600 dark:text-slate-300">
          <span>{t("dashboard.filters.status")}</span>
          <div
            role="group"
            aria-label={t("dashboard.filters.status")}
            className="enterprise-inner-surface inline-flex w-full gap-1 rounded-2xl p-1.5 shadow-none md:w-auto md:p-1"
          >
            {statusOptions.map((option) => {
              const active = status === option.value
              return (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={active}
                  onClick={() => onStatusChange(option.value)}
                  className={`h-11 flex-1 cursor-pointer rounded-xl border border-transparent px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] transition-all duration-200 focus:outline-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-inset active:translate-y-px active:scale-[0.99] md:h-10 md:flex-initial ${
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
