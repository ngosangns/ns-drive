<script setup lang="ts">
/**
 * Run-only file strip. Reads backend-owned runtime (snapshot + SSE), not a
 * local reconstruction. Hidden when the active flow is not running.
 */
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import {
  PhSpinner,
  PhCheckCircle,
  PhWarningCircle,
  PhFile,
  PhStop,
} from '@phosphor-icons/vue'
import { useCanvasStore } from '@/stores/canvas'
import { useFlowsStore } from '@/stores/flows'
import { runBarVisible } from '@/lib/runChrome'
import type { FileTransferInfo } from '@/api/types'

const { t } = useI18n()
const canvas = useCanvasStore()
const flows = useFlowsStore()
const { activeFlow } = storeToRefs(canvas)
const { runStatus, opSyncStatus } = storeToRefs(flows)

const flowId = computed(() => activeFlow.value?.id ?? '')
const flowStatus = computed(() => {
  const id = flowId.value
  if (!id) return 'idle'
  return runStatus.value[id] || activeFlow.value?.status || 'idle'
})
const visible = computed(() => runBarVisible(flowStatus.value))
const snap = computed(() => (flowId.value ? opSyncStatus.value[flowId.value] ?? null : null))
const transfers = computed(() => snap.value?.transfers ?? [])

const rank: Record<string, number> = {
  transferring: 0,
  checking: 1,
  failed: 2,
  pending: 3,
  checked: 4,
  completed: 5,
}

const files = computed(() => {
  const list = [...transfers.value]
  list.sort((a, b) => (rank[a.status] ?? 9) - (rank[b.status] ?? 9) || a.name.localeCompare(b.name))
  return list
})

const counts = computed(() => {
  let syncing = 0
  let complete = 0
  let failed = 0
  let pending = 0
  for (const f of transfers.value) {
    switch (f.status) {
      case 'transferring':
      case 'checking':
        syncing++
        break
      case 'completed':
      case 'checked':
        complete++
        break
      case 'failed':
        failed++
        break
      default:
        pending++
    }
  }
  return { syncing, complete, failed, pending, total: transfers.value.length }
})

const progress = computed(() => {
  const p = snap.value?.progress ?? 0
  return Math.max(0, Math.min(100, p))
})

function fileBaseName(path: string): string {
  const parts = path.split(/[/\\]/)
  return parts[parts.length - 1] || path
}

function statusLabel(st: string): string {
  switch (st) {
    case 'transferring':
    case 'checking':
      return t('workspace.fileTabs.syncing')
    case 'completed':
    case 'checked':
      return t('workspace.fileTabs.complete')
    case 'failed':
      return t('workspace.fileTabs.failed')
    default:
      return t('workspace.fileTabs.pending')
  }
}

function rowClass(f: FileTransferInfo): string {
  if (f.status === 'transferring' || f.status === 'checking') return 'bg-warning/10 text-running'
  if (f.status === 'failed') return 'bg-danger/10 text-danger'
  if (f.status === 'completed' || f.status === 'checked') return 'bg-success/10 text-success'
  return 'bg-bg/60 text-text-muted'
}
</script>

<template>
  <div
    v-if="visible && activeFlow"
    class="shrink-0 border-t border-border bg-surface"
    data-testid="flow-run-bar"
    :data-flow-id="activeFlow.id"
    role="region"
    :aria-label="t('workspace.runBar.aria')"
  >
    <div class="flex items-center gap-3 border-b border-border px-3 py-1.5 text-xs">
      <PhStop v-if="flowStatus === 'cancelling'" :size="14" class="shrink-0 text-text-dim" weight="fill" />
      <PhSpinner v-else :size="14" class="shrink-0 animate-spin text-running" />
      <span class="font-semibold">{{ t('workspace.runStatus') }}</span>
      <span class="tabular-nums font-bold">{{ progress.toFixed(0) }}%</span>
      <span class="text-text-muted">
        {{ t('workspace.runBar.counts', counts) }}
      </span>
      <div class="h-1.5 min-w-[6rem] flex-1 overflow-hidden rounded-sm bg-bg">
        <div class="h-full bg-accent-strong transition-all duration-300" :style="{ width: `${progress}%` }" />
      </div>
    </div>

    <div class="max-h-36 overflow-auto" data-testid="flow-run-bar-files">
      <div
        v-for="file in files"
        :key="file.name"
        class="flex items-center gap-2 px-3 py-1 text-sm"
        :class="rowClass(file)"
        :data-testid="`flow-run-file-${file.name}`"
      >
        <PhSpinner
          v-if="file.status === 'transferring' || file.status === 'checking'"
          :size="12"
          class="shrink-0 animate-spin"
        />
        <PhCheckCircle
          v-else-if="file.status === 'completed' || file.status === 'checked'"
          :size="12"
          class="shrink-0"
          weight="fill"
        />
        <PhWarningCircle v-else-if="file.status === 'failed'" :size="12" class="shrink-0" weight="fill" />
        <PhFile v-else :size="12" class="shrink-0" />
        <span class="min-w-0 flex-1 truncate" :title="file.name">{{ fileBaseName(file.name) }}</span>
        <span class="shrink-0 text-[11px] font-semibold uppercase tracking-wide">{{ statusLabel(file.status) }}</span>
        <span v-if="file.status === 'transferring'" class="shrink-0 tabular-nums">{{ file.progress.toFixed(0) }}%</span>
      </div>
      <p v-if="!files.length" class="px-3 py-2 text-xs text-text-muted">{{ t('workspace.runBar.empty') }}</p>
    </div>
  </div>
</template>
