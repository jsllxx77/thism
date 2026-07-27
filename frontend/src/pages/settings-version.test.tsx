import { beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { Settings } from "./Settings"

const nodesMock = vi.fn()
const changePasswordMock = vi.fn()
const agentReleaseMock = vi.fn()
const metricsRetentionMock = vi.fn()
const updateMetricsRetentionMock = vi.fn()
const geoIPSettingsMock = vi.fn()
const updateGeoIPSettingsMock = vi.fn()
const updateGeoIPDatabaseMock = vi.fn()
const versionMetaMock = vi.fn()

vi.mock("../lib/api", () => ({
  api: {
    nodes: (...args: unknown[]) => nodesMock(...args),
    changePassword: (...args: unknown[]) => changePasswordMock(...args),
    agentRelease: (...args: unknown[]) => agentReleaseMock(...args),
    metricsRetention: (...args: unknown[]) => metricsRetentionMock(...args),
    updateMetricsRetention: (...args: unknown[]) => updateMetricsRetentionMock(...args),
    geoIPSettings: (...args: unknown[]) => geoIPSettingsMock(...args),
    updateGeoIPSettings: (...args: unknown[]) => updateGeoIPSettingsMock(...args),
    updateGeoIPDatabase: (...args: unknown[]) => updateGeoIPDatabaseMock(...args),
    versionMeta: (...args: unknown[]) => versionMetaMock(...args),
  },
}))

function renderSettings(path = "/settings?section=security") {
  window.history.replaceState({}, "", path)
  return render(<Settings />)
}

describe("settings version metadata", () => {
  beforeEach(() => {
    nodesMock.mockReset()
    changePasswordMock.mockReset()
    agentReleaseMock.mockReset()
    metricsRetentionMock.mockReset()
    updateMetricsRetentionMock.mockReset()
    geoIPSettingsMock.mockReset()
    updateGeoIPSettingsMock.mockReset()
    updateGeoIPDatabaseMock.mockReset()
    versionMetaMock.mockReset()

    nodesMock.mockResolvedValue({ nodes: [] })
    metricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
    updateMetricsRetentionMock.mockResolvedValue({ retention_days: 30, options: [30, 90, 180, 365] })
    geoIPSettingsMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    updateGeoIPSettingsMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    updateGeoIPDatabaseMock.mockResolvedValue({ provider: "maxmind", ip2location_token_set: false, maxmind_license_key_set: false, enabled: true, database_exists: true, supported_providers: ["maxmind", "ip2location"] })
    agentReleaseMock.mockResolvedValue({ target_version: "v1.2.3", download_url: "https://example.com/amd64", sha256: "sha", check_interval_seconds: 1800 })
    versionMetaMock.mockResolvedValue({ version: "v1.2.3", commit: "abc1234", build_time: "2026-03-18T04:00:00Z" })
  })

  it("renders server version metadata in the security section", async () => {
    renderSettings()

    expect(await screen.findByText("Server version")).toBeInTheDocument()
    expect(screen.getByText("abc1234")).toBeInTheDocument()
    expect(screen.getByText("2026-03-18T04:00:00Z")).toBeInTheDocument()
    expect(versionMetaMock).toHaveBeenCalledTimes(1)
  })
})
