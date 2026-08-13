/**
 * Map rclone file transfers onto edge particles.
 * Queued statuses stay on the edge; only success is allowed to enter the target.
 */
import type { FileTransferInfo } from '@/api/types'

export const PARTICLE_CAP = 24

/** Queue band on the edge (source → target). Success leaves this band. */
export const QUEUE_START = 0.18
export const QUEUE_END = 0.7
export const TARGET_T = 0.98

export type ParticleDir = 1 | -1

export interface FileParticle {
  id: string
  /** Desired position 0 = source, 1 = target. SyncEdge lerps toward this. */
  t: number
  status: string
  dir: ParticleDir
  progress: number
  /** Slot among queued (non-success) particles, for packing. */
  slot: number
  queued: number
}

const STATUS_RANK: Record<string, number> = {
  transferring: 0,
  failed: 1,
  pending: 2,
  checking: 3,
  checked: 4,
  completed: 5,
}

export function statusColorToken(status: string): string {
  switch (status) {
    case 'transferring':
      return '--color-accent-strong'
    case 'failed':
      return '--color-danger'
    case 'completed':
      return '--color-success'
    case 'checking':
    case 'checked':
      return '--color-info'
    default:
      return '--color-text-dim'
  }
}

export function isBidirectionalAction(action: string | undefined): boolean {
  return action === 'bi' || action === 'bi-resync'
}

export function isSuccessStatus(status: string): boolean {
  return status === 'completed' || status === 'checked'
}

/**
 * Position a particle should seek.
 * Success → target node. Everything else stays in the queue band.
 * Transferring uses file progress but still cannot reach the target.
 */
export function desiredT(p: {
  status: string
  progress: number
  dir: ParticleDir
  slot: number
  queued: number
}): number {
  const forward = p.dir === 1
  if (isSuccessStatus(p.status)) {
    return forward ? TARGET_T : 1 - TARGET_T
  }
  const queued = Math.max(p.queued, 1)
  const span = QUEUE_END - QUEUE_START
  let local: number
  if (p.status === 'transferring') {
    const pr = Math.max(0, Math.min(1, Number(p.progress) / 100 || 0))
    local = QUEUE_START + pr * span
  } else {
    local = QUEUE_START + ((p.slot + 0.5) / queued) * span
  }
  if (local > QUEUE_END) local = QUEUE_END
  return forward ? local : 1 - local
}

/** Pending files wait off-edge; only work in flight is drawn. */
export function isEdgeVisibleStatus(status: string): boolean {
  switch (status) {
    case 'transferring':
    case 'checking':
    case 'failed':
    case 'completed':
    case 'checked':
      return true
    default:
      return false
  }
}

/**
 * One particle per in-flight transfer, capped.
 * Pending is omitted — those files are not on the edge yet.
 */
export function transfersToParticles(
  transfers: FileTransferInfo[] | undefined | null,
  action = 'push',
): FileParticle[] {
  const list = (Array.isArray(transfers) ? transfers : []).filter((tr) =>
    isEdgeVisibleStatus(tr.status || ''),
  )
  list.sort((a, b) => rank(a.status) - rank(b.status))
  const picked = list.length <= PARTICLE_CAP ? list : list.slice(0, PARTICLE_CAP)
  const bi = isBidirectionalAction(action)
  const queuedIdx: number[] = []
  for (let i = 0; i < picked.length; i++) {
    if (!isSuccessStatus(picked[i].status || '')) queuedIdx.push(i)
  }
  const queued = queuedIdx.length
  let q = 0
  return picked.map((tr, i) => {
    const status = tr.status || 'pending'
    const dir: ParticleDir = bi && i % 2 === 1 ? -1 : 1
    const slot = isSuccessStatus(status) ? 0 : q++
    const progress = Number(tr.progress) || 0
    const p = { status, progress, dir, slot, queued }
    return {
      id: particleId(tr, i),
      t: desiredT(p),
      status,
      dir,
      progress,
      slot,
      queued,
    }
  })
}

function rank(status: string | undefined): number {
  return STATUS_RANK[status ?? ''] ?? 9
}

function particleId(tr: FileTransferInfo, index: number): string {
  const name = (tr.name || '').trim()
  return name || `file-${index}`
}
