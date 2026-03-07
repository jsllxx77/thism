import { useLanguage } from "../../i18n/language"
import { Card, CardContent } from "../ui/card"

type Props = {
  inboundTotal: string
  outboundTotal: string
  inboundSpeed: string
  outboundSpeed: string
}

type SummaryItem = {
  label: string
  value: string
  accentClass: string
}

export function NetworkSummary({
  inboundTotal,
  outboundTotal,
  inboundSpeed,
  outboundSpeed,
}: Props) {
  const { t } = useLanguage()
  const items: SummaryItem[] = [
    { label: t("nodeDetail.inboundTotal"), value: inboundTotal, accentClass: "text-slate-900 dark:text-slate-50" },
    { label: t("nodeDetail.outboundTotal"), value: outboundTotal, accentClass: "text-slate-900 dark:text-slate-50" },
    { label: t("nodeDetail.inboundSpeed"), value: inboundSpeed, accentClass: "text-[#496fae] dark:text-[#8fb0e8]" },
    { label: t("nodeDetail.outboundSpeed"), value: outboundSpeed, accentClass: "text-emerald-700 dark:text-emerald-300" },
  ]

  return (
    <section aria-label={t("nodeDetail.networkSummaryAria")}>
      <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {items.map((item) => (
          <li key={item.label}>
            <Card className="panel-card enterprise-surface rounded-[24px]">
              <CardContent className="p-4">
                <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500 dark:text-slate-400">{item.label}</p>
                <p className={`mt-2 text-lg font-semibold tracking-tight sm:text-xl ${item.accentClass}`}>{item.value}</p>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
    </section>
  )
}
