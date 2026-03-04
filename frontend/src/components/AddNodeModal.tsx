import { useState } from "react"
import { api } from "../lib/api"
import { Copy, Check, X, Loader2 } from "lucide-react"

type Props = {
  onClose: () => void
  onCreated: () => void
}

export function AddNodeModal({ onClose, onCreated }: Props) {
  const [name, setName] = useState("")
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<{ token: string; command: string } | null>(null)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState("")

  const handleSubmit = async () => {
    if (!name.trim()) return
    setLoading(true)
    setError("")
    try {
      const res = await api.register(name.trim())
      const host = window.location.host
      const scheme = window.location.protocol === "https:" ? "https" : "http"
      const command = `curl -sL ${scheme}://${host}/install.sh | bash -s -- --token ${res.token} --name ${name.trim()}`
      setResult({ token: res.token, command })
      onCreated()
    } catch (e: any) {
      setError(e.message || "Failed to register node")
    } finally {
      setLoading(false)
    }
  }

  const copyCommand = async () => {
    if (!result) return
    await navigator.clipboard.writeText(result.command)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-[#12121a] border border-white/10 rounded-2xl p-6 w-full max-w-lg mx-4 space-y-4"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold">Add Node</h3>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-white/10 text-white/40 hover:text-white transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {!result ? (
          <>
            <div>
              <label className="text-xs text-white/40 block mb-1.5">Node Name</label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSubmit()}
                placeholder="e.g. web-server-01"
                className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/30"
                autoFocus
              />
            </div>
            {error && <p className="text-xs text-red-400">{error}</p>}
            <button
              onClick={handleSubmit}
              disabled={!name.trim() || loading}
              className="w-full bg-emerald-500/20 border border-emerald-500/30 text-emerald-400 rounded-lg px-4 py-2 text-sm font-medium hover:bg-emerald-500/30 transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {loading && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              Generate Install Command
            </button>
          </>
        ) : (
          <>
            <div>
              <label className="text-xs text-white/40 block mb-1.5">Token</label>
              <code className="block bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-xs text-white/70 font-mono break-all">
                {result.token}
              </code>
            </div>
            <div>
              <label className="text-xs text-white/40 block mb-1.5">Install Command (run on target machine as root)</label>
              <div className="relative">
                <code className="block bg-white/5 border border-white/10 rounded-lg px-3 py-2 pr-10 text-xs text-emerald-400 font-mono break-all">
                  {result.command}
                </code>
                <button
                  onClick={copyCommand}
                  className="absolute top-2 right-2 p-1 rounded hover:bg-white/10 text-white/40 hover:text-white transition-colors"
                >
                  {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>
            </div>
            <button
              onClick={onClose}
              className="w-full bg-white/5 border border-white/10 text-white/70 rounded-lg px-4 py-2 text-sm hover:bg-white/10 transition-colors"
            >
              Done
            </button>
          </>
        )}
      </div>
    </div>
  )
}
