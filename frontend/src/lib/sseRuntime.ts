import { shouldApplyLiveEvent } from '@/lib/runtimeSync'
import type { RuntimeSnapshot } from '@/api/types'

export const SYNC_TOPICS = ['sync:started', 'sync:progress', 'sync:completed', 'sync:failed'] as const
export const FLOW_RUN_TOPIC = 'flow:execution'
export const LEGACY_FLOW_TOPIC = 'board:execution'

export function parseJsonObject(raw: string): Record<string, unknown> | null {
  try {
    const data = JSON.parse(raw) as unknown
    if (!data || typeof data !== 'object') return null
    return data as Record<string, unknown>
  } catch {
    return null
  }
}

export function parseFlowPayload(data: Record<string, unknown>): {
  id: string
  status: string
  opId?: string
  error?: string
} | null {
  const rawId = data.flow_id ?? data.board_id
  const rawStatus = data.status
  if (typeof rawId !== 'string' || typeof rawStatus !== 'string') return null
  const id = rawId
  const status = rawStatus
  if (!id || !status) return null
  const opId = data.op_id ? String(data.op_id) : data.node_id ? String(data.node_id) : undefined
  let error: string | undefined
  if (data.error) error = String(data.error)
  else if (status === 'failed' && data.action) error = String(data.action)
  return { id, status, opId, error }
}

export function shouldReloadFlows(domain: string, dirty: boolean): boolean {
  if (dirty) return false
  return domain === 'flows' || domain === ''
}

export function shouldReloadRemotes(domain: string): boolean {
  return domain === 'remotes'
}

export function acceptLive(snapshotSeen: boolean, snapshotRev: number, data: Record<string, unknown>): boolean {
  return shouldApplyLiveEvent(snapshotSeen, snapshotRev, data.revision)
}

export function isSyncTopic(topic: string): boolean {
  return (SYNC_TOPICS as readonly string[]).includes(topic)
}

export function isFlowTopic(topic: string): boolean {
  return topic === FLOW_RUN_TOPIC || topic === LEGACY_FLOW_TOPIC
}

export function parseSnapshot(raw: string): RuntimeSnapshot | null {
  const data = parseJsonObject(raw)
  return data as RuntimeSnapshot | null
}
