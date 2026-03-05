type StatusFilter = "all" | "online" | "offline"

type Props = {
  status: StatusFilter
  search: string
  onStatusChange: (value: StatusFilter) => void
  onSearchChange: (value: string) => void
  onReset: () => void
}

export function NodeFilters({ status, search, onStatusChange, onSearchChange, onReset }: Props) {
  return (
    <section className="glass-panel rounded-xl p-3 flex flex-col md:flex-row md:items-end gap-3">
      <label className="text-xs text-white/60 flex flex-col gap-1.5">
        Status filter
        <select
          aria-label="Status filter"
          value={status}
          onChange={(event) => onStatusChange(event.target.value as StatusFilter)}
          className="bg-white/10 border border-white/20 rounded-md px-2 py-1.5 text-white"
        >
          <option value="all">All</option>
          <option value="online">Online</option>
          <option value="offline">Offline</option>
        </select>
      </label>

      <label className="text-xs text-white/60 flex-1 flex flex-col gap-1.5">
        Search nodes
        <input
          aria-label="Search nodes"
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search by node name"
          className="bg-white/10 border border-white/20 rounded-md px-3 py-1.5 text-white placeholder:text-white/35"
        />
      </label>

      <button
        onClick={onReset}
        className="h-10 w-full md:w-auto px-3 rounded-md border border-white/20 bg-white/10 text-xs text-white/80 hover:bg-white/15 transition-colors"
      >
        Reset filters
      </button>
    </section>
  )
}
