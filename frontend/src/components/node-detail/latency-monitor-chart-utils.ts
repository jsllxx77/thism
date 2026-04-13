export function shouldRenderLatencyDots(rangeSeconds: number, pointCount: number): boolean {
  return rangeSeconds < 86400 && pointCount <= 120
}
