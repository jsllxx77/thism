import type { ReactNode } from "react"

type Props = {
  children: ReactNode
  open: boolean
}

export function CollapsibleContent({ children, open }: Props) {
  if (!open) {
    return null
  }

  return (
    <div className="motion-collapsible-content" data-open="true">
      <div className="motion-collapsible-content__inner">{children}</div>
    </div>
  )
}
