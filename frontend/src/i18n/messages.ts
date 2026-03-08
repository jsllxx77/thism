import { messagesEn } from "./messages.en"

export type AppLanguage = "en" | "zh-CN"

type DeepMessageShape<T> = T extends string
  ? string
  : T extends Record<string, unknown>
    ? { [K in keyof T]: DeepMessageShape<T[K]> }
    : T

export type AppMessages = DeepMessageShape<typeof messagesEn>

const messageCache: Partial<Record<AppLanguage, AppMessages>> = {
  en: messagesEn,
}

export async function loadLanguageMessages(language: AppLanguage): Promise<AppMessages> {
  if (messageCache[language]) {
    return messageCache[language]
  }

  if (language === "zh-CN") {
    const module = await import("./messages.zh-CN")
    messageCache[language] = module.messagesZhCN as AppMessages
    return messageCache[language]
  }

  return messagesEn
}

export function getCachedLanguageMessages(language: AppLanguage): AppMessages {
  return messageCache[language] ?? messagesEn
}

const legacyMessageKeyMap: Record<string, string> = {
  "ThisM Console": "common.brand",
  "Online": "common.online",
  "Offline": "common.offline",
  "—": "common.unavailable",
  "Retry": "common.retry",
  "Close": "common.close",
  "Cancel": "common.cancel",
  "Save": "common.save",
  "Done": "common.done",
  "Copy": "common.copy",
  "Copied": "common.copied",
  "Delete": "common.delete",
  "Rename": "common.rename",
  "Get Script": "common.getScript",
  "Install Command": "common.installCommand",
  "Token": "common.token",
  "Current password": "common.currentPassword",
  "New password": "common.newPassword",
  "Confirm new password": "common.confirmPassword",
  "Node Name": "common.nodeName",
  "Security": "common.security",
  "Status": "common.status",
  "IP": "common.ip",
  "Name": "common.name",
  "Memory": "common.memory",
  "Disk": "common.disk",
  "Platform": "common.platform",
  "Processes": "common.processes",
  "Services": "common.services",
  "Settings": "common.settings",
  "Add Node": "common.addNode",
  "Delete Node": "common.deleteNode",
  "Last seen": "common.lastSeen",
  "Unknown node": "common.unknownNode",
  "Skip to main content": "shell.skipToMainContent",
  "Dashboard": "shell.pageTitles.dashboard",
  "Node Detail": "shell.pageTitles.nodeDetail",
  "Not Found": "shell.pageTitles.notFound",
  "Refresh data": "shell.actions.refreshData",
  "Toggle dark mode": "shell.actions.toggleDarkMode",
  "Open settings": "shell.actions.openSettings",
  "Back to Dashboard": "shell.actions.backToDashboard",
  "Back to dashboard": "notFound.back",
  "Open navigation": "navigation.openNavigation",
  "Navigate": "navigation.navigate",
  "Primary Navigation": "navigation.primaryNavigation",
  "Resource Grid": "dashboard.title",
  "Real-time node inventory with lightweight filters and quick detail access.": "dashboard.subtitle",
  "Loading node inventory...": "dashboard.loadingInventory",
  "We couldn't load nodes. Please try again.": "dashboard.loadError",
  "No nodes registered yet": "dashboard.noNodesRegistered",
  "Register a node using the API, then start thisM-agent": "dashboard.registrationHint",
  "No nodes match current filters": "dashboard.noNodesMatch",
  "Try adjusting search keywords or status conditions.": "dashboard.adjustFilters",
  "Node Inventory": "dashboard.inventoryTitle",
  "Open node {name}": "dashboard.openNodeAria",
  "Status filter": "dashboard.filters.status",
  "Search nodes": "dashboard.filters.search",
  "Search by node name": "dashboard.filters.searchPlaceholder",
  "Reset filters": "dashboard.filters.reset",
  "All": "dashboard.filters.all",
  "Cards View": "dashboard.viewMode.cards",
  "Table View": "dashboard.viewMode.table",
  "Total Nodes": "dashboard.stats.totalNodes",
  "Online Nodes": "dashboard.stats.onlineNodes",
  "{online} / {total} online": "dashboard.stats.onlineValue",
  "Avg CPU": "dashboard.stats.avgCpu",
  "Avg Memory": "dashboard.stats.avgMemory",
  "Offline Alerts": "dashboard.stats.offlineAlerts",
  "CPU": "dashboard.nodeCard.cpu",
  "MEM": "dashboard.nodeCard.memory",
  "Control Plane": "settingsPage.eyebrow",
  "Manage node enrollment, registry actions, and administrator credentials from the same engineering-passport shell used across the dashboard.": "settingsPage.subtitle",
  "Node registry": "settingsPage.chipRegistry",
  "Provisioning": "settingsPage.chipProvisioning",
  "Node Management": "settingsPage.nodeManagement",
  "Loading node registry...": "settingsPage.loadingRegistry",
  "We couldn't load settings data. Please try again.": "settingsPage.loadError",
  "Change Password": "settingsPage.changePasswordTitle",
  "Node Registry": "settingsTable.title",
  "Update the administrator password used on the login page.": "settingsPage.changePasswordSubtitle",
  "Updating...": "settingsPage.updatingPassword",
  "Update Password": "settingsPage.updatePassword",
  "All password fields are required.": "settingsPage.allPasswordFieldsRequired",
  "New password and confirmation do not match.": "settingsPage.passwordMismatch",
  "New password must be different from the current password.": "settingsPage.passwordMustDiffer",
  "Password updated successfully.": "settingsPage.passwordUpdated",
  "Failed to update password.": "settingsPage.passwordUpdateFailed",
  "Node Provisioning": "addNodeModal.title",
  "Node name is required": "addNodeModal.nodeNameRequired",
  "e.g. web-server-01": "addNodeModal.placeholder",
  "Generate command": "addNodeModal.generateCommand",
  "Run this on the target machine as root:": "addNodeModal.runAsRoot",
  "Copy command": "addNodeModal.copyCommandAria",
  "Failed to register node": "addNodeModal.failedToRegister",
  "Settings status filter": "settingsTable.statusFilterAria",
  "Manage node lifecycle and access commands.": "settingsTable.description",
  "No nodes match the active status filter.": "settingsTable.emptyFilter",
  "Created": "common.created",
  "Actions": "common.actions",
  "OS / Arch": "common.osArch",
  "Rename Node": "settingsTable.renameNodeTitle",
  "Update the display name for this node.": "settingsTable.renameNodeDescription",
  "Enter a node name": "settingsTable.renamePlaceholder",
  "Generating command...": "settingsTable.generatingCommand",
  "This will remove node \"{name}\" and its historical metrics.": "settingsTable.deleteNodeDescription",
  "We couldn't load node details. Please try again.": "nodeDetail.loadError",
  "Loading node details...": "nodeDetail.loading",
  "Node Overview": "nodeDetail.heroEyebrow",
  "Operational identity and platform signature for this node.": "nodeDetail.heroDescription",
  "Hardware Passport": "nodeDetail.hardwarePassportEyebrow",
  "A static hardware fingerprint for this node, surfaced alongside live telemetry.": "nodeDetail.hardwarePassportDescription",
  "Asset profile": "nodeDetail.assetProfile",
  "Cores / Threads": "nodeDetail.coresThreads",
  "Virtualization": "nodeDetail.virtualization",
  "CPU Usage": "nodeDetail.cpuUsage",
  "Memory Usage": "nodeDetail.memoryUsage",
  "Network Traffic": "nodeDetail.networkTraffic",
  "Disk Usage": "nodeDetail.diskUsage",
  "Inbound Traffic": "nodeDetail.inboundTraffic",
  "Outbound Traffic": "nodeDetail.outboundTraffic",
  "Inbound Speed": "nodeDetail.inboundSpeed",
  "Outbound Speed": "nodeDetail.outboundSpeed",
  "Inbound Total": "nodeDetail.inboundTotal",
  "Outbound Total": "nodeDetail.outboundTotal",
  "Network summary": "nodeDetail.networkSummaryAria",
  "Process Snapshot": "nodeDetail.processSnapshot",
  "CPU%": "nodeDetail.processCpu",
  "Service Health": "nodeDetail.serviceHealth",
  "Page not found": "notFound.title",
  "The page you requested does not exist in this console.": "notFound.description",
}

function interpolate(template: string, params?: Record<string, string | number | undefined>) {
  if (!params) return template
  return template.replace(/\{(\w+)\}/g, (_, key: string) => String(params[key] ?? `{${key}}`))
}

function resolveValue(messages: AppMessages, key: string): string | undefined {
  const value = key.split(".").reduce<unknown>((current, segment) => {
    if (!current || typeof current !== "object") return undefined
    return (current as Record<string, unknown>)[segment]
  }, messages)
  return typeof value === "string" ? value : undefined
}

export function translateMessage(language: AppLanguage, key: string, params?: Record<string, string | number | undefined>) {
  const catalog = getCachedLanguageMessages(language)
  const resolvedKey = resolveValue(catalog, key) ? key : (legacyMessageKeyMap[key] ?? key)
  return interpolate(resolveValue(catalog, resolvedKey) ?? key, params)
}

export function getLanguageToggleLabel(language: AppLanguage) {
  return language === "en" ? "中文" : "English"
}

const apiErrorKeyMap: Record<string, string> = {
  "Action failed. Please try again.": "errors.actionFailed",
  "Clipboard access was denied. Please copy the command manually.": "errors.clipboardDenied",
  "invalid current password": "errors.invalidCurrentPassword",
  "current_password and new_password are required": "errors.currentPasswordAndNewPasswordRequired",
  "new password must be different": "errors.newPasswordMustBeDifferent",
  "password login is not configured": "errors.passwordLoginNotConfigured",
  "invalid request body": "errors.invalidRequestBody",
  "invalid credentials": "errors.invalidCredentials",
  "name is required": "errors.nameRequired",
  "token generation failed": "errors.tokenGenerationFailed",
  "node not found": "errors.nodeNotFound",
  unauthorized: "errors.unauthorized",
}

export function translateApiErrorMessage(language: AppLanguage, message: string) {
  const key = apiErrorKeyMap[message]
  return key ? translateMessage(language, key) : message
}

const serviceStatusKeyMap: Record<string, string> = {
  running: "serviceStatus.running",
  failed: "serviceStatus.failed",
  dead: "serviceStatus.dead",
}

export function translateServiceStatus(language: AppLanguage, status: string) {
  const key = serviceStatusKeyMap[status]
  return key ? translateMessage(language, key) : status
}

export function formatRelativeLastSeen(language: AppLanguage, lastSeen: number, nowMs: number) {
  const catalog = getCachedLanguageMessages(language)
  if (lastSeen <= 0) return catalog.common.unavailable

  const diffSeconds = Math.max(0, Math.floor(nowMs / 1000) - lastSeen)
  if (diffSeconds <= 4) return catalog.dashboard.nodeCard.justNow
  if (diffSeconds < 60) return interpolate(catalog.dashboard.nodeCard.secondsAgo, { value: diffSeconds })

  const diffMinutes = Math.floor(diffSeconds / 60)
  if (diffMinutes < 60) return interpolate(catalog.dashboard.nodeCard.minutesAgo, { value: diffMinutes })

  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return interpolate(catalog.dashboard.nodeCard.hoursAgo, { value: diffHours })

  const diffDays = Math.floor(diffHours / 24)
  return interpolate(catalog.dashboard.nodeCard.daysAgo, { value: diffDays })
}

export function formatUptimeDuration(language: AppLanguage, uptimeSeconds?: number | null) {
  const catalog = getCachedLanguageMessages(language)
  if (!uptimeSeconds || uptimeSeconds <= 0) {
    return catalog.common.unavailable
  }
  if (uptimeSeconds < 60) {
    return language === "zh-CN" ? "不到1分钟" : "<1m"
  }

  const totalMinutes = Math.floor(uptimeSeconds / 60)
  const days = Math.floor(totalMinutes / (24 * 60))
  const hours = Math.floor((totalMinutes % (24 * 60)) / 60)
  const minutes = totalMinutes % 60

  if (language === "zh-CN") {
    const parts: string[] = []
    if (days > 0) parts.push(`${days}天`)
    if (hours > 0) parts.push(`${hours}小时`)
    if (minutes > 0 || parts.length === 0) parts.push(`${minutes}分`)
    return parts.join(" ")
  }

  const parts: string[] = []
  if (days > 0) parts.push(`${days}d`)
  if (hours > 0) parts.push(`${hours}h`)
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes}m`)
  return parts.join(" ")
}
