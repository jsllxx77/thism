import { useEffect, useState } from "react"
import { ChevronDown } from "lucide-react"
import type { Process } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { formatBytes } from "../../lib/units"
import { Card, CardContent } from "../ui/card"

type Props = {
  processes: Process[]
  defaultOpen?: boolean
}

export function ProcessTable({ processes, defaultOpen = false }: Props) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(defaultOpen)

  useEffect(() => {
    setOpen(defaultOpen)
  }, [defaultOpen])

  if (processes.length === 0) {
    return null
  }

  return (
    <Card className="panel-card enterprise-surface rounded-[24px]">
      <CardContent className="p-4">
        <button type="button" onClick={() => setOpen((value) => !value)} className="flex w-full items-center justify-between text-left">
          <span className="text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("nodeDetail.processSnapshot")}</span>
          <ChevronDown className={`h-4 w-4 text-slate-500 transition-transform dark:text-slate-400 ${open ? "rotate-180" : ""}`} />
        </button>

        {open && (
          <div className="enterprise-inner-surface mt-3 overflow-x-auto rounded-2xl p-2">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-slate-200 text-slate-500 dark:border-slate-700 dark:text-slate-400">
                  <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.processName")}</th>
                  <th className="px-2 py-2 text-right font-medium">{t("nodeDetail.processPid")}</th>
                  <th className="px-2 py-2 text-right font-medium">{t("nodeDetail.processCpu")}</th>
                  <th className="px-2 py-2 text-right font-medium">{t("nodeDetail.processMemory")}</th>
                </tr>
              </thead>
              <tbody>
                {processes.slice(0, 15).map((process) => (
                  <tr key={process.pid} className="border-b border-slate-100 last:border-b-0 dark:border-slate-800">
                    <td className="max-w-[180px] truncate px-2 py-2 text-slate-800 dark:text-slate-100">{process.name}</td>
                    <td className="px-2 py-2 text-right text-slate-600 dark:text-slate-300">{process.pid}</td>
                    <td className="px-2 py-2 text-right text-slate-700 dark:text-slate-200">{process.cpu.toFixed(2)}%</td>
                    <td className="px-2 py-2 text-right text-slate-700 dark:text-slate-200">{formatBytes(process.mem)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
