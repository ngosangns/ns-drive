<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getBezierPath } from '@vue-flow/core'
import type { FileTransferInfo } from '@/api/types'
import { useI18n } from 'vue-i18n'
import {
  PhWarningCircle,
  PhFile,
  PhSpinner,
  PhCheckCircle,
} from '@phosphor-icons/vue'
import {
  QUEUE_START,
  desiredT,
  isSuccessStatus,
  statusColorToken,
  transfersToParticles,
  isBidirectionalAction,
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
    flowRunning?: boolean
    transfers?: FileTransferInfo[]
  }
}>()

type LiveDot = FileParticle & { x: number; y: number; opacity: number }

const pathEl = ref<SVGPathElement | null>(null)
const dots = ref<LiveDot[]>([])
const live = new Map<string, LiveDot>()
const panel = ref<'files' | 'errors' | 'processed' | null>(null)
const panelAnchor = ref<'source' | 'target'>('source')
const { t } = useI18n()

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
const particles = computed(() =>
  transfersToParticles(props.data?.transfers, props.data?.action || 'push'),
)

const transfers = computed(() => props.data?.transfers ?? [])
const errorFiles = computed(() => transfers.value.filter((file) => file.status === 'failed'))
const processedFiles = computed(() =>
  transfers.value.filter((file) => file.status === 'completed' || file.status === 'checked'),
)
const visibleFiles = computed(() => {
  const rank: Record<string, number> = { transferring: 0, checking: 1, failed: 2, pending: 3, checked: 4, completed: 5 }
  return [...transfers.value].sort(
    (a, b) => (rank[a.status] ?? 9) - (rank[b.status] ?? 9) || a.name.localeCompare(b.name),
  )
})
const isBidirectional = computed(() => isBidirectionalAction(props.data?.action))
const sourceTriggerX = computed(() => nodeCenterX(props.sourceX, props.sourcePosition))
const targetTriggerX = computed(() => nodeCenterX(props.targetX, props.targetPosition))
const panelX = computed(() => (panelAnchor.value === 'target' ? targetTriggerX.value : sourceTriggerX.value) + 24)
const panelY = computed(() => (panelAnchor.value === 'target' ? props.targetY : props.sourceY) + 28)
const markerId = computed(() => `sync-edge-arrow-${props.id.replace(/[^a-zA-Z0-9_-]/g, '-')}`)
const edgeColor = computed(() =>
  props.selected || props.data?.running ? 'var(--color-accent-strong)' : 'var(--color-border)',
)

function nodeCenterX(handleX: number, position: string): number {
  if (position.toLowerCase() === 'right') return handleX - 96
  if (position.toLowerCase() === 'left') return handleX + 96
  return handleX
}

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
  const running = !!props.data?.flowRunning
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
  const running = !!props.data?.flowRunning
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

function fileBaseName(path: string): string {
  const parts = path.split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function statusLabel(status: string): string {
  if (status === 'failed') return t('workspace.fileTabs.failed')
  if (status === 'completed' || status === 'checked') return t('workspace.fileTabs.complete')
  if (status === 'transferring' || status === 'checking') return t('workspace.fileTabs.syncing')
  return t('workspace.fileTabs.pending')
}

function rowClass(status: string): string {
  if (status === 'failed') return 'bg-danger/10 text-danger'
  if (status === 'completed' || status === 'checked') return 'bg-success/10 text-success'
  if (status === 'transferring' || status === 'checking') return 'bg-warning/10 text-running'
  return 'bg-bg/60 text-text-muted'
}

function togglePanel(next: 'files' | 'errors' | 'processed', anchor: 'source' | 'target', event: MouseEvent) {
  event.stopPropagation()
  panelAnchor.value = anchor
  panel.value = panel.value === next ? null : next
}

watch(
  () => [props.selected, props.data?.flowRunning],
  ([selected, flowRunning]) => {
    if (selected && flowRunning) {
      panelAnchor.value = 'source'
      panel.value = 'files'
    }
    else if (!selected || !flowRunning) panel.value = null
  },
)
</script>

<template>
  <g :data-testid="`op-row-${id}`">
    <defs>
      <marker
        :id="markerId"
        markerWidth="10"
        markerHeight="10"
        markerUnits="userSpaceOnUse"
        viewBox="0 0 10 10"
        refX="8"
        refY="5"
        orient="auto-start-reverse"
      >
        <path d="M 0 0 L 10 5 L 0 10 z" :fill="edgeColor" />
      </marker>
    </defs>
    <path ref="pathEl" :d="d" fill="none" class="pointer-events-none opacity-0" />
    <BaseEdge
      :id="id"
      :path="d"
      :marker-end="`url(#${markerId})`"
      :marker-start="isBidirectional ? `url(#${markerId})` : undefined"
      :data-testid="`canvas-edge-${id}`"
      class="cursor-pointer"
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
        v-if="data?.flowRunning"
        class="pointer-events-none nodrag nopan absolute inset-0"
        :style="{
          pointerEvents: 'none',
        }"
      >
        <button
          type="button"
          class="pointer-events-auto absolute flex h-6 w-6 -translate-x-1/2 cursor-pointer items-center justify-center rounded-full border bg-surface shadow-sm transition-colors hover:border-danger hover:text-danger"
          :class="errorFiles.length ? 'border-danger text-danger' : 'border-border text-text-dim'"
          :style="{ left: `${sourceTriggerX}px`, top: `${sourceY + 28}px` }"
          :aria-label="t('workspace.canvas.errorFiles')"
          :data-testid="`canvas-edge-errors-${id}`"
          @click="togglePanel('errors', 'source', $event)"
        >
          <PhWarningCircle :size="15" :weight="errorFiles.length ? 'fill' : 'regular'" />
          <span v-if="errorFiles.length" class="absolute -right-1 -top-1 min-w-3 rounded-full bg-danger px-0.5 text-[9px] leading-3 text-white">
            {{ errorFiles.length }}
          </span>
        </button>
        <button
          type="button"
          class="pointer-events-auto absolute flex h-6 -translate-x-1/2 cursor-pointer items-center gap-1 rounded-full border border-border bg-surface px-1.5 text-text-dim shadow-sm transition-colors hover:border-success hover:text-success"
          :style="{ left: `${targetTriggerX}px`, top: `${targetY + 28}px` }"
          :aria-label="t('workspace.canvas.processedFiles')"
          :data-testid="`canvas-edge-processed-${id}`"
          @click="togglePanel('processed', 'target', $event)"
        >
          <PhCheckCircle :size="14" weight="fill" />
          <span v-if="processedFiles.length" class="text-[10px] font-semibold tabular-nums">{{ processedFiles.length }}</span>
        </button>
      </div>
      <div
        v-if="panel"
        class="nodrag nopan absolute z-20 w-72 rounded-lg border border-border bg-surface text-text shadow-xl"
        :style="{
          left: `${panelX}px`,
          top: `${panelY}px`,
          transform: 'translateY(-50%)',
          pointerEvents: 'all',
        }"
        :data-testid="`canvas-edge-files-${id}`"
        @wheel.stop
        @click.stop
      >
        <div class="flex items-center justify-between border-b border-border px-2.5 py-2 text-[11px] font-semibold uppercase tracking-wide">
          <span>
            {{
              panel === 'errors'
                ? t('workspace.canvas.errorFiles')
                : panel === 'processed'
                  ? t('workspace.canvas.processedFiles')
                  : t('workspace.canvas.processingFiles')
            }}
          </span>
          <button type="button" class="text-text-dim hover:text-text" @click="panel = null">×</button>
        </div>
        <div class="max-h-52 overflow-auto p-1" @wheel.stop>
          <div
            v-for="file in panel === 'errors' ? errorFiles : panel === 'processed' ? processedFiles : visibleFiles"
            :key="file.name"
            class="flex items-center gap-2 rounded px-1.5 py-1 text-xs"
            :class="rowClass(file.status)"
            :data-testid="`canvas-edge-file-${file.name}`"
          >
            <PhSpinner v-if="file.status === 'transferring' || file.status === 'checking'" :size="12" class="shrink-0 animate-spin" />
            <PhCheckCircle v-else-if="file.status === 'completed' || file.status === 'checked'" :size="12" weight="fill" />
            <PhWarningCircle v-else-if="file.status === 'failed'" :size="12" weight="fill" />
            <PhFile v-else :size="12" />
            <span class="min-w-0 flex-1 truncate" :title="file.name">{{ fileBaseName(file.name) }}</span>
            <span class="shrink-0 text-[10px] uppercase">{{ statusLabel(file.status) }}</span>
          </div>
          <p v-if="!(panel === 'errors' ? errorFiles : panel === 'processed' ? processedFiles : visibleFiles).length" class="px-1.5 py-2 text-xs text-text-muted">
            {{
              panel === 'errors'
                ? t('workspace.canvas.noErrors')
                : panel === 'processed'
                  ? t('workspace.canvas.noProcessedFiles')
                  : t('workspace.canvas.preparingFiles')
            }}
          </p>
        </div>
      </div>
    </EdgeLabelRenderer>
  </g>
</template>
