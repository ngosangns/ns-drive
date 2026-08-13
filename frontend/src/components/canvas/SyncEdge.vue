<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getBezierPath } from '@vue-flow/core'
import type { FileTransferInfo } from '@/api/types'
import {
  QUEUE_START,
  desiredT,
  isSuccessStatus,
  statusColorToken,
  transfersToParticles,
  type FileParticle,
  type ParticleDir,
} from '@/lib/fileParticles'

const props = defineProps<{
  id: string
  sourceX: number
  sourceY: number
  targetX: number
  targetY: number
  sourcePosition: string
  targetPosition: string
  selected?: boolean
  data?: {
    action?: string
    running?: boolean
    transfers?: FileTransferInfo[]
  }
}>()

type LiveDot = FileParticle & { x: number; y: number; opacity: number }

const pathEl = ref<SVGPathElement | null>(null)
const dots = ref<LiveDot[]>([])
const live = new Map<string, LiveDot>()

const path = computed(() =>
  getBezierPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    targetX: props.targetX,
    targetY: props.targetY,
    sourcePosition: props.sourcePosition as never,
    targetPosition: props.targetPosition as never,
  }),
)

const d = computed(() => path.value[0])
const labelX = computed(() => path.value[1])
const labelY = computed(() => path.value[2])

const particles = computed(() =>
  transfersToParticles(props.data?.transfers, props.data?.action || 'push'),
)

let raf = 0

function spawnT(dir: ParticleDir, status: string): number {
  if (isSuccessStatus(status)) return desiredT({ status, progress: 100, dir, slot: 0, queued: 0 })
  return dir === 1 ? QUEUE_START * 0.4 : 1 - QUEUE_START * 0.4
}

function syncLive() {
  const incoming = particles.value
  const seen = new Set<string>()
  for (const p of incoming) {
    seen.add(p.id)
    const prev = live.get(p.id)
    if (!prev) {
      live.set(p.id, {
        ...p,
        t: spawnT(p.dir, p.status),
        x: 0,
        y: 0,
        opacity: isSuccessStatus(p.status) ? 0 : 1,
      })
      continue
    }
    prev.status = p.status
    prev.progress = p.progress
    prev.dir = p.dir
    prev.slot = p.slot
    prev.queued = p.queued
    if (prev.opacity <= 0 && !isSuccessStatus(p.status)) prev.opacity = 1
  }
  const running = !!props.data?.running
  for (const [id, m] of live) {
    if (seen.has(id)) continue
    if (!running && m.opacity <= 0) {
      live.delete(id)
      continue
    }
    if (!running) {
      if (isSuccessStatus(m.status)) {
        m.progress = 100
      }
    }
  }
}

function stepLive(): boolean {
  let moving = false
  const running = !!props.data?.running
  for (const [id, m] of live) {
    const target = desiredT(m)
    if (isSuccessStatus(m.status) || running) {
      const next = m.t + (target - m.t) * 0.1
      if (Math.abs(next - m.t) > 0.001) moving = true
      m.t = next
    }
    if (isSuccessStatus(m.status) && Math.abs(m.t - target) < 0.03) {
      m.opacity = Math.max(0, m.opacity - 0.05)
      if (m.opacity > 0) moving = true
      else if (!running) live.delete(id)
    } else if (!running && !isSuccessStatus(m.status)) {
      m.opacity = Math.max(0, m.opacity - 0.06)
      if (m.opacity > 0) moving = true
      else live.delete(id)
    }
  }
  return moving || (running && live.size > 0)
}

function place() {
  const el = pathEl.value
  const len = el?.getTotalLength() ?? 0
  const out: LiveDot[] = []
  for (const m of live.values()) {
    if (m.opacity <= 0.02) continue
    if (!el || len <= 0) {
      out.push({ ...m, x: 0, y: 0 })
      continue
    }
    const pt = el.getPointAtLength(Math.max(0, Math.min(1, m.t)) * len)
    out.push({ ...m, x: pt.x, y: pt.y })
  }
  dots.value = out
}

function tick() {
  syncLive()
  stepLive()
  place()
  raf = requestAnimationFrame(tick)
}

onMounted(() => {
  raf = requestAnimationFrame(tick)
})
onBeforeUnmount(() => cancelAnimationFrame(raf))
watch(
  () => [props.sourceX, props.sourceY, props.targetX, props.targetY],
  () => place(),
)

function fill(status: string): string {
  return `var(${statusColorToken(status)})`
}
</script>

<template>
  <g :data-testid="`op-row-${id}`">
    <path ref="pathEl" :d="d" fill="none" class="pointer-events-none opacity-0" />
    <BaseEdge
      :id="id"
      :path="d"
      :style="{
        stroke: selected || data?.running ? 'var(--color-accent-strong)' : 'var(--color-border)',
        strokeWidth: selected || data?.running ? 2.4 : 1.6,
      }"
    />
    <circle
      v-for="dot in dots"
      :key="dot.id"
      :cx="dot.x"
      :cy="dot.y"
      r="3.5"
      :fill="fill(dot.status)"
      :opacity="dot.opacity"
      class="pointer-events-none"
    />
    <EdgeLabelRenderer>
      <div
        class="nodrag nopan rounded-md border border-border bg-surface px-1.5 py-0.5 font-mono text-[10px] font-semibold uppercase text-text"
        :style="{
          position: 'absolute',
          transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
          pointerEvents: 'all',
        }"
        :data-testid="`canvas-edge-${id}`"
      >
        {{ data?.action || 'push' }}
      </div>
    </EdgeLabelRenderer>
  </g>
</template>
