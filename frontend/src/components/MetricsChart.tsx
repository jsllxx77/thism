import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
} from "recharts"

type DataPoint = { ts: number; value: number }

type Props = {
  data: DataPoint[]
  label: string
  color: string
  unit?: string
  domain?: [number | string, number | string]
}

export function MetricsChart({ data, label, color, unit = "%", domain = [0, 100] }: Props) {
  const gradId = `grad-${label.replace(/\s+/g, "-")}`

  return (
    <div className="bg-white/[0.03] border border-white/10 rounded-xl p-4">
      <p className="text-xs text-white/40 mb-3">{label}</p>
      <ResponsiveContainer width="100%" height={120}>
        <AreaChart data={data} margin={{ top: 2, right: 0, left: -28, bottom: 0 }}>
          <defs>
            <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.3} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="ts"
            tickFormatter={(v: number) =>
              new Date(v * 1000).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
            }
            tick={{ fill: "#ffffff30", fontSize: 9 }}
            axisLine={false}
            tickLine={false}
            interval="preserveStartEnd"
          />
          <YAxis
            domain={domain}
            tick={{ fill: "#ffffff30", fontSize: 9 }}
            axisLine={false}
            tickLine={false}
          />
          <Tooltip
            contentStyle={{
              background: "#1a1a2e",
              border: "1px solid #ffffff20",
              borderRadius: 8,
              fontSize: 11,
            }}
            labelFormatter={(v) => new Date((v as number) * 1000).toLocaleTimeString()}
            formatter={(v) => [`${(v as number).toFixed(1)}${unit}`, label]}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke={color}
            fill={`url(#${gradId})`}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  )
}
