import { useEffect, useState } from "react"
import { ChevronDown } from "lucide-react"
import type { DockerContainer } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { Card, CardContent } from "../ui/card"

type Props = {
  containers: DockerContainer[]
  defaultOpen?: boolean
}

export function DockerContainerTable({ containers, defaultOpen = false }: Props) {
  const { t } = useLanguage()
  const [open, setOpen] = useState(defaultOpen)

  useEffect(() => {
    setOpen(defaultOpen)
  }, [defaultOpen])

  return (
    <Card className="panel-card enterprise-surface rounded-[24px]">
      <CardContent className="p-4">
        <button type="button" onClick={() => setOpen((value) => !value)} className="flex w-full items-center justify-between text-left">
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("nodeDetail.dockerSnapshot")}</span>
            <span className="inline-flex min-w-6 items-center justify-center rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-600 dark:bg-slate-800/80 dark:text-slate-300">
              {containers.length}
            </span>
          </div>
          <ChevronDown className={`h-4 w-4 text-slate-500 transition-transform dark:text-slate-400 ${open ? "rotate-180" : ""}`} />
        </button>

        {open && (
          <div className="enterprise-inner-surface mt-3 overflow-x-auto rounded-2xl p-2">
            {containers.length === 0 ? (
              <p className="px-2 py-3 text-xs text-slate-500 dark:text-slate-400">{t("nodeDetail.dockerEmpty")}</p>
            ) : (
              <table className="w-full text-xs">
                <thead>
                  <tr className="border-b border-slate-200 text-slate-500 dark:border-slate-700 dark:text-slate-400">
                    <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.dockerName")}</th>
                    <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.dockerImage")}</th>
                    <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.dockerState")}</th>
                    <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.dockerStatus")}</th>
                    <th className="px-2 py-2 text-left font-medium">{t("nodeDetail.dockerId")}</th>
                  </tr>
                </thead>
                <tbody>
                  {containers.slice(0, 25).map((container) => (
                    <tr key={container.id} className="border-b border-slate-100 last:border-b-0 dark:border-slate-800">
                      <td className="max-w-[140px] truncate px-2 py-2 text-slate-800 dark:text-slate-100">
                        {container.name || "—"}
                      </td>
                      <td className="max-w-[220px] truncate px-2 py-2 text-slate-700 dark:text-slate-200">{container.image}</td>
                      <td className="px-2 py-2 text-slate-600 dark:text-slate-300">{container.state}</td>
                      <td className="max-w-[220px] truncate px-2 py-2 text-slate-600 dark:text-slate-300">{container.status}</td>
                      <td className="font-mono px-2 py-2 text-slate-600 dark:text-slate-300">{container.id}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

