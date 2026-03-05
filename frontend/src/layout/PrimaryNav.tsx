import { Button } from "antd"

export type NavItem = {
  key: string
  label: string
  path: string
  icon: React.ReactNode
}

type Props = {
  items: NavItem[]
  activeKey: string
  onNavigate: (path: string) => void
}

export function PrimaryNav({ items, activeKey, onNavigate }: Props) {
  return (
    <nav aria-label="Primary Navigation" className="flex flex-col gap-2">
      {items.map((item) => {
        const isActive = item.key === activeKey
        return (
          <Button
            key={item.key}
            type={isActive ? "primary" : "default"}
            icon={item.icon}
            onClick={() => onNavigate(item.path)}
            aria-current={isActive ? "page" : undefined}
            className={`justify-start !h-10 ${
              isActive
                ? "!bg-emerald-500 !border-emerald-400/60 !text-slate-950"
                : "!bg-white/5 !border-white/15 !text-white/75 hover:!text-white"
            }`}
          >
            {item.label}
          </Button>
        )
      })}
    </nav>
  )
}
