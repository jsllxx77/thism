type Props = {
  children: React.ReactNode
  showBack: boolean
  onBack: () => void
}

export function RouteContainer({ children, showBack, onBack }: Props) {
  return (
    <div>
      {showBack && (
        <button
          onClick={onBack}
          className="text-xs text-white/40 hover:text-white mb-4 flex items-center gap-1 transition-colors"
        >
          ← Back to Dashboard
        </button>
      )}
      {children}
    </div>
  )
}
