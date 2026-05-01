import { useMemo, useState } from "react"
import { useLanguage } from "../../i18n/language"
import type { Node } from "../../lib/api"
import { countryCodeToFlagEmoji } from "../../lib/flags"

type SortKey = "name" | "status"

type Props = {
  nodes: Node[]
  onSelectNode: (id: string) => void
}

export function NodeTable({ nodes, onSelectNode }: Props) {
  const { t } = useLanguage()
  const [sortKey, setSortKey] = useState<SortKey>("status")
  const [sortAsc, setSortAsc] = useState(true)

  const sorted = useMemo(() => {
    const list = [...nodes]
    list.sort((left, right) => {
      let value = 0
      if (sortKey === "name") {
        value = left.name.localeCompare(right.name)
      } else {
        value = Number(right.online) - Number(left.online)
      }
      return sortAsc ? value : -value
    })
    return list
  }, [nodes, sortAsc, sortKey])

  const toggleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortAsc((current) => !current)
      return
    }
    setSortKey(key)
    setSortAsc(true)
  }

  return (
    <div className="panel-card enterprise-surface overflow-x-auto rounded-[24px] p-4">
      <p className="mb-3 text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("dashboard.table.title")}</p>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-[0.18em] text-slate-500 dark:border-slate-700 dark:text-slate-400">
            <th className="pb-2 pr-3 font-medium">
              <button type="button" onClick={() => toggleSort("name")} className="text-left hover:text-slate-900 dark:hover:text-slate-200">
                {t("dashboard.table.nodeName")}
              </button>
            </th>
            <th className="pb-2 pr-3 font-medium">{t("dashboard.table.agent")}</th>
            <th className="pb-2 pr-3 font-medium">{t("dashboard.table.ip")}</th>
            <th className="pb-2 pr-3 font-medium">
              <button type="button" onClick={() => toggleSort("status")} className="text-left hover:text-slate-900 dark:hover:text-slate-200">
                {t("dashboard.table.status")}
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((node) => (
            <tr
              key={node.id}
              className="border-b border-slate-100 hover:bg-white/80 dark:border-slate-800 dark:hover:bg-white/[0.02]"
            >
              <td className="py-2.5 pr-3 text-slate-900 dark:text-slate-100">
                <button
                  type="button"
                  onClick={() => onSelectNode(node.id)}
                  aria-label={t("dashboard.openNodeAria", { name: node.name })}
                  className="rounded-sm text-left font-medium text-slate-900 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-slate-100 dark:hover:text-slate-200 dark:focus-visible:ring-offset-slate-950"
                >
                  {countryCodeToFlagEmoji(node.country_code) ? <span className="mr-1" aria-hidden="true">{countryCodeToFlagEmoji(node.country_code)}</span> : null}
                  <span>{node.name}</span>
                </button>
              </td>
              <td className="py-2.5 pr-3 font-mono text-xs text-slate-600 dark:text-slate-300">{node.agent_version || "—"}</td>
              <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{node.ip || t("common.unavailable")}</td>
              <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{node.online ? t("common.online") : t("common.offline")}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
