import { useMemo, useState, type ReactNode } from "react"
import { ArrowUpDown, ChevronUp } from "lucide-react"
import { useLanguage } from "../../i18n/language"
import type { Node } from "../../lib/api"
import { CountryFlag } from "../../components/CountryFlag"
import { NodeTagChips } from "../NodeTagChips"

type SortKey = "name" | "status"

type Props = {
  nodes: Node[]
  onSelectNode: (id: string) => void
}

function SortButton({
  active,
  ascending,
  children,
  onClick,
}: {
  active: boolean
  ascending: boolean
  children: ReactNode
  onClick: () => void
}) {
  const Icon = active ? ChevronUp : ArrowUpDown

  return (
    <button
      type="button"
      onClick={onClick}
      data-active={active}
      className="motion-sort-button text-left hover:text-slate-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 dark:hover:text-slate-200"
    >
      <span>{children}</span>
      <Icon className={`motion-sort-icon h-3.5 w-3.5 ${active && !ascending ? "rotate-180" : ""}`} />
    </button>
  )
}

function StatusPill({ online, label }: { online: boolean; label: string }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-semibold ${
        online
          ? "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-500/20 dark:bg-emerald-500/10 dark:text-emerald-300"
          : "border-[hsl(var(--destructive)/0.35)] bg-[hsl(var(--destructive)/0.12)] text-[hsl(var(--destructive))]"
      }`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${
          online ? "bg-emerald-500" : "bg-[hsl(var(--destructive))]"
        }`}
      />
      {label}
    </span>
  )
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
    <div className="motion-results-region panel-card enterprise-surface overflow-x-auto rounded-[24px] p-4">
      <p className="mb-3 text-sm font-semibold uppercase tracking-[0.16em] text-slate-700 dark:text-slate-100">{t("dashboard.table.title")}</p>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-[11px] uppercase tracking-[0.18em] text-slate-500 dark:border-slate-700 dark:text-slate-400">
            <th className="pb-2 pr-3 font-medium" aria-sort={sortKey === "name" ? (sortAsc ? "ascending" : "descending") : "none"}>
              <SortButton active={sortKey === "name"} ascending={sortAsc} onClick={() => toggleSort("name")}>
                {t("dashboard.table.nodeName")}
              </SortButton>
            </th>
            <th className="pb-2 pr-3 font-medium">{t("dashboard.table.agent")}</th>
            <th className="pb-2 pr-3 font-medium">{t("dashboard.table.tags")}</th>
            <th className="pb-2 pr-3 font-medium">{t("dashboard.table.ip")}</th>
            <th className="pb-2 pr-3 font-medium" aria-sort={sortKey === "status" ? (sortAsc ? "ascending" : "descending") : "none"}>
              <SortButton active={sortKey === "status"} ascending={sortAsc} onClick={() => toggleSort("status")}>
                {t("dashboard.table.status")}
              </SortButton>
            </th>
          </tr>
        </thead>
        <tbody className="motion-table-body">
          {sorted.map((node) => (
            <tr
              key={node.id}
              className={`motion-table-row border-b border-slate-100 dark:border-slate-800 ${
                !node.online ? "bg-[hsl(var(--destructive)/0.05)]" : ""
              }`}
            >
              <td className="py-2.5 pr-3 text-slate-900 dark:text-slate-100">
                <button
                  type="button"
                  onClick={() => onSelectNode(node.id)}
                  aria-label={t("dashboard.openNodeAria", { name: node.name })}
                  className="inline-flex max-w-full items-center rounded-sm text-left font-medium text-slate-900 transition-colors hover:text-slate-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:text-slate-100 dark:hover:text-slate-200 dark:focus-visible:ring-offset-slate-950"
                >
                  <CountryFlag countryCode={node.country_code} className="mr-1" />
                  <span className="truncate">{node.name}</span>
                </button>
              </td>
              <td className="py-2.5 pr-3 font-mono text-xs text-slate-600 dark:text-slate-300">{node.agent_version || "—"}</td>
              <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">
                <NodeTagChips tags={node.tags} />
              </td>
              <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">{node.ip || t("common.unavailable")}</td>
              <td className="py-2.5 pr-3 text-slate-600 dark:text-slate-300">
                <StatusPill online={node.online} label={node.online ? t("common.online") : t("common.offline")} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
