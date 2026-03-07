import { useState } from "react"
import { Check, Copy, Loader2 } from "lucide-react"
import { Button } from "./ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "./ui/dialog"
import { Input } from "./ui/input"
import { api } from "../lib/api"
import { useLanguage } from "../i18n/language"
import { copyTextToClipboard } from "../lib/clipboard"

type Props = {
  onClose: () => void
  onCreated: () => Promise<void> | void
}

type ResultState = {
  token: string
  command: string
}

export function AddNodeModal({ onClose, onCreated }: Props) {
  const { t, translateError } = useLanguage()
  const [name, setName] = useState("")
  const [step, setStep] = useState<1 | 2>(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [result, setResult] = useState<ResultState | null>(null)
  const [copied, setCopied] = useState(false)

  const handleGenerate = async () => {
    if (!name.trim()) {
      setError(t("Node name is required"))
      return
    }

    setLoading(true)
    setError("")
    try {
      const response = await api.register(name.trim())
      const host = window.location.host
      const scheme = window.location.protocol === "https:" ? "https" : "http"
      const params = new URLSearchParams({ token: response.token, name: name.trim() })
      const command = `curl -fsSL "${scheme}://${host}/install.sh?${params.toString()}" | bash`

      setResult({ token: response.token, command })
      setStep(2)
      onCreated()
    } catch (requestError: unknown) {
      const message = requestError instanceof Error ? translateError(requestError.message) : t("Failed to register node")
      setError(message)
    } finally {
      setLoading(false)
    }
  }

  const copyCommand = async () => {
    if (!result) return
    setError("")

    const copyResult = await copyTextToClipboard(result.command)
    if (copyResult.ok) {
      setCopied(true)
      setTimeout(() => setCopied(false), 1200)
      return
    }

    setCopied(false)
    setError(translateError(copyResult.message))
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="enterprise-hero max-w-xl rounded-[28px] border p-6">
        <DialogHeader>
          <DialogTitle>{t("Node Provisioning")}</DialogTitle>
          <DialogDescription>{t("addNodeModal.stepDescription", { current: step, total: 2 })}</DialogDescription>
        </DialogHeader>

        {step === 1 && (
          <>
            <div className="enterprise-inner-surface rounded-2xl p-4">
              <label htmlFor="node-name" className="mb-1.5 block text-xs font-medium text-slate-600 dark:text-slate-300">
                {t("Node Name")}
              </label>
              <Input
                id="node-name"
                aria-label={t("Node Name")}
                type="text"
                value={name}
                onChange={(event) => setName(event.target.value)}
                onKeyDown={(event) => event.key === "Enter" && void handleGenerate()}
                placeholder={t("e.g. web-server-01")}
                className="enterprise-outline-control h-10 rounded-xl border text-slate-800 placeholder:text-slate-400 dark:bg-slate-950/90 dark:text-slate-100 dark:placeholder:text-slate-500"
                autoFocus
              />
            </div>
            {error && <p className="text-xs text-red-600 dark:text-red-300">{error}</p>}
            <Button
              onClick={() => void handleGenerate()}
              disabled={loading}
              className="enterprise-accent-button h-10 w-full rounded-xl"
            >
              {loading && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              {t("Generate command")}
            </Button>
          </>
        )}

        {step === 2 && result && (
          <>
            <div className="enterprise-inner-surface rounded-2xl p-4">
              <p className="mb-2 text-sm font-medium text-slate-800 dark:text-slate-100">{t("Install Command")}</p>
              <p className="mb-2 text-xs text-slate-500 dark:text-slate-400">{t("Run this on the target machine as root:")}</p>
              <code className="block break-all rounded-2xl border border-slate-200/80 bg-slate-50/90 px-3 py-2 font-mono text-xs text-slate-700 dark:border-white/8 dark:bg-slate-950 dark:text-slate-200">
                {result.command}
              </code>
            </div>

            <div className="enterprise-inner-surface rounded-2xl p-4">
              <p className="mb-1.5 text-xs text-slate-500 dark:text-slate-400">{t("Token")}</p>
              <code className="block break-all rounded-2xl border border-slate-200/80 bg-slate-50/90 px-3 py-2 font-mono text-xs text-slate-700 dark:border-white/8 dark:bg-slate-950 dark:text-slate-200">
                {result.token}
              </code>
            </div>

            <div className="flex flex-col gap-2 sm:flex-row">
              <Button
                type="button"
                onClick={() => void copyCommand()}
                aria-label={t("Copy command")}
                variant="outline"
                className="h-10 flex-1 rounded-xl border-slate-300 bg-white text-slate-700 hover:bg-slate-100 dark:border-white/10 dark:bg-slate-950 dark:text-slate-200 dark:hover:bg-slate-900"
              >
                {copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                {copied ? t("Copied") : t("Copy")}
              </Button>
              <Button
                type="button"
                onClick={onClose}
                className="enterprise-accent-button h-10 flex-1 rounded-xl"
              >
                {t("Done")}
              </Button>
            </div>
            {error && <p className="text-xs text-red-600 dark:text-red-300">{error}</p>}
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
