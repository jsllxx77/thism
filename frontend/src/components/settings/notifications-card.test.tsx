import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NotificationsCard } from "./NotificationsCard"

const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()
const sendTestNotificationMock = vi.fn()
const nodesMock = vi.fn()
const dispatcherRuntimeStatsMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    sendTestNotification: (...args: unknown[]) => sendTestNotificationMock(...args),
    nodes: (...args: unknown[]) => nodesMock(...args),
    dispatcherRuntimeStats: (...args: unknown[]) => dispatcherRuntimeStatsMock(...args),
  },
}))

describe("notifications card", () => {
  beforeEach(() => {
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    sendTestNotificationMock.mockReset()
    nodesMock.mockReset()
    dispatcherRuntimeStatsMock.mockReset()
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
      dispatcher_queue_capacity: 32,
      notify_dispatcher_drops: true,
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
      dispatcher_queue_capacity: 64,
      notify_dispatcher_drops: false,
    })
    sendTestNotificationMock.mockResolvedValue({ ok: true })
    nodesMock.mockResolvedValue({
      nodes: [
        { id: "node-1", name: "Alpha", ip: "1.1.1.1", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: true },
        { id: "node-2", name: "Beta", ip: "2.2.2.2", os: "linux", arch: "amd64", created_at: 1, last_seen: 1, online: false },
      ],
    })
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 0,
      high_watermark: 0,
      enqueued: 0,
      processed: 0,
      dropped: 0,
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
    expect(screen.getByLabelText("Dispatcher queue capacity")).toHaveValue(32)
    expect(screen.getByRole("switch", { name: "Notify when dispatcher drops queued alerts" })).toHaveAttribute("aria-checked", "true")

    await user.type(screen.getByLabelText("Search nodes by name or ID"), "Beta")
    expect(screen.getByLabelText("Enable notification for Beta")).toBeInTheDocument()
    await user.click(screen.getByLabelText("Enable notification for Beta"))
    await user.clear(screen.getByLabelText("Dispatcher queue capacity"))
    await user.type(screen.getByLabelText("Dispatcher queue capacity"), "64")
    await user.click(screen.getByRole("switch", { name: "Notify when dispatcher drops queued alerts" }))
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
      dispatcher_queue_capacity: 64,
      notify_dispatcher_drops: false,
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

  it("shows a normal dispatcher health summary when the queue is clear", async () => {
    render(<NotificationsCard />)

    expect(await screen.findByText("Alert delivery status")).toBeInTheDocument()
    expect(screen.getByText("Normal")).toBeInTheDocument()
  })

  it("shows a backlogged dispatcher health summary when alerts are queued", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 3,
      high_watermark: 4,
      enqueued: 9,
      processed: 6,
      dropped: 0,
    })

    render(<NotificationsCard />)

    expect(await screen.findByText("Alert delivery status")).toBeInTheDocument()
    expect(screen.getByText("Backlogged")).toBeInTheDocument()
  })

  it("shows drops detected when dispatcher jobs were dropped", async () => {
    dispatcherRuntimeStatsMock.mockResolvedValue({
      active_dispatchers: 1,
      total_capacity: 256,
      queue_depth: 8,
      high_watermark: 8,
      enqueued: 15,
      processed: 7,
      dropped: 2,
    })

    render(<NotificationsCard />)

    expect(await screen.findByText("Alert delivery status")).toBeInTheDocument()
    expect(screen.getByText("Drops detected")).toBeInTheDocument()
  })

  it("renders offline grace input in a separate block from the toggle buttons", async () => {
    render(<NotificationsCard />)

    const graceInput = await screen.findByLabelText("Offline grace (minutes)")
    const toggleGrid = screen.getByTestId("notification-toggle-grid")
    const graceField = screen.getByTestId("offline-grace-field")

    expect(toggleGrid).not.toContainElement(graceInput)
    expect(graceField).toContainElement(graceInput)
  })
})
