import { Layers3, MapPinned } from "lucide-react"
import type { Node } from "../../lib/api"
import { useLanguage } from "../../i18n/language"
import { Badge } from "../ui/badge"
import { Card, CardContent } from "../ui/card"

type Props = {
  node: Node | null
  showIP?: boolean
}

export function NodeHero({ node, showIP = true }: Props) {
  const { t } = useLanguage()
  const platformLabel = `${node?.os || "—"}/${node?.arch || "—"}`

  return (
    <Card className="panel-card enterprise-hero rounded-[28px]">
      <CardContent className="p-5 md:p-6">
        <div className="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
          <div className="space-y-4">
            <div className="space-y-2">
              <p className="enterprise-kicker text-[11px] font-semibold uppercase tracking-[0.24em]">{t("nodeDetail.heroEyebrow")}</p>
              <h2 className="text-2xl font-semibold tracking-tight text-slate-950 dark:text-slate-100 md:text-[2rem]">{node?.name ?? t("common.unknownNode")}</h2>
              <p className="max-w-2xl text-sm text-slate-600 dark:text-slate-400">{t("nodeDetail.heroDescription")}</p>
            </div>

            <div className="flex flex-wrap items-center gap-2.5 text-[11px] font-medium tracking-wide text-slate-600 dark:text-slate-300">
              {showIP && (
                <div className="enterprise-chip inline-flex max-w-full items-center gap-2 rounded-full px-3 py-1.5">
                  <MapPinned className="h-3.5 w-3.5 text-slate-500 dark:text-slate-400" />
                  <span className="truncate">{node?.ip || "—"}</span>
                </div>
              )}
              <div className="enterprise-chip inline-flex items-center gap-2 rounded-full px-3 py-1.5">
                <Layers3 className="h-3.5 w-3.5 text-slate-500 dark:text-slate-400" />
                <span>{platformLabel}</span>
              </div>
            </div>
          </div>

          <div className="flex flex-col items-start gap-3 xl:items-end">
            <Badge
              variant={node?.online ? "secondary" : "outline"}
              className={
                node?.online
                  ? "self-start border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
                  : "self-start border-slate-300 bg-slate-50 text-slate-600 dark:border-white/10 dark:bg-slate-900 dark:text-slate-300"
              }
            >
              {node?.online ? t("common.online") : t("common.offline")}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
