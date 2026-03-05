import { useState } from "react"
import { api } from "../lib/api"
import { Check, Copy, Loader2, X } from "lucide-react"

type Props = {
  onClose: () => void
  onCreated: () => void
}

type ResultState = {
  token: string
  command: string
}

function errorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message
  }
  return "Failed to register node"
}

export function AddNodeModal({ onClose, onCreated }: Props) {
  const [name, setName] = useState("")
  const [step, setStep] = useState<1 | 2>(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [result, setResult] = useState<ResultState | null>(null)
  const [copied, setCopied] = useState(false)

  const handleGenerate = async () => {
    if (!name.trim()) {
      setError("Node name is required")
      return
    }

    setLoading(true)
    setError("")
    try {
      const response = await api.register(name.trim())
      const host = window.location.host
      const scheme = window.location.protocol === "https:" ? "https" : "http"
      const command = `curl -sL ${scheme}://${host}/install.sh | bash -s -- --token ${response.token} --name ${name.trim()}`

      setResult({ token: response.token, command })
      setStep(2)
      onCreated()
    } catch (requestError: unknown) {
      setError(errorMessage(requestError))
    } finally {
      setLoading(false)
    }
  }

  const copyCommand = async () => {
    if (!result) return
    await navigator.clipboard.writeText(result.command)
    setCopied(true)
    setTimeout(() => setCopied(false), 1200)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="glass-panel rounded-2xl p-6 w-full max-w-xl mx-4 space-y-4"
        onClick={(event) => event.stopPropagation()}
      >
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-base font-semibold text-white">Add Node</h3>
            <p className="text-xs text-white/50 mt-0.5">Step {step} of 2</p>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-white/10 text-white/40 hover:text-white transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {step === 1 && (
          <>
            <div>
              <label htmlFor="node-name" className="text-xs text-white/55 block mb-1.5">
                Node Name
              </label>
              <input
                id="node-name"
                aria-label="Node Name"
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                onKeyDown={(event) => event.key === "Enter" && void handleGenerate()}
                placeholder="e.g. web-server-01"
                className="w-full bg-white/5 border border-white/15 rounded-lg px-3 py-2 text-sm text-white placeholder:text-white/25 focus:outline-none focus:border-white/40"
                autoFocus
              />
            </div>
            {error && <p className="text-xs text-red-300">{error}</p>}
            <button
              onClick={() => void handleGenerate()}
              disabled={loading}
              className="w-full bg-emerald-500/25 border border-emerald-400/30 text-emerald-300 rounded-lg px-4 py-2 text-sm font-medium hover:bg-emerald-500/35 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {loading && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
              Generate Install Command
            </button>
          </>
        )}

        {step === 2 && result && (
          <>
            <div>
              <p className="text-sm font-medium text-white mb-2">Install Command</p>
              <p className="text-xs text-white/55 mb-2">Run this on the target machine as root:</p>
              <code className="block bg-black/25 border border-white/15 rounded-lg px-3 py-2 text-xs text-emerald-300 font-mono break-all">
                {result.command}
              </code>
            </div>

            <div>
              <p className="text-xs text-white/55 mb-1.5">Token</p>
              <code className="block bg-black/25 border border-white/15 rounded-lg px-3 py-2 text-xs text-white/80 font-mono break-all">
                {result.token}
              </code>
            </div>

            <div className="flex flex-col sm:flex-row gap-2">
              <button
                onClick={() => void copyCommand()}
                aria-label="Copy command"
                className="h-10 flex-1 bg-white/10 border border-white/20 text-white/80 rounded-lg px-4 py-2 text-sm hover:bg-white/15 transition-colors flex items-center justify-center gap-2"
              >
                {copied ? <Check className="w-3.5 h-3.5 text-emerald-300" /> : <Copy className="w-3.5 h-3.5" />}
                {copied ? "Copied" : "Copy"}
              </button>
              <button
                onClick={onClose}
                className="h-10 flex-1 bg-emerald-500/25 border border-emerald-400/30 text-emerald-300 rounded-lg px-4 py-2 text-sm hover:bg-emerald-500/35 transition-colors"
              >
                Done
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
