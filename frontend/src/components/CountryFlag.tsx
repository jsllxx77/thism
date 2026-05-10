import { memo } from "react"
import { countryCodeToFlagEmoji, countryCodeToFlagLabel } from "../lib/flags"

type CountryFlagProps = {
  countryCode?: string | null
  className?: string
  size?: "sm" | "md" | "lg"
}

const sizeClasses: Record<NonNullable<CountryFlagProps["size"]>, string> = {
  sm: "h-[0.82rem] w-[1.1rem] text-[10px]",
  md: "h-[0.92rem] w-[1.23rem] text-[11px]",
  lg: "h-[1.35rem] w-[1.8rem] text-sm",
}

export const CountryFlag = memo(function CountryFlag({ countryCode, className = "", size = "md" }: CountryFlagProps) {
  const code = countryCodeToFlagLabel(countryCode)
  if (!code) {
    return null
  }

  const emoji = countryCodeToFlagEmoji(code)

  return (
    <span
      className={`country-flag fi fi-${code.toLowerCase()} ${sizeClasses[size]} ${className}`.trim()}
      role="img"
      aria-label={code}
      title={code}
      data-country-code={code}
    >
      <span className="country-flag__emoji" aria-hidden="true">{emoji}</span>
      <span className="country-flag__code" aria-hidden="true">{code}</span>
    </span>
  )
})
