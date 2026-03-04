import { Monitor } from "lucide-react"

export function Layout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-[#0a0a0f] text-white flex flex-col">
      {/* Header */}
      <header className="border-b border-white/10 px-6 py-3 flex items-center gap-3 backdrop-blur-sm sticky top-0 z-10 bg-[#0a0a0f]/80">
        <Monitor className="w-5 h-5 text-emerald-400" />
        <span className="font-semibold tracking-tight text-sm">ThisM</span>
        <div className="ml-auto flex items-center gap-2 text-xs text-white/40">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse inline-block" />
          Live
        </div>
      </header>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
    </div>
  )
}
