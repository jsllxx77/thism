import { useState } from "react"
import { Button, Drawer } from "antd"
import { MenuOutlined } from "@ant-design/icons"
import { PrimaryNav, type NavItem } from "./PrimaryNav"

type Props = {
  items: NavItem[]
  activeKey: string
  onNavigate: (path: string) => void
}

export function MobileNav({ items, activeKey, onNavigate }: Props) {
  const [open, setOpen] = useState(false)

  const handleNavigate = (path: string) => {
    onNavigate(path)
    setOpen(false)
  }

  return (
    <>
      <Button
        icon={<MenuOutlined />}
        onClick={() => setOpen(true)}
        aria-label="Open navigation"
        className="md:!hidden !bg-white/5 !border-white/20 !text-white/85"
      />
      <Drawer
        title="Navigate"
        open={open}
        onClose={() => setOpen(false)}
        className="md:!hidden"
        styles={{
          body: {
            background: "linear-gradient(165deg, #0a1020 0%, #0f172a 100%)",
          },
          header: {
            background: "rgba(10, 16, 32, 0.92)",
            borderBottom: "1px solid rgba(255, 255, 255, 0.14)",
          },
        }}
      >
        <PrimaryNav items={items} activeKey={activeKey} onNavigate={handleNavigate} />
      </Drawer>
    </>
  )
}
