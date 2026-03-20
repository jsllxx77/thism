import type { Node } from "./api"

export const ONLINE_GRACE_PERIOD_SECONDS = 15

export function isNodeEffectivelyOnline(node: Pick<Node, "online" | "last_seen">, nowMs: number): boolean {
  if (node.online) return true
  if (node.last_seen <= 0) return false

  return Math.max(0, Math.floor(nowMs / 1000) - node.last_seen) <= ONLINE_GRACE_PERIOD_SECONDS
}

export function withEffectiveNodeStatus<T extends Pick<Node, "online" | "last_seen">>(node: T, nowMs: number): T {
  return { ...node, online: isNodeEffectivelyOnline(node, nowMs) }
}
