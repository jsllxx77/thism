import { Card, Tag } from "antd"
import type { Node } from "../../lib/api"

type Props = {
  node: Node | null
}

export function NodeHero({ node }: Props) {
  return (
    <Card className="glass-panel !border-white/15 !rounded-xl [&_.ant-card-body]:p-5">
      <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-white">{node?.name ?? "Unknown node"}</h2>
          <p className="text-xs text-white/55 mt-1">
            {node?.ip || "—"} · {node?.os || "—"}/{node?.arch || "—"}
          </p>
        </div>
        <Tag
          color={node?.online ? "green" : "default"}
          className="!m-0 !text-xs !font-semibold !px-3 !py-1 self-start"
        >
          {node?.online ? "Online" : "Offline"}
        </Tag>
      </div>
    </Card>
  )
}
