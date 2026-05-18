/** Shared helpers for opsfleet-backend host resource rings (header / dashboard). */

export function clampPct(n: number): number {
  if (!Number.isFinite(n) || n < 0) return 0
  if (n > 100) return 100
  return n
}

export function usageRingColor(percentage: number): string {
  const p = clampPct(percentage)
  if (p < 50) return '#67c23a'
  if (p < 80) return '#e6a23c'
  return '#f56c6c'
}
