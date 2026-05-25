import { memo } from "react"
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

export const CountryFlag = memo(function CountryFlag({ countryCode, className = "", size = "md" }: CountryFlagProps) {
  const code = countryCodeToFlagLabel(countryCode)
  if (!code) {
    return null
  }

  const emoji = countryCodeToFlagEmoji(code)

  return (
    <span
      className={`country-flag ${sizeClasses[size]} ${className}`.trim()}
      role="img"
      aria-label={code}
      title={code}
      data-country-code={code}
    >
      <span className="country-flag__emoji" aria-hidden="true">{emoji}</span>
    </span>
  )
})
