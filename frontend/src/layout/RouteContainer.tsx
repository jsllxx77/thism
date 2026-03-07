import { ArrowLeft } from "lucide-react"
import { Button } from "../components/ui/button"
import { useLanguage } from "../i18n/language"

type Props = {
  children: React.ReactNode
  showBack: boolean
  onBack: () => void
}

export function RouteContainer({ children, showBack, onBack }: Props) {
  const { t } = useLanguage()

  return (
    <div>
      {showBack && (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onBack}
          className="mb-6 h-11 border-slate-300 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800 sm:h-9"
        >
          <ArrowLeft aria-hidden />
          {t("shell.actions.backToDashboard")}
        </Button>
      )}
      {children}
    </div>
  )
}
