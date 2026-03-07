import { useState } from "react"
import { useLanguage } from "../../i18n/language"
import type { Node } from "../../lib/api"
import { Button } from "../ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../ui/dialog"
import { Input } from "../ui/input"

type Props = {
  open: boolean
  nodes: Node[]
  submitting?: boolean
  error?: string | null
  onClose: () => void
  onSubmit: (values: { targetVersion: string; downloadURL: string; sha256: string }) => Promise<void> | void
}

export function BatchUpgradeDialog({ open, nodes, submitting = false, error, onClose, onSubmit }: Props) {
  const { t } = useLanguage()
  const [targetVersion, setTargetVersion] = useState("")
  const [downloadURL, setDownloadURL] = useState("")
  const [sha256, setSHA256] = useState("")

  const handleSubmit = async () => {
    await onSubmit({ targetVersion: targetVersion.trim(), downloadURL: downloadURL.trim(), sha256: sha256.trim() })
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onClose()}>
      <DialogContent className="enterprise-hero max-w-lg rounded-[28px] border p-6">
        <DialogHeader>
          <DialogTitle>{t("batchUpdate.dialogTitle")}</DialogTitle>
          <DialogDescription>{t("batchUpdate.dialogDescription")}</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <p className="text-xs font-medium text-slate-500 dark:text-slate-400">{t("batchUpdate.selectedNodes", { count: nodes.length })}</p>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
            {t("batchUpdate.targetVersion")}
            <Input aria-label={t("batchUpdate.targetVersion")} value={targetVersion} onChange={(event) => setTargetVersion(event.target.value)} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90" />
          </label>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
            {t("batchUpdate.downloadUrl")}
            <Input aria-label={t("batchUpdate.downloadUrl")} value={downloadURL} onChange={(event) => setDownloadURL(event.target.value)} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90" />
          </label>
          <label className="block text-xs font-medium text-slate-600 dark:text-slate-300">
            {t("batchUpdate.sha256")}
            <Input aria-label={t("batchUpdate.sha256")} value={sha256} onChange={(event) => setSHA256(event.target.value)} className="enterprise-outline-control mt-2 rounded-xl border dark:bg-slate-950/90" />
          </label>
          {error && <p role="alert" className="text-xs text-red-600 dark:text-red-300">{error}</p>}
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} className="rounded-xl dark:border-white/10 dark:bg-slate-950 dark:hover:bg-slate-900">
            {t("Cancel")}
          </Button>
          <Button type="button" onClick={() => void handleSubmit()} disabled={submitting} className="enterprise-accent-button rounded-xl">
            {submitting ? `${t("Loading")}...` : t("batchUpdate.start")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
