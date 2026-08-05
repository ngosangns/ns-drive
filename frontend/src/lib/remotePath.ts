/** Compose / parse rclone-style remote:path strings for workspace path fields. */

export function composeOp(remote: string, path: string): string {
  if (!remote || remote === 'local') return path || '/'
  const p = path || '/'
  return p.startsWith('/') ? `${remote}:${p}` : `${remote}:${p}`
}

export function parseComposed(composed: string): { remote: string; path: string } {
  const v = (composed ?? '').trim()
  if (!v) return { remote: '', path: '/' }
  if (v.startsWith('/')) return { remote: '', path: v }
  const colon = v.indexOf(':')
  if (colon > 0) {
    return {
      remote: v.slice(0, colon),
      path: v.slice(colon + 1) || '/',
    }
  }
  return { remote: '', path: v }
}

export function displayPath(remote: string, path: string, emptyLabel = '—'): string {
  return composeOp(remote, path) || emptyLabel
}
