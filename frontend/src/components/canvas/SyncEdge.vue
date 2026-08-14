<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { BaseEdge, EdgeLabelRenderer, getSmoothStepPath } from '@vue-flow/core'
import type { Position } from '@vue-flow/core'
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
import { formatFileProgressBadge, formatTransferProgress } from '@/lib/edgeProgress'

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
    syncStatus?: string
    progress?: number
    filesTransferred?: number
    totalFiles?: number
    checks?: number
    totalChecks?: number
    stage?: string
    stageDetail?: string
    transfers?: FileTransferInfo[]
  }
}>()

type LiveDot = FileParticle & { x: number; y: number; opacity: number }
type PanelKind = 'files' | 'errors' | 'processed'

const pathEl = ref<SVGPathElement | null>(null)
const dots = ref<LiveDot[]>([])
const live = new Map<string, LiveDot>()
const panel = ref<PanelKind | null>(null)
const triggerLayerEl = ref<HTMLElement | null>(null)
const panelEl = ref<HTMLElement | null>(null)
const { t } = useI18n()

const path = computed(() =>
  getSmoothStepPath({
    sourceX: props.sourceX,
    sourceY: props.sourceY,
    targetX: props.targetX,
    targetY: props.targetY,
    sourcePosition: props.sourcePosition as Position,
    targetPosition: props.targetPosition as Position,
    offset: 18,
    borderRadius: 6,
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
const panelFiles = computed(() => {
  if (panel.value === 'errors') return errorFiles.value
  if (panel.value === 'processed') return processedFiles.value
  return visibleFiles.value
})
const isBidirectional = computed(() => isBidirectionalAction(props.data?.action))
const sourceTriggerX = computed(() => nodeCenterX(props.sourceX, props.sourcePosition))
const targetTriggerX = computed(() => nodeCenterX(props.targetX, props.targetPosition))
const statusTriggerOffsetY = 44
const edgeCenterX = computed(() => path.value[1])
const edgeCenterY = computed(() => path.value[2])
const totalFiles = computed(() => props.data?.totalFiles || transfers.value.length)
const resolvedFiles = computed(() => processedFiles.value.length)
const fileProgressBadge = computed(() =>
  formatFileProgressBadge(props.data?.filesTransferred, props.data?.totalFiles, transfers.value.length),
)
const overallProgress = computed(() =>
  props.data?.progress === undefined ? '' : formatTransferProgress(props.data.progress),
)
const hasCheckingFiles = computed(() => transfers.value.some((file) => file.status === 'checking'))
const hasTransferringFiles = computed(() => transfers.value.some((file) => file.status === 'transferring'))
const allFilesComplete = computed(() => transfers.value.length > 0 && resolvedFiles.value === transfers.value.length)

type StepState = 'pending' | 'active' | 'complete' | 'error'
type ResolvingStep = {
  key: 'preparing' | 'resolving' | 'checking' | 'transferring' | 'completed'
  state: StepState
  detail?: string
}

const resolvingSteps = computed<ResolvingStep[]>(() => {
  const running = !!props.data?.flowRunning
  const failed = props.data?.syncStatus === 'failed' || errorFiles.value.length > 0
  const hasFileList = transfers.value.length > 0
  const checksDone = (props.data?.checks ?? 0) > 0 || hasCheckingFiles.value || hasTransferringFiles.value || resolvedFiles.value > 0
  const transferDone = resolvedFiles.value > 0 || allFilesComplete.value
  const progressDetail = props.data?.progress === undefined ? '' : ` · ${Math.round(props.data.progress)}%`

  return [
    {
      key: 'preparing',
      state: hasFileList ? 'complete' : running ? 'active' : 'pending',
    },
    {
      key: 'resolving',
      state: hasFileList ? 'complete' : running ? 'active' : 'pending',
    },
    {
      key: 'checking',
      state: hasCheckingFiles.value ? 'active' : checksDone ? 'complete' : 'pending',
      detail: props.data?.totalChecks ? `${props.data.checks ?? 0} / ${props.data.totalChecks}` : undefined,
    },
    {
      key: 'transferring',
      state: hasTransferringFiles.value ? 'active' : transferDone ? 'complete' : 'pending',
      detail: totalFiles.value
        ? `${props.data?.filesTransferred ?? resolvedFiles.value} / ${totalFiles.value}${progressDetail}`
        : progressDetail.slice(3) || undefined,
    },
    {
      key: 'completed',
      state: failed ? 'error' : props.data?.syncStatus === 'completed' || allFilesComplete.value ? 'complete' : 'pending',
    },
  ]
})

const triggerClass = computed(() => {
  if (errorFiles.value.length || props.data?.syncStatus === 'failed') return 'border-danger text-danger'
  if (allFilesComplete.value || props.data?.syncStatus === 'completed') return 'border-success text-success'
  return 'border-border text-text-dim'
})
const panelStyle = computed<Record<string, string>>(() => {
  if (panel.value === 'errors') {
    return {
      left: `${sourceTriggerX.value + 18}px`,
      top: `${props.sourceY + statusTriggerOffsetY}px`,
      transform: 'translateY(-50%)',
      pointerEvents: 'all',
    }
  }
  if (panel.value === 'processed') {
    return {
      left: `${targetTriggerX.value - 18}px`,
      top: `${props.targetY + statusTriggerOffsetY}px`,
      transform: 'translate(-100%, -50%)',
      pointerEvents: 'all',
    }
  }
  return {
    left: `${edgeCenterX.value}px`,
    top: `${edgeCenterY.value + 22}px`,
    transform: 'translateX(-50%)',
    pointerEvents: 'all',
  }
})
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
  window.addEventListener('click', onDocumentClick)
})
onBeforeUnmount(() => {
  cancelAnimationFrame(raf)
  window.removeEventListener('click', onDocumentClick)
})
watch(
  () => [props.sourceX, props.sourceY, props.targetX, props.targetY],
  () => place(),
)
watch(
  () => [props.selected, props.data?.running],
  ([selected, running]) => {
    if (selected && running && !panel.value) panel.value = 'files'
  },
  { immediate: true },
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

function activeProgressLabel(file: FileTransferInfo): string {
  return file.status === 'transferring' ? formatTransferProgress(file.progress) : ''
}

function rowClass(status: string): string {
  if (status === 'failed') return 'bg-danger/10 text-danger'
  if (status === 'completed' || status === 'checked') return 'bg-success/10 text-success'
  if (status === 'transferring' || status === 'checking') return 'bg-warning/10 text-running'
  return 'bg-bg/60 text-text-muted'
}

function rowStyle(file: FileTransferInfo): Record<string, string> {
  const progress = Math.min(100, Math.max(0, file.progress ?? 0))
  const color = `var(${statusColorToken(file.status)})`
  const fill = `color-mix(in srgb, ${color} 18%, transparent)`
  return {
    backgroundImage: `linear-gradient(90deg, ${fill} 0%, ${fill} ${progress}%, transparent ${progress}%, transparent 100%)`,
  }
}

function togglePanel(next: PanelKind, event: MouseEvent) {
  event.stopPropagation()
  panel.value = next
}

function onDocumentClick(event: MouseEvent) {
  if (!panel.value) return
  const target = event.target
  if (!(target instanceof Node)) return
  if (triggerLayerEl.value?.contains(target) || panelEl.value?.contains(target)) return
  panel.value = null
}
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
      stroke="var(--color-surface)"
      stroke-width="1.25"
      vector-effect="non-scaling-stroke"
      :opacity="dot.opacity"
      class="pointer-events-none"
    />
    <EdgeLabelRenderer>
      <div
        v-if="data?.flowRunning || transfers.length"
        ref="triggerLayerEl"
        class="pointer-events-none nodrag nopan absolute inset-0"
        :style="{
          pointerEvents: 'none',
        }"
      >
        <button
          type="button"
          class="pointer-events-auto absolute flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border bg-surface text-text-dim shadow-sm transition-colors"
          :class="[errorFiles.length ? 'border-danger text-danger' : 'border-border', panel === 'errors' && 'ring-2 ring-accent/30']"
          :style="{ left: `${sourceTriggerX}px`, top: `${sourceY + statusTriggerOffsetY}px` }"
          :aria-label="t('workspace.canvas.errorFiles')"
          :aria-expanded="panel === 'errors'"
          :data-testid="`canvas-edge-errors-${id}`"
          @click="togglePanel('errors', $event)"
        >
          <PhWarningCircle :size="15" :weight="errorFiles.length ? 'fill' : 'regular'" class="shrink-0" />
          <span v-if="errorFiles.length" class="absolute -right-1 -top-1 min-w-3 rounded-full bg-danger px-0.5 text-[9px] leading-3 text-white">
            {{ errorFiles.length }}
          </span>
        </button>
        <button
          type="button"
          class="pointer-events-auto absolute flex h-6 w-6 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border border-border bg-surface text-text-dim shadow-sm transition-colors"
          :class="[allFilesComplete ? 'border-success text-success' : '', panel === 'processed' && 'ring-2 ring-accent/30']"
          :style="{ left: `${targetTriggerX}px`, top: `${targetY + statusTriggerOffsetY}px` }"
          :aria-label="t('workspace.canvas.processedFiles')"
          :aria-expanded="panel === 'processed'"
          :data-testid="`canvas-edge-processed-${id}`"
          @click="togglePanel('processed', $event)"
        >
          <PhCheckCircle :size="15" weight="fill" class="shrink-0" />
          <span v-if="processedFiles.length" class="absolute -right-1 -top-1 min-w-3 rounded-full bg-success px-0.5 text-[9px] leading-3 text-white">
            {{ processedFiles.length }}
          </span>
        </button>
        <button
          type="button"
          class="pointer-events-auto absolute flex h-7 w-7 -translate-x-1/2 -translate-y-1/2 cursor-pointer items-center justify-center rounded-full border bg-surface shadow-sm transition-colors"
          :class="[triggerClass, panel === 'files' && 'bg-surface-hover ring-2 ring-accent/30']"
          :style="{ left: `${edgeCenterX}px`, top: `${edgeCenterY}px` }"
          :aria-label="t('workspace.canvas.processingFiles')"
          :aria-expanded="panel === 'files'"
          :data-testid="`canvas-edge-files-trigger-${id}`"
          @click="togglePanel('files', $event)"
        >
          <PhFile :size="15" weight="bold" class="shrink-0" />
          <span v-if="fileProgressBadge" class="absolute -right-1 -top-1 min-w-3 rounded-full bg-accent-strong px-0.5 text-[9px] leading-3 text-text">
            {{ fileProgressBadge }}
          </span>
        </button>
      </div>
      <div
        v-if="panel"
        ref="panelEl"
        class="nodrag nopan absolute z-[1000] w-80 rounded-lg border border-border bg-surface text-text shadow-xl"
        :style="panelStyle"
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
          <span class="font-mono text-[10px] text-text-muted">
            <template v-if="panel === 'files'">
              {{ resolvedFiles }} / {{ totalFiles }}<template v-if="overallProgress"> · {{ overallProgress }}</template>
            </template>
            <template v-else-if="panel === 'errors'">{{ errorFiles.length }}</template>
            <template v-else>{{ processedFiles.length }}</template>
          </span>
        </div>
        <div v-if="panel === 'files'" class="border-b border-border px-2.5 py-2">
          <div v-if="data?.stage" class="mb-2 flex items-center justify-between gap-2 rounded bg-accent/10 px-1.5 py-1 text-[11px] text-text">
            <span class="font-semibold">{{ t(`workspace.syncStages.${data.stage}`) }}</span>
            <span v-if="data.stageDetail" class="min-w-0 truncate font-mono text-[10px] text-text-muted">{{ data.stageDetail }}</span>
          </div>
          <div class="mb-1.5 text-[10px] font-bold uppercase tracking-wide text-text-muted">
            {{ t('workspace.syncLabels.resolving') }}
          </div>
          <div class="flex flex-col gap-1">
            <div v-for="step in resolvingSteps" :key="step.key" class="flex items-center gap-2 rounded bg-bg/60 px-1.5 py-1 text-[11px]">
              <PhCheckCircle v-if="step.state === 'complete'" :size="13" weight="fill" class="shrink-0 text-success" />
              <PhWarningCircle v-else-if="step.state === 'error'" :size="13" weight="fill" class="shrink-0 text-danger" />
              <PhSpinner v-else-if="step.state === 'active'" :size="13" class="shrink-0 animate-spin text-running" />
              <span v-else class="h-2 w-2 shrink-0 rounded-full border border-border-muted" />
              <span class="min-w-0 flex-1 font-semibold" :class="step.state === 'pending' ? 'text-text-dim' : 'text-text'">
                {{ t(`workspace.syncLabels.${step.key}`) }}
              </span>
              <span v-if="step.detail" class="font-mono text-[10px] text-text-muted">{{ step.detail }}</span>
            </div>
          </div>
        </div>
        <div class="flex max-h-52 flex-col gap-1 overflow-auto p-1.5" @wheel.stop>
          <div
            v-for="file in panelFiles"
            :key="file.name"
            class="flex items-center gap-2 rounded px-1.5 py-1 text-xs"
            :class="rowClass(file.status)"
            :style="rowStyle(file)"
            :data-testid="`canvas-edge-file-${file.name}`"
          >
            <PhSpinner v-if="file.status === 'transferring' || file.status === 'checking'" :size="12" class="shrink-0 animate-spin" />
            <PhCheckCircle v-else-if="file.status === 'completed' || file.status === 'checked'" :size="12" weight="fill" />
            <PhWarningCircle v-else-if="file.status === 'failed'" :size="12" weight="fill" />
            <PhFile v-else :size="12" />
            <span class="min-w-0 flex-1 truncate" :title="file.name">{{ fileBaseName(file.name) }}</span>
            <span class="shrink-0 text-[10px] uppercase">
              {{ statusLabel(file.status) }}<template v-if="activeProgressLabel(file)"> · {{ activeProgressLabel(file) }}</template>
            </span>
          </div>
          <p v-if="!panelFiles.length" class="px-1.5 py-2 text-xs text-text-muted">
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
