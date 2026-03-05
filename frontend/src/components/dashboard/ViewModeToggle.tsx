type ViewMode = "cards" | "table"

type Props = {
  mode: ViewMode
  onChange: (mode: ViewMode) => void
}

export function ViewModeToggle({ mode, onChange }: Props) {
  return (
    <div className="inline-flex w-full sm:w-auto rounded-lg border border-white/20 bg-white/5 p-1 gap-1">
      <button
        onClick={() => onChange("cards")}
        className={`flex-1 sm:flex-initial px-3 py-2 rounded-md text-xs transition-colors ${
          mode === "cards" ? "bg-emerald-500 text-slate-950 font-medium" : "text-white/70 hover:text-white"
        }`}
      >
        Cards
      </button>
      <button
        onClick={() => onChange("table")}
        className={`flex-1 sm:flex-initial px-3 py-2 rounded-md text-xs transition-colors ${
          mode === "table" ? "bg-emerald-500 text-slate-950 font-medium" : "text-white/70 hover:text-white"
        }`}
      >
        Table
      </button>
    </div>
  )
}
