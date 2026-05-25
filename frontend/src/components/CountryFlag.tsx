import { memo, type ReactElement } from "react"
import { countryCodeToFlagEmoji, countryCodeToFlagLabel } from "../lib/flags"

type CountryFlagProps = {
  countryCode?: string | null
  className?: string
  size?: "sm" | "md" | "lg"
}

const sizeClasses: Record<NonNullable<CountryFlagProps["size"]>, string> = {
  sm: "w-[1.05rem] text-[13px]",
  md: "w-[1.25rem] text-[16px]",
  lg: "w-[1.5rem] text-[20px]",
}

type FlagFallbackRenderer = (className: string) => ReactElement

const flagFallbacks: Partial<Record<string, FlagFallbackRenderer>> = {
  AU: (className) => (
    <svg className={className} viewBox="0 0 64 48" aria-hidden="true" focusable="false">
      <rect width="64" height="48" fill="#012169" />
      <path d="M0 0h32v24H0z" fill="#012169" />
      <path d="m0 0 32 24M32 0 0 24" stroke="#fff" strokeWidth="5" />
      <path d="m0 0 32 24M32 0 0 24" stroke="#c8102e" strokeWidth="2.8" />
      <path d="M16 0v24M0 12h32" stroke="#fff" strokeWidth="8" />
      <path d="M16 0v24M0 12h32" stroke="#c8102e" strokeWidth="4.6" />
      <path d="M16 29.5 18 36h6.4l-5.2 3.8 2 6.2-5.2-3.8-5.2 3.8 2-6.2L7.6 36H14z" fill="#fff" />
      <circle cx="46" cy="16" r="2.6" fill="#fff" />
      <circle cx="54" cy="24" r="2.6" fill="#fff" />
      <circle cx="45" cy="35" r="2.6" fill="#fff" />
      <circle cx="56" cy="38" r="2.2" fill="#fff" />
    </svg>
  ),
  HK: (className) => (
    <svg className={className} viewBox="0 0 64 48" aria-hidden="true" focusable="false">
      <rect width="64" height="48" fill="#de2910" />
      <g transform="translate(32 24)" fill="#fff">
        <path d="M0-12c5.2 0 9.2 3.2 9.2 7.4 0 4.8-4.2 7.4-8 8.6C-.6-.4-1.2-6 0-12z" />
        <path d="M0-12c5.2 0 9.2 3.2 9.2 7.4 0 4.8-4.2 7.4-8 8.6C-.6-.4-1.2-6 0-12z" transform="rotate(72)" />
        <path d="M0-12c5.2 0 9.2 3.2 9.2 7.4 0 4.8-4.2 7.4-8 8.6C-.6-.4-1.2-6 0-12z" transform="rotate(144)" />
        <path d="M0-12c5.2 0 9.2 3.2 9.2 7.4 0 4.8-4.2 7.4-8 8.6C-.6-.4-1.2-6 0-12z" transform="rotate(216)" />
        <path d="M0-12c5.2 0 9.2 3.2 9.2 7.4 0 4.8-4.2 7.4-8 8.6C-.6-.4-1.2-6 0-12z" transform="rotate(288)" />
      </g>
      <circle cx="32" cy="24" r="1.7" fill="#de2910" />
    </svg>
  ),
  SG: (className) => (
    <svg className={className} viewBox="0 0 64 48" aria-hidden="true" focusable="false">
      <path d="M0 0h64v24H0z" fill="#ef3340" />
      <path d="M0 24h64v24H0z" fill="#fff" />
      <circle cx="16" cy="12" r="8" fill="#fff" />
      <circle cx="19" cy="12" r="6.6" fill="#ef3340" />
      <g fill="#fff">
        <circle cx="30" cy="6.5" r="1.5" />
        <circle cx="35" cy="10" r="1.5" />
        <circle cx="33" cy="16" r="1.5" />
        <circle cx="27" cy="16" r="1.5" />
        <circle cx="25" cy="10" r="1.5" />
      </g>
    </svg>
  ),
}

export const CountryFlag = memo(function CountryFlag({ countryCode, className = "", size = "md" }: CountryFlagProps) {
  const code = countryCodeToFlagLabel(countryCode)
  if (!code) {
    return null
  }

  const emoji = countryCodeToFlagEmoji(code)
  const renderFallback = flagFallbacks[code]
  const fallbackClassName = "country-flag__fallback"

  return (
    <span
      className={`country-flag ${renderFallback ? "country-flag--svg" : ""} ${sizeClasses[size]} ${className}`.trim()}
      role="img"
      aria-label={code}
      title={code}
      data-country-code={code}
      data-country-flag-fallback={renderFallback ? "svg" : "emoji"}
    >
      <span className="country-flag__emoji" aria-hidden="true">{emoji}</span>
      {renderFallback?.(fallbackClassName)}
    </span>
  )
})
