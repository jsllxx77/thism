import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NotificationsCard } from "./NotificationsCard"

const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()
const sendTestNotificationMock = vi.fn()
const nodesMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    sendTestNotification: (...args: unknown[]) => sendTestNotificationMock(...args),
    nodes: (...args: unknown[]) => nodesMock(...args),
  },
}))

describe("notifications card", () => {
  beforeEach(() => {
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    sendTestNotificationMock.mockReset()
    nodesMock.mockReset()
    notificationSettingsMock.mockResolvedValue({
      enabled: true,
      channel: "telegram",
      telegram_bot_token_set: true,
      telegram_targets: [{ name: "Ops", chat_id: "-100123", topic_id: 99 }],
      enabled_node_ids: ["node-1"],
      node_scope_mode: "include",
      node_scope_node_ids: ["node-1"],
      cpu_warning_percent: 80,
      cpu_critical_percent: 90,
      mem_warning_percent: 81,
      mem_critical_percent: 91,
      disk_warning_percent: 82,
      disk_critical_percent: 92,
      cooldown_minutes: 15,
      notify_node_offline: true,
      notify_node_online: true,
      node_offline_grace_minutes: 2,
    })
    updateNotificationSettingsMock.mockResolvedValue({
      enabled: true,
      channel: "telegram",
      telegram_bot_token_set: true,
      telegram_targets: [{ name: "Ops", chat_id: "-100123", topic_id: 99 }],
      enabled_node_ids: ["node-1", "node-2"],
      node_scope_mode: "include",
      node_scope_node_ids: ["node-1", "node-2"],
      cpu_warning_percent: 80,
      cpu_critical_percent: 90,
      mem_warning_percent: 81,
      mem_critical_percent: 91,
      disk_warning_percent: 82,
      disk_critical_percent: 92,
      cooldown_minutes: 15,
      notify_node_offline: true,
      notify_node_online: true,
      node_offline_grace_minutes: 2,
    })
    sendTestNotificationMock.mockResolvedValue({ ok: true })
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "node-1", name: "Alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
        { id: "node-2", name: "Beta", ip: "2.2.2.2", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: false },
      ],
    })
  })

  it("loads notification settings and saves telegram targets", async () => {
    const user = userEvent.setup()
    render(<NotificationsCard />)

    expect(await screen.findByRole("heading", { name: "Notifications", level: 3 })).toBeInTheDocument()
    expect(screen.getByDisplayValue("-100123")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Only selected nodes" })).toBeInTheDocument()
    expect(screen.getByLabelText("Enable notification for Alpha")).toBeChecked()
    expect(screen.getByLabelText("Enable notification for Beta")).not.toBeChecked()

    await user.type(screen.getByLabelText("Search nodes by name or ID"), "Beta")
    expect(screen.getByLabelText("Enable notification for Beta")).toBeInTheDocument()
    await user.click(screen.getByLabelText("Enable notification for Beta"))
    await user.clear(screen.getByLabelText("Telegram bot token"))
    await user.type(screen.getByLabelText("Telegram bot token"), "123:abc")
    await user.click(screen.getByRole("button", { name: "Save notifications" }))

    await waitFor(() => expect(updateNotificationSettingsMock).toHaveBeenCalled())
    expect(updateNotificationSettingsMock.mock.calls[0][0]).toMatchObject({
      channel: "telegram",
      telegram_bot_token: "123:abc",
      telegram_targets: [{ chat_id: "-100123", topic_id: 99, name: "Ops" }],
      enabled_node_ids: ["node-1", "node-2"],
      node_scope_mode: "include",
      node_scope_node_ids: ["node-1", "node-2"],
      notify_node_offline: true,
      notify_node_online: true,
      node_offline_grace_minutes: 2,
    })
    expect(await screen.findByText("Notification settings updated.")).toBeInTheDocument()
  })

  it("sends a test notification to the first telegram target", async () => {
    const user = userEvent.setup()
    render(<NotificationsCard />)

    expect(await screen.findByRole("button", { name: "Send test notification" })).toBeInTheDocument()
    await user.click(screen.getByRole("button", { name: "Send test notification" }))

    await waitFor(() => expect(sendTestNotificationMock).toHaveBeenCalled())
    expect(sendTestNotificationMock).toHaveBeenCalledWith({
      telegram_bot_token: "",
      target: { name: "Ops", chat_id: "-100123", topic_id: 99 },
    })
    expect(await screen.findByText("Test notification sent.")).toBeInTheDocument()
  })
})
