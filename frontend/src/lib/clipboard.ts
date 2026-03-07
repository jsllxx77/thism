export const CLIPBOARD_COPY_DENIED_MESSAGE = "Clipboard access was denied. Please copy the command manually."

export type ClipboardCopyResult =
  | { ok: true }
  | { ok: false; message: string }

export async function copyTextToClipboard(text: string): Promise<ClipboardCopyResult> {
  if (!navigator.clipboard?.writeText) {
    return { ok: false, message: CLIPBOARD_COPY_DENIED_MESSAGE }
  }

  try {
    await navigator.clipboard.writeText(text)
    return { ok: true }
  } catch {
    return { ok: false, message: CLIPBOARD_COPY_DENIED_MESSAGE }
  }
}
