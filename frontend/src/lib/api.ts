const TOKEN = (import.meta.env.VITE_ADMIN_TOKEN as string) ?? ""

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      Authorization: `Bearer ${TOKEN}`,
      "Content-Type": "application/json",
      ...(options?.headers ?? {}),
    },
  })
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
  return res.json()
}

export type Node = {
  id: string
  name: string
  ip: string
  os: string
  arch: string
  created_at: number
  last_seen: number
  online: boolean
}

export type MetricsRow = {
  ts: number
  cpu: number
  mem_used: number
  mem_total: number
  disk_used: number
  disk_total: number
  net_rx: number
  net_tx: number
}

export type Process = {
  pid: number
  name: string
  cpu: number
  mem: number
}

export type ServiceCheck = {
  name: string
  status: string
  last_checked: number
}

export const api = {
  nodes: () => req<{ nodes: Node[] }>("/api/nodes"),
  metrics: (id: string, from?: number, to?: number) => {
    const params = new URLSearchParams()
    if (from != null) params.set("from", String(from))
    if (to != null) params.set("to", String(to))
    const qs = params.toString()
    return req<{ metrics: MetricsRow[] }>(`/api/nodes/${id}/metrics${qs ? `?${qs}` : ""}`)
  },
  processes: (id: string) => req<Process[]>(`/api/nodes/${id}/processes`),
  services: (id: string) => req<{ services: ServiceCheck[] }>(`/api/nodes/${id}/services`),
  register: (name: string) =>
    req<{ id: string; token: string }>("/api/nodes/register", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
}
