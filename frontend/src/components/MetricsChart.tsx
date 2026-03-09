import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts"
import { useLanguage } from "../i18n/language"

export type DataPoint = { ts: number; value: number | null }

type ValueFormatter = (value: number) => string
type TimeFormatter = (value: number) => string

type Props = {
  data: DataPoint[]
  label: string
  color: string
  unit?: string
  domain?: [number | string, number | string]
  valueFormatter?: ValueFormatter
  axisTickFormatter?: ValueFormatter
  xAxisTickFormatter?: TimeFormatter
  tooltipLabelFormatter?: TimeFormatter
}

export function MetricsChart({
  data,
  label,
  color,
  unit = "%",
  domain = [0, 100],
  valueFormatter,
  axisTickFormatter,
  xAxisTickFormatter,
  tooltipLabelFormatter,
}: Props) {
  const { language } = useLanguage()
  const gradId = `grad-${label.replace(/\s+/g, "-")}`
  const formatTooltipValue = valueFormatter ?? ((value: number) => `${value.toFixed(1)}${unit}`)
  const formatAxisTick = axisTickFormatter
  const formatXAxisTick = xAxisTickFormatter ?? ((value: number) =>
    new Date(value * 1000).toLocaleTimeString(language, { hour: "2-digit", minute: "2-digit" })
  )
  const formatTooltipLabel = tooltipLabelFormatter ?? ((value: number) => new Date(value * 1000).toLocaleTimeString(language))

  return (
    <div className="panel-card enterprise-surface rounded-[24px] p-4">
      <p className="mb-3 text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-500 dark:text-slate-400">{label}</p>
      <ResponsiveContainer width="100%" height={120}>
        <AreaChart data={data} margin={{ top: 2, right: 0, left: -28, bottom: 0 }}>
          <defs>
            <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.18} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="ts"
            tickFormatter={formatXAxisTick}
            tick={{ fill: "#6b7280", fontSize: 9 }}
            axisLine={false}
            tickLine={false}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={domain}
            tickFormatter={formatAxisTick}
            tick={{ fill: "#6b7280", fontSize: 9 }}
            axisLine={false}
            tickLine={false}
          />
          <Tooltip
            contentStyle={{
              background: "hsl(var(--popover))",
              border: "1px solid hsl(var(--border))",
              borderRadius: 10,
              fontSize: 11,
              color: "hsl(var(--popover-foreground))",
              boxShadow: "0 12px 28px rgba(0,0,0,0.18)",
            }}
            itemStyle={{ color: "hsl(var(--popover-foreground))" }}
            labelStyle={{ color: "hsl(var(--popover-foreground))" }}
            labelFormatter={(value) => formatTooltipLabel(value as number)}
            formatter={(value) => [value == null ? "—" : formatTooltipValue(value as number), label]}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={`url(#${gradId})`}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
            connectNulls={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
