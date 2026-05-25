type Props = {
  tags?: string[]
  limit?: number
  emptyLabel?: string
  className?: string
}

export function NodeTagChips({ tags = [], limit = 3, emptyLabel, className = "" }: Props) {
  const normalizedTags = Array.isArray(tags) ? tags : []
  const visibleTags = normalizedTags.slice(0, limit)
  const overflow = Math.max(0, normalizedTags.length - visibleTags.length)

  if (visibleTags.length === 0) {
    if (!emptyLabel) return null
    return <span className={`text-xs text-slate-400 dark:text-slate-500 ${className}`}>{emptyLabel}</span>
  }

  return (
    <span className={`flex min-w-0 flex-wrap gap-1 ${className}`} aria-label={`Tags: ${normalizedTags.join(", ")}`}>
      {visibleTags.map((tag) => (
        <span
          key={tag}
          className="inline-flex max-w-[7rem] items-center truncate rounded-md border border-slate-200 bg-slate-50 px-1.5 py-0.5 text-[11px] font-medium text-slate-600 dark:border-white/10 dark:bg-slate-900 dark:text-slate-300"
        >
          {tag}
        </span>
      ))}
      {overflow > 0 && (
        <span className="inline-flex items-center rounded-md border border-slate-200 bg-white px-1.5 py-0.5 text-[11px] font-medium text-slate-500 dark:border-white/10 dark:bg-slate-950 dark:text-slate-400">
          +{overflow}
        </span>
      )}
    </span>
  )
}
