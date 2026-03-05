import { ConfigProvider } from "antd"
import type { ThemeConfig } from "antd"
import { appFontStack } from "./fonts"

type Props = {
  children: React.ReactNode
}

const appTheme: ThemeConfig = {
  token: {
    colorPrimary: "#34d399",
    colorBgBase: "#05070f",
    colorTextBase: "#f5f7ff",
    colorBorder: "rgba(255, 255, 255, 0.12)",
    borderRadius: 14,
    fontFamily: appFontStack,
  },
}

export function AppThemeProvider({ children }: Props) {
  return <ConfigProvider theme={appTheme}>{children}</ConfigProvider>
}
