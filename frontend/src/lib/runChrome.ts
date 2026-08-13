/** Run chrome visibility and edit lock — shared by bar, inspector, canvas. */

export function isActiveRun(status: string | undefined | null): boolean {
  return status === 'running' || status === 'cancelling'
}

export function runBarVisible(status: string | undefined | null): boolean {
  return isActiveRun(status)
}

export function fieldsLocked(status: string | undefined | null): boolean {
  return isActiveRun(status)
}
