import { getPreferredLanguage } from "../i18n/language"

const CSRF_REQUIRED_ERROR = "csrf token required"
const SAFE_METHODS = new Set(["GET", "HEAD", "OPTIONS", "TRACE"])

async function req<T>(path: string, options?: RequestInit, allowCSRFRefresh = true): Promise<T> {
  const csrfToken = csrfTokenFromCookie()
  const method = options?.method?.toUpperCase() ?? "GET"
  const headers = new Headers(options?.headers)
  headers.set("Content-Type", "application/json")
  headers.set("Accept-Language", getPreferredLanguage())
  if (csrfToken && !SAFE_METHODS.has(method)) {
    headers.set("X-CSRF-Token", csrfToken)
  }
  const res = await fetch(path, {
    ...options,
    headers,
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
    if (allowCSRFRefresh && message === CSRF_REQUIRED_ERROR && !SAFE_METHODS.has(method)) {
      await req<SessionInfo>("/api/auth/session", undefined, false)
      return req<T>(path, options, false)
    }
    throw new Error(message)
  }
  return res.json()
}

function csrfTokenFromCookie(): string {
  if (typeof document === "undefined") {
    return ""
  }
  const match = document.cookie.split(";").map((part) => part.trim()).find((part) => part.startsWith("thism_csrf="))
  if (!match) {
    return ""
  }
  return decodeURIComponent(match.slice("thism_csrf=".length))
}

export type AccessMode = "admin" | "guest"

export type SessionInfo = {
  role: AccessMode
}

export type Node = {
  id: string
  name: string
  ip: string
  ip_families?: string[]
  os: string
  arch: string
  country_code?: string
  agent_version?: string
  created_at: number
  last_seen: number
  online: boolean
  tags?: string[]
  hardware?: NodeHardware | null
  latest_metrics?: MetricsRow | null
}

export type NodeUpdatePayload = {
  name?: string
  tags?: string[]
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
  uptime_seconds?: number
}

export type LatencyMonitorType = "icmp" | "tcp" | "http"

export type LatencyMonitor = {
  id: string
  name: string
  type: LatencyMonitorType
  target: string
  interval_seconds: number
  auto_assign_new_nodes: boolean
  assigned_node_count: number
  assigned_node_ids: string[]
  created_at: number
  updated_at: number
}

export type LatencyMonitorResult = {
  monitor_id: string
  node_id: string
  ts: number
  latency_ms: number | null
  loss_percent?: number | null
  jitter_ms?: number | null
  success: boolean
  error_message?: string
}

export type LatencyMonitorHistory = {
  monitors: LatencyMonitor[]
  results: LatencyMonitorResult[]
}

export type LatencyMonitorUpsertPayload = {
  name: string
  type: LatencyMonitorType
  target: string
  interval_seconds: number
  auto_assign_new_nodes: boolean
  node_ids: string[]
}

export type Process = {
  pid: number
  name: string
  cpu: number
  mem: number
}

export type DockerContainer = {
  id: string
  name: string
  image: string
  state: string
  status: string
}

export type DockerSnapshot = {
  docker_available: boolean
  containers: DockerContainer[]
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
  signature: string
  check_interval_seconds: number
}

export type MetricsRetentionSettings = {
  retention_days: number
  options: number[]
}

export type DashboardSettings = {
  show_dashboard_card_ip: boolean
}

export type PublicURLSettings = {
  public_url: string
}

export type TelegramTarget = {
  name?: string
  chat_id: string
  topic_id?: number
}

export type NotificationSettings = {
  enabled: boolean
  channel: "telegram"
  telegram_bot_token?: string
  telegram_bot_token_set: boolean
  telegram_targets: TelegramTarget[]
  time_zone_mode?: "system" | "custom"
  time_zone?: string
  system_time_zone?: string
  effective_time_zone?: string
  enabled_node_ids: string[]
  node_scope_mode?: "all" | "include" | "exclude"
  node_scope_node_ids?: string[]
  cpu_warning_percent: number
  cpu_critical_percent: number
  mem_warning_percent: number
  mem_critical_percent: number
  disk_warning_percent: number
  disk_critical_percent: number
  cooldown_minutes: number
  notify_node_offline: boolean
  notify_node_online: boolean
  node_offline_grace_minutes: number
  dispatcher_queue_capacity: number
  notify_dispatcher_drops: boolean
}

export type TestNotificationRequest = {
  telegram_bot_token?: string
  target?: TelegramTarget
}

export type TestNotificationResponse = {
  ok: boolean
}

export type VersionMeta = {
  version: string
  commit: string
  build_time: string
}

export type DispatcherRuntimeStats = {
  active_dispatchers: number
  total_capacity: number
  queue_depth: number
  high_watermark: number
  enqueued: number
  processed: number
  dropped: number
}

export type FrontendSkinSource = "built-in" | "custom"

export type FrontendSkin = {
  id: string
  name: string
  description?: string
  source: FrontendSkinSource
  entry: string
  api_version: string
  preview?: string
}

export type FrontendSkinsResponse = {
  active_skin_id: string
  skins: FrontendSkin[]
}

export type FrontendSkinInstallResponse = {
  active_skin_id: string
  skin: FrontendSkin
}

export type FrontendSkinSelectResponse = {
  active_skin_id: string
}

export type AvailabilityReportRange = {
  from: number
  to: number
}

export type AvailabilityReportFilter = {
  tag?: string
}

export type AvailabilityReportOverview = {
  total_nodes: number
  average_availability_percent: number
  nodes_below_99: number
  total_offline_duration_seconds: number
  highest_latency_p95_ms?: number | null
}

export type NodeAvailabilityReport = {
  node_id: string
  name: string
  tags: string[]
  last_seen: number
  availability_percent: number
  expected_samples: number
  observed_samples: number
  offline_duration_seconds: number
  outage_count: number
  last_outage_start?: number | null
  last_outage_end?: number | null
  latency_p50_ms?: number | null
  latency_p95_ms?: number | null
}

export type AvailabilityReport = {
  range: AvailabilityReportRange
  filter: AvailabilityReportFilter
  available_tags: string[]
  overview: AvailabilityReportOverview
  nodes: NodeAvailabilityReport[]
}

export const api = {
  session: () => req<SessionInfo>("/api/auth/session"),
  nodes: () => req<{ nodes: Node[] }>("/api/nodes"),
  node: (id: string) => req<{ node: Node | null }>(`/api/nodes/${id}`),
  updateNode: (id: string, payload: NodeUpdatePayload) =>
    req<Node>(`/api/nodes/${id}`, {
      method: "PATCH",
      body: JSON.stringify(payload),
    }),
  renameNode: (id: string, name: string) => api.updateNode(id, { name }),
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
  latencyResults: (id: string, from?: number, to?: number) => {
    const params = new URLSearchParams()
    if (from != null) params.set("from", String(from))
    if (to != null) params.set("to", String(to))
    const qs = params.toString()
    return req<LatencyMonitorHistory>(`/api/nodes/${id}/latency-results${qs ? `?${qs}` : ""}`)
  },
  processes: (id: string) => req<Process[]>(`/api/nodes/${id}/processes`),
  services: (id: string) => req<{ services: ServiceCheck[] }>(`/api/nodes/${id}/services`),
  docker: (id: string) => req<DockerSnapshot>(`/api/nodes/${id}/docker`),
  register: (name: string) =>
    req<{ id: string; token: string; command: string }>("/api/nodes/register", {
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
  logout: () =>
    req<{ ok: boolean }>("/api/auth/logout", {
      method: "POST",
    }),
  createAgentUpdateJob: (nodeIDs: string[], targetVersion: string, downloadURL: string, sha256: string, signature: string) =>
    req<UpdateJobResponse>("/api/agent-updates", {
      method: "POST",
      body: JSON.stringify({
        node_ids: nodeIDs,
        target_version: targetVersion,
        download_url: downloadURL,
        sha256: sha256,
        signature: signature,
      }),
    }),
  getAgentUpdateJob: (id: string) => req<UpdateJobResponse>(`/api/agent-updates/${id}`),
  metricsRetention: () => req<MetricsRetentionSettings>("/api/settings/metrics-retention"),
  updateMetricsRetention: (retentionDays: number) =>
    req<MetricsRetentionSettings>("/api/settings/metrics-retention", {
      method: "PUT",
      body: JSON.stringify({ retention_days: retentionDays }),
    }),
  latencyMonitors: () => req<{ monitors: LatencyMonitor[] }>("/api/settings/latency-monitors"),
  createLatencyMonitor: (payload: LatencyMonitorUpsertPayload) =>
    req<LatencyMonitor>("/api/settings/latency-monitors", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  updateLatencyMonitor: (id: string, payload: LatencyMonitorUpsertPayload) =>
    req<LatencyMonitor>(`/api/settings/latency-monitors/${id}`, {
      method: "PUT",
      body: JSON.stringify(payload),
    }),
  deleteLatencyMonitor: (id: string) =>
    req<{ ok: boolean }>(`/api/settings/latency-monitors/${id}`, {
      method: "DELETE",
    }),
  dashboardSettings: () => req<DashboardSettings>("/api/settings/dashboard"),
  updateDashboardSettings: (settings: DashboardSettings) =>
    req<DashboardSettings>("/api/settings/dashboard", {
      method: "PUT",
      body: JSON.stringify(settings),
    }),
  publicURLSettings: () => req<PublicURLSettings>("/api/settings/public-url"),
  updatePublicURLSettings: (settings: PublicURLSettings) =>
    req<PublicURLSettings>("/api/settings/public-url", {
      method: "PUT",
      body: JSON.stringify(settings),
    }),
  notificationSettings: () => req<NotificationSettings>("/api/settings/notifications"),
  updateNotificationSettings: (settings: Omit<NotificationSettings, "telegram_bot_token_set">) =>
    req<NotificationSettings>("/api/settings/notifications", {
      method: "PUT",
      body: JSON.stringify(settings),
    }),
  sendTestNotification: (payload: TestNotificationRequest) =>
    req<TestNotificationResponse>("/api/settings/notifications/test", {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  frontendSkins: () => req<FrontendSkinsResponse>("/api/frontend-skins"),
  installFrontendSkinArchive: (name: string, data: string) =>
    req<FrontendSkinInstallResponse>("/api/frontend-skins/install", {
      method: "POST",
      body: JSON.stringify({ source: "archive", name, data }),
    }),
  installFrontendSkinFromGitHub: (url: string) =>
    req<FrontendSkinInstallResponse>("/api/frontend-skins/install", {
      method: "POST",
      body: JSON.stringify({ source: "github", url }),
    }),
  selectFrontendSkin: (id: string) =>
    req<FrontendSkinSelectResponse>("/api/frontend-skins/select", {
      method: "POST",
      body: JSON.stringify({ id }),
    }),
  deleteFrontendSkin: (id: string) =>
    req<{ ok: boolean; active_skin_id: string }>(`/api/frontend-skins/${encodeURIComponent(id)}`, {
      method: "DELETE",
    }),
  versionMeta: () => req<VersionMeta>("/api/meta/version"),
  dispatcherRuntimeStats: () => req<DispatcherRuntimeStats>("/api/meta/dispatcher"),
  availabilityReport: (from: number, to: number, tag?: string) => {
    const params = new URLSearchParams()
    params.set("from", String(from))
    params.set("to", String(to))
    if (tag) params.set("tag", tag)
    return req<AvailabilityReport>(`/api/reports/availability?${params.toString()}`)
  },
  agentRelease: (os: string, arch: string) => req<AgentReleaseManifest>(`/api/agent-release?os=${encodeURIComponent(os)}&arch=${encodeURIComponent(arch)}`),
}
