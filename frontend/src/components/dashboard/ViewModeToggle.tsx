import { useLanguage } from "../../i18n/language"

type ViewMode = "cards" | "table"

type Props = {
  mode: ViewMode
  onChange: (mode: ViewMode) => void
}

export function ViewModeToggle({ mode, onChange }: Props) {
  const { t } = useLanguage()

  return (
    <div className="enterprise-inner-surface inline-flex w-full gap-1 rounded-2xl p-1.5 sm:w-auto sm:p-1">
      <button
        type="button"
        onClick={() => onChange("cards")}
        className={`h-11 flex-1 rounded-xl border border-transparent px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] transition-all duration-200 focus:outline-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-inset active:translate-y-px active:scale-[0.99] sm:h-9 sm:flex-initial ${
          mode === "cards"
            ? "border border-slate-200/80 bg-slate-50/90 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50 dark:ring-1 dark:ring-inset dark:ring-white/10 dark:shadow-none"
            : "text-slate-600 hover:bg-white/85 dark:text-slate-200 dark:hover:bg-slate-900"
        }`}
      >
        {t("Cards View")}
      </button>
      <button
        type="button"
        onClick={() => onChange("table")}
        className={`h-11 flex-1 rounded-xl border border-transparent px-3 py-2 text-xs font-medium uppercase tracking-[0.16em] transition-all duration-200 focus:outline-none focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-inset active:translate-y-px active:scale-[0.99] sm:h-9 sm:flex-initial ${
          mode === "table"
            ? "border border-slate-200/80 bg-slate-50/90 text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-50 dark:ring-1 dark:ring-inset dark:ring-white/10 dark:shadow-none"
            : "text-slate-600 hover:bg-white/85 dark:text-slate-200 dark:hover:bg-slate-900"
        }`}
      >
        {t("Table View")}
      </button>
    </div>
  )
}
