import { Outlet, useLocation, useNavigate } from "react-router-dom"
import { Layout, Typography } from "antd"
import { DashboardOutlined, SettingOutlined } from "@ant-design/icons"
import { RouteContainer } from "./RouteContainer"
import { PrimaryNav, type NavItem } from "./PrimaryNav"
import { MobileNav } from "./MobileNav"

const { Sider, Header, Content } = Layout

const navItems: NavItem[] = [
  { key: "dashboard", label: "Dashboard", path: "/", icon: <DashboardOutlined aria-hidden /> },
  { key: "settings", label: "Settings", path: "/settings", icon: <SettingOutlined aria-hidden /> },
]

export function AppShell() {
  const navigate = useNavigate()
  const location = useLocation()
  const showBack = location.pathname !== "/"
  const activeKey = location.pathname.startsWith("/settings") ? "settings" : "dashboard"

  return (
    <div className="min-h-screen app-gradient-bg text-white">
      <Layout className="!bg-transparent min-h-screen">
        <Sider
          width={248}
          breakpoint="md"
          collapsedWidth={0}
          className="!bg-transparent !hidden md:!block px-4 py-5"
        >
          <div className="glass-panel rounded-2xl p-4 h-full">
            <Typography.Title level={4} className="!text-white !mb-5 !text-lg !font-semibold">
              ThisM
            </Typography.Title>
            <PrimaryNav items={navItems} activeKey={activeKey} onNavigate={navigate} />
          </div>
        </Sider>
        <Layout className="!bg-transparent">
          <Header className="!bg-[#0a1020]/70 backdrop-blur border-b border-white/15 px-4 md:px-6 flex items-center justify-between gap-3 sticky top-0 z-20">
            <div className="flex items-center gap-2">
              <span className="inline-flex h-2 w-2 rounded-full bg-emerald-400 animate-pulse" />
              <span className="text-xs text-white/70 tracking-wide">Live</span>
            </div>
            <MobileNav items={navItems} activeKey={activeKey} onNavigate={navigate} />
          </Header>
          <Content className="p-4 md:p-6">
            <RouteContainer showBack={showBack} onBack={() => navigate(-1)}>
              <Outlet />
            </RouteContainer>
          </Content>
        </Layout>
      </Layout>
    </div>
  )
}
