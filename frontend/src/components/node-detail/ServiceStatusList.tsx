import { useEffect, useState } from "react"
import { Collapse } from "antd"
import type { ServiceCheck } from "../../lib/api"

type Props = {
  services: ServiceCheck[]
  defaultOpen?: boolean
}

function serviceClass(status: string): string {
  if (status === "running") return "bg-emerald-500/15 border-emerald-400/30 text-emerald-300"
  if (status === "failed" || status === "dead") return "bg-red-500/15 border-red-400/30 text-red-300"
  return "bg-white/10 border-white/20 text-white/70"
}

const PANEL_KEY = "services"

function normalizeActiveKey(key: string | string[]): string[] {
  return Array.isArray(key) ? key : [key]
}

export function ServiceStatusList({ services, defaultOpen = false }: Props) {
  const [activeKey, setActiveKey] = useState<string[]>(defaultOpen ? [PANEL_KEY] : [])

  useEffect(() => {
    setActiveKey(defaultOpen ? [PANEL_KEY] : [])
  }, [defaultOpen])

  if (services.length === 0) {
    return null
  }

  return (
    <Collapse
      activeKey={activeKey}
      destroyOnHidden
      onChange={(key) => setActiveKey(normalizeActiveKey(key))}
      className="glass-panel !border-white/15 !bg-transparent"
      items={[
        {
          key: PANEL_KEY,
          label: "Services",
          children: (
            <div className="flex flex-wrap gap-2">
              {services.map((service) => (
                <div
                  key={service.name}
                  className={`text-xs px-3 py-1.5 rounded-md border ${serviceClass(service.status)}`}
                >
                  <span className="mr-2">{service.name}</span>
                  <span data-status={service.status}>{service.status}</span>
                </div>
              ))}
            </div>
          ),
        },
      ]}
    />
  )
}
