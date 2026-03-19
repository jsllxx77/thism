import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NotificationsCard } from "./NotificationsCard"

const notificationSettingsMock = vi.fn()
const updateNotificationSettingsMock = vi.fn()
const sendTestNotificationMock = vi.fn()

vi.mock("../../lib/api", () => ({
  api: {
    notificationSettings: (...args: unknown[]) => notificationSettingsMock(...args),
    updateNotificationSettings: (...args: unknown[]) => updateNotificationSettingsMock(...args),
    sendTestNotification: (...args: unknown[]) => sendTestNotificationMock(...args),
  },
}))

describe("notifications card", () => {
  beforeEach(() => {
    notificationSettingsMock.mockReset()
    updateNotificationSettingsMock.mockReset()
    sendTestNotificationMock.mockReset()
    notificationSettingsMock.mockResolvedValue({
      enabled: true,
      channel: "telegram",
      telegram_bot_token_set: true,
      telegram_targets: [{ name: "Ops", chat_id: "-100123", topic_id: 99 }],
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
  })

  it("loads notification settings and saves telegram targets", async () => {
    const user = userEvent.setup()
    render(<NotificationsCard />)

    expect(await screen.findByRole("heading", { name: "Notifications", level: 3 })).toBeInTheDocument()
    expect(screen.getByDisplayValue("-100123")).toBeInTheDocument()

    await user.clear(screen.getByLabelText("Telegram bot token"))
    await user.type(screen.getByLabelText("Telegram bot token"), "123:abc")
    await user.click(screen.getByRole("button", { name: "Save notifications" }))

    await waitFor(() => expect(updateNotificationSettingsMock).toHaveBeenCalled())
    expect(updateNotificationSettingsMock.mock.calls[0][0]).toMatchObject({
      channel: "telegram",
      telegram_bot_token: "123:abc",
      telegram_targets: [{ chat_id: "-100123", topic_id: 99, name: "Ops" }],
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
