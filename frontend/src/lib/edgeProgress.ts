export function formatFileProgressBadge(
  filesTransferred: number | undefined,
  totalFiles: number | undefined,
  knownTransfers: number,
): string {
  const total = totalFiles ?? 0
  if (total > 0) {
    const completed = Math.min(Math.max(filesTransferred ?? 0, 0), total)
    return `${completed}/${total}`
  }
  return knownTransfers > 0 ? String(knownTransfers) : ''
}

export function formatTransferProgress(progress: number | undefined): string {
  const value = Number.isFinite(progress) ? Number(progress) : 0
  return `${Math.round(Math.min(Math.max(value, 0), 100))}%`
}
