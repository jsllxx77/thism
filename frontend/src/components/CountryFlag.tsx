import { memo } from "react"
import { countryCodeToFlagAssetPath, countryCodeToFlagLabel } from "../lib/flags"

type CountryFlagProps = {
  countryCode?: string | null
  className?: string
  size?: "sm" | "md" | "lg"
}

const sizeClasses: Record<NonNullable<CountryFlagProps["size"]>, string> = {
  sm: "h-[1.05rem] w-[1.05rem]",
  md: "h-[1.25rem] w-[1.25rem]",
  lg: "h-[1.5rem] w-[1.5rem]",
}

export const CountryFlag = memo(function CountryFlag({ countryCode, className = "", size = "md" }: CountryFlagProps) {
  const code = countryCodeToFlagLabel(countryCode)
  if (!code) {
    return null
  }

  const src = countryCodeToFlagAssetPath(code)

  return (
    <span
      className={`country-flag ${sizeClasses[size]} ${className}`.trim()}
      role="img"
      aria-label={code}
      title={code}
      data-country-code={code}
      data-country-flag-source="local-svg"
    >
      <img className="country-flag__image" src={src} alt="" aria-hidden="true" loading="lazy" />
    </span>
  )
})
