import { createContext, useContext, useEffect, useMemo, useState } from "react"
import {
  formatRelativeLastSeen,
  formatUptimeDuration,
  getCachedLanguageMessages,
  getLanguageToggleLabel,
  loadLanguageMessages,
  translateApiErrorMessage,
  translateMessage,
  translateServiceStatus,
  type AppLanguage,
  type AppMessages,
} from "./messages"

const STORAGE_KEY = "thism-language"
const COOKIE_NAME = "thism-lang"
const ONE_YEAR_SECONDS = 60 * 60 * 24 * 365

type LanguageContextValue = {
  language: AppLanguage
  messages: AppMessages
  setLanguage: (language: AppLanguage) => void
  toggleLanguage: () => void
  labelForLanguageToggle: string
  t: (key: string, params?: Record<string, string | number | undefined>) => string
  translateApiError: (message: string) => string
  translateError: (message: string) => string
  translateServiceStatus: (status: string) => string
  formatRelativeLastSeen: (lastSeen: number, nowMs: number) => string
  formatUptimeDuration: (uptimeSeconds?: number | null) => string
}

function normalizeLanguage(value?: string | null): AppLanguage | null {
  if (!value) return null
  const normalized = value.trim().toLowerCase()
  if (normalized === "en" || normalized.startsWith("en-")) return "en"
  if (normalized === "zh-cn" || normalized === "zh" || normalized.startsWith("zh-")) return "zh-CN"
  return null
}

function readCookie(name: string): string | null {
  if (typeof document === "undefined") return null
  const prefix = `${name}=`
  const match = document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix))
  return match ? decodeURIComponent(match.slice(prefix.length)) : null
}

export function getPreferredLanguage(): AppLanguage {
  if (typeof window !== "undefined") {
    const stored = normalizeLanguage(window.localStorage.getItem(STORAGE_KEY))
    if (stored) {
      return stored
    }
  }

  const cookie = normalizeLanguage(readCookie(COOKIE_NAME))
  if (cookie) {
    return cookie
  }

  const browserLanguage = typeof window !== "undefined" ? normalizeLanguage(window.navigator.language) : null
  return browserLanguage ?? "en"
}

function persistLanguage(language: AppLanguage) {
  if (typeof document !== "undefined") {
    document.documentElement.setAttribute("lang", language)
    document.cookie = `${COOKIE_NAME}=${encodeURIComponent(language)}; path=/; max-age=${ONE_YEAR_SECONDS}; samesite=lax`
  }

  if (typeof window !== "undefined") {
    window.localStorage.setItem(STORAGE_KEY, language)
  }
}

const defaultLanguageContext: LanguageContextValue = {
  language: "en",
  messages: getCachedLanguageMessages("en"),
  setLanguage: () => undefined,
  toggleLanguage: () => undefined,
  labelForLanguageToggle: getLanguageToggleLabel("en"),
  t: (key, params) => translateMessage("en", key, params),
  translateApiError: (message) => translateApiErrorMessage("en", message),
  translateError: (message) => translateApiErrorMessage("en", message),
  translateServiceStatus: (status) => translateServiceStatus("en", status),
  formatRelativeLastSeen: (lastSeen, nowMs) => formatRelativeLastSeen("en", lastSeen, nowMs),
  formatUptimeDuration: (uptimeSeconds) => formatUptimeDuration("en", uptimeSeconds),
}

const LanguageContext = createContext<LanguageContextValue>(defaultLanguageContext)

type Props = {
  children: React.ReactNode
}

export function LanguageProvider({ children }: Props) {
  const [language, setLanguageState] = useState<AppLanguage>(getPreferredLanguage)
  const [messages, setMessages] = useState<AppMessages>(() => getCachedLanguageMessages(getPreferredLanguage()))

  useEffect(() => {
    persistLanguage(language)
  }, [language])

  useEffect(() => {
    let cancelled = false

    void loadLanguageMessages(language).then((loadedMessages) => {
      if (!cancelled) {
        setMessages(loadedMessages)
      }
    })

    return () => {
      cancelled = true
    }
  }, [language])

  const value = useMemo<LanguageContextValue>(
    () => ({
      language,
      messages,
      setLanguage: (nextLanguage) => {
        setMessages(getCachedLanguageMessages(nextLanguage))
        setLanguageState(nextLanguage)
      },
      toggleLanguage: () => {
        const nextLanguage = language === "en" ? "zh-CN" : "en"
        setMessages(getCachedLanguageMessages(nextLanguage))
        setLanguageState(nextLanguage)
      },
      labelForLanguageToggle: getLanguageToggleLabel(language),
      t: (key, params) => translateMessage(language, key, params),
      translateApiError: (message) => translateApiErrorMessage(language, message),
      translateError: (message) => translateApiErrorMessage(language, message),
      translateServiceStatus: (status) => translateServiceStatus(language, status),
      formatRelativeLastSeen: (lastSeen, nowMs) => formatRelativeLastSeen(language, lastSeen, nowMs),
      formatUptimeDuration: (uptimeSeconds) => formatUptimeDuration(language, uptimeSeconds),
    }),
    [language, messages],
  )

  return <LanguageContext.Provider value={value}>{children}</LanguageContext.Provider>
}

export function useLanguage() {
  return useContext(LanguageContext)
}

export type { AppLanguage }
