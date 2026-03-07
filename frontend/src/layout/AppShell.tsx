import { useEffect, useState } from "react"
import { Outlet, useLocation, useNavigate } from "react-router-dom"
import { BarChart3, LogIn, Moon, RefreshCw, Settings2, Sun } from "lucide-react"
import { api, type AccessMode } from "../lib/api"
import { Button } from "../components/ui/button"
import { RouteContainer } from "./RouteContainer"
import { useThemeMode } from "../theme/mode"
import { useLanguage } from "../i18n/language"

function ShellLoadingState() {
  return (
    <div className="panel-card rounded-2xl border border-slate-200 px-4 py-8 text-center text-sm text-slate-500 dark:border-slate-700 dark:text-slate-400">
      Loading...
    </div>
  )
}

export function AppShell() {
  const navigate = useNavigate()
  const location = useLocation()
  const { mode, toggleMode } = useThemeMode()
  const { messages, labelForLanguageToggle, toggleLanguage } = useLanguage()
  const [refreshNonce, setRefreshNonce] = useState(0)
  const [accessMode, setAccessMode] = useState<AccessMode | null>(null)
  const showBack = location.pathname !== "/"
  const onSettingsPage = location.pathname.startsWith("/settings")
  const onRootPage = location.pathname === "/"
  const pageTitle = location.pathname.startsWith("/nodes/")
    ? messages.shell.pageTitles.nodeDetail
    : onSettingsPage
      ? messages.shell.pageTitles.settings
      : onRootPage
        ? messages.shell.pageTitles.dashboard
        : messages.shell.pageTitles.notFound

  useEffect(() => {
    let active = true

    void api.session()
      .then((session) => {
        if (!active) return
        setAccessMode(session.role === "guest" ? "guest" : "admin")
      })
      .catch(() => {
        if (!active) return
        setAccessMode("guest")
      })

    return () => {
      active = false
    }
  }, [])

  return (
    <div className="min-h-screen app-surface-bg text-slate-900 dark:text-slate-100">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-white focus:px-3 focus:py-2 focus:text-xs focus:text-slate-900 dark:focus:bg-slate-900 dark:focus:text-slate-100"
      >
        {messages.shell.skipToMainContent}
      </a>
      <header className="sticky top-0 z-20 border-b border-slate-200/80 bg-white/90 backdrop-blur dark:border-slate-800/80 dark:bg-slate-950/80">
        <div className="mx-auto flex h-16 max-w-[1440px] items-center justify-between gap-2 px-3 sm:gap-4 sm:px-4 md:px-6 lg:px-8">
          <div className="flex min-w-0 items-center gap-2 sm:gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 dark:border-slate-700 dark:bg-slate-900 sm:h-9 sm:w-9">
              <BarChart3 className="h-4 w-4 text-slate-700 dark:text-slate-200" aria-hidden />
            </div>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{messages.common.brand}</p>
              <p className="hidden truncate text-xs text-slate-500 dark:text-slate-400 sm:block">{pageTitle}</p>
            </div>
            {accessMode === "guest" && (
              <a
                href="/login"
                aria-label={messages.shell.actions.returnToLogin}
                title={messages.shell.actions.returnToLogin}
                className="hidden items-center gap-1 rounded-full border border-amber-300/80 bg-amber-50 px-2.5 py-1 text-[11px] font-medium text-amber-800 transition-colors hover:bg-amber-100 dark:border-amber-900/70 dark:bg-amber-950/40 dark:text-amber-200 dark:hover:bg-amber-950/60 sm:inline-flex"
              >
                <LogIn className="h-3.5 w-3.5" aria-hidden />
                <span>{messages.shell.actions.guestMode}</span>
              </a>
            )}
          </div>

          <div className="flex items-center gap-1 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={labelForLanguageToggle}
              onClick={toggleLanguage}
              className="h-11 min-w-[4.75rem] border-slate-300 bg-white px-3 text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800 sm:h-9"
            >
              {labelForLanguageToggle}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={messages.shell.actions.refreshData}
              onClick={() => setRefreshNonce((value) => value + 1)}
              className="h-11 w-11 border-slate-300 bg-white px-0 text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800 sm:h-9 sm:w-9"
            >
              <RefreshCw className="h-4 w-4" aria-hidden />
            </Button>
            <Button
              type="button"
              variant="outline"
              size="icon"
              aria-label={messages.shell.actions.toggleDarkMode}
              onClick={toggleMode}
              className="h-11 w-11 border-slate-300 bg-white text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800 sm:h-9 sm:w-9"
            >
              {mode === "dark" ? <Sun className="h-4 w-4" aria-hidden /> : <Moon className="h-4 w-4" aria-hidden />}
            </Button>
            {accessMode !== "guest" && (
              <Button
                type="button"
                variant={onSettingsPage ? "default" : "outline"}
                size="sm"
                onClick={() => navigate("/settings")}
                aria-label={messages.shell.actions.openSettings}
                aria-current={onSettingsPage ? "page" : undefined}
                className={
                  onSettingsPage
                    ? "h-11 w-11 bg-primary px-0 text-primary-foreground hover:bg-primary/90 sm:h-9 sm:w-9"
                    : "h-11 w-11 border-slate-300 bg-white px-0 text-slate-700 hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-200 dark:hover:bg-slate-800 sm:h-9 sm:w-9"
                }
              >
                <Settings2 className="h-4 w-4" aria-hidden />
              </Button>
            )}
          </div>
        </div>
      </header>

      <main id="main-content" className="mx-auto max-w-[1440px] px-4 py-6 md:px-6 md:py-8 lg:px-8">
        <RouteContainer showBack={showBack} onBack={() => navigate("/")}>
          {accessMode === null ? <ShellLoadingState /> : <Outlet context={{ refreshNonce, accessMode }} />}
        </RouteContainer>
      </main>
    </div>
  )
}
