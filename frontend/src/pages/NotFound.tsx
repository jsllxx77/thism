import { AlertCircle } from "lucide-react"
import { useNavigate } from "react-router-dom"
import { useLanguage } from "../i18n/language"

export function NotFound() {
  const navigate = useNavigate()
  const { t } = useLanguage()

  return (
    <section className="panel-card rounded-2xl border border-slate-200 px-6 py-14 text-center dark:border-slate-700">
      <div className="mx-auto flex max-w-xl flex-col items-center">
        <AlertCircle className="h-9 w-9 text-slate-500 dark:text-slate-400" aria-hidden />
        <h1 className="mt-3 text-3xl font-semibold tracking-tight text-slate-900 dark:text-slate-100">{t("notFound.title")}</h1>
        <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">{t("notFound.description")}</p>
        <button
          type="button"
          onClick={() => navigate("/", { replace: true })}
          className="mt-5 rounded-md border border-slate-300 bg-white px-4 py-2 text-sm font-medium text-slate-700 transition-colors hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800"
        >
          {t("notFound.back")}
        </button>
      </div>
    </section>
  )
}
