import { useEffect, useState } from "react"
import { ChevronDown } from "lucide-react"
import type { ServiceCheck } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { Card, CardContent } from "../ui/card"

type Props = {
  services: ServiceCheck[]
  defaultOpen?: boolean
}

function serviceClass(status: string): string {
  if (status === "running") return "border-emerald-200 bg-emerald-50/90 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
  if (status === "failed" || status === "dead") return "border-red-200 bg-red-50/90 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300"
  return "border-slate-300 bg-slate-50/90 text-slate-600 dark:border-white/10 dark:bg-slate-900 dark:text-slate-300"
}

export function ServiceStatusList({ services, defaultOpen = false }: Props) {
  const { t, translateServiceStatus } = useLanguage()
  const [open, setOpen] = useState(defaultOpen)

  useEffect(() => {
    setOpen(defaultOpen)
  }, [defaultOpen])

  if (services.length === 0) {
    return null
  }

  return (
    <Card className="panel-card enterprise-surface rounded-[24px]">
      <CardContent className="p-4">
        <button type="button" onClick={() => setOpen((value) => !value)} className="flex w-full items-center justify-between text-left">
          <span className="text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("nodeDetail.serviceHealth")}</span>
          <ChevronDown className={`h-4 w-4 text-slate-500 transition-transform dark:text-slate-400 ${open ? "rotate-180" : ""}`} />
        </button>

        {open && (
          <div className="enterprise-inner-surface mt-3 flex flex-wrap gap-2 rounded-2xl p-3">
            {services.map((service) => (
              <div key={service.name} className={`rounded-full border px-3 py-1.5 text-xs font-medium ${serviceClass(service.status)}`}>
                <span className="mr-2">{service.name}</span>
                <span data-status={service.status}>{translateServiceStatus(service.status)}</span>
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
