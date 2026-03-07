import { getPreferredLanguage } from "../i18n/language"

async function req<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      "Accept-Language": getPreferredLanguage(),
      ...(options?.headers ?? {}),
    },
  })
  if (!res.ok) {
    let message = `${res.status} ${res.statusText}`
    try {
      const payload = await res.json()
      if (payload && typeof payload.error === "string" && payload.error.trim()) {
        message = payload.error
      }
    } catch {
      // Ignore non-JSON error responses and keep status-based message.
    }
    throw new Error(message)
  }
  return res.json()
}

export type AccessMode = "admin" | "guest"

export type SessionInfo = {
  role: AccessMode
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
  hardware?: NodeHardware | null
  latest_metrics?: MetricsRow | null
}

export type NodeHardware = {
  cpu_model: string
  cpu_cores: number
  cpu_threads: number
  memory_total: number
  disk_total: number
  virtualization_system: string
  virtualization_role: string
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

export type AgentUpdateJob = {
  id: string
  kind: string
  target_version: string
  download_url: string
  sha256: string
  created_at: number
  created_by: string
  status: string
}

export type AgentUpdateJobTarget = {
  job_id: string
  node_id: string
  status: string
  message: string
  reported_version: string
  updated_at: number
}

export type UpdateJobStatus = "pending" | "running" | "completed" | "partial_failed" | "failed"
export type UpdateJobTargetStatus =
  | "pending"
  | "dispatched"
  | "accepted"
  | "downloading"
  | "verifying"
  | "restarting"
  | "succeeded"
  | "failed"
  | "timeout"
  | "offline_skipped"

export type UpdateJob = {
  id: string
  kind: string
  target_version: string
  download_url: string
  sha256: string
  created_at: number
  updated_at: number
  created_by: string
  status: UpdateJobStatus
}

export type UpdateJobTarget = {
  job_id: string
  node_id: string
  status: UpdateJobTargetStatus
  message?: string
  updated_at: number
  reported_version?: string
}

export type UpdateJobResponse = {
  job: UpdateJob
  targets: UpdateJobTarget[]
}

export type AgentReleaseManifest = {
  target_version: string
  download_url: string
  sha256: string
  check_interval_seconds: number
}

export const api = {
  session: () => req<SessionInfo>("/api/auth/session"),
  nodes: () => req<{ nodes: Node[] }>("/api/nodes"),
  renameNode: (id: string, name: string) =>
    req<Node>(`/api/nodes/${id}`, {
      method: "PATCH",
      body: JSON.stringify({ name }),
    }),
  deleteNode: (id: string) =>
    req<{ ok: boolean }>(`/api/nodes/${id}`, {
      method: "DELETE",
    }),
  installCommand: (id: string) =>
    req<{ command: string }>(`/api/nodes/${id}/install-command`),
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
  changePassword: (currentPassword: string, newPassword: string) =>
    req<{ ok: boolean }>("/api/auth/change-password", {
      method: "POST",
      body: JSON.stringify({
        current_password: currentPassword,
        new_password: newPassword,
      }),
    }),
  createAgentUpdateJob: (nodeIDs: string[], targetVersion: string, downloadURL: string, sha256: string) =>
    req<UpdateJobResponse>("/api/agent-updates", {
      method: "POST",
      body: JSON.stringify({
        node_ids: nodeIDs,
        target_version: targetVersion,
        download_url: downloadURL,
        sha256: sha256,
      }),
    }),
  getAgentUpdateJob: (id: string) => req<UpdateJobResponse>(`/api/agent-updates/${id}`),
  agentRelease: (os: string, arch: string) => req<AgentReleaseManifest>(`/api/agent-release?os=${encodeURIComponent(os)}&arch=${encodeURIComponent(arch)}`),
}
