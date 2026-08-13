<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { PhPlay, PhStop, PhFloppyDisk } from '@phosphor-icons/vue'
import { storeToRefs } from 'pinia'
import { useCanvasStore } from '@/stores/canvas'
import { useFlowsStore } from '@/stores/flows'
import { useRemotesStore } from '@/stores/remotes'
import { useToast } from '@/composables/useToast'
import CronField from '@/components/forms/CronField.vue'
import RemotePathField from '@/components/forms/RemotePathField.vue'
import AppCheckbox from '@/components/ui/Checkbox.vue'
import OperationSettingsPanel from '@/components/flows/OperationSettingsPanel.vue'
import WorkspaceRemotesSection from '@/components/workspace/WorkspaceRemotesSection.vue'
import { composeOp, parseComposed } from '@/lib/remotePath'
import { resolveOpAction } from '@/stores/flows'
import type { FlowAction } from '@/constants/forms'

const { t } = useI18n()
const canvas = useCanvasStore()
const flows = useFlowsStore()
const remotes = useRemotesStore()
const toast = useToast()
const { activeFlow, selection, graph, dirty } = storeToRefs(canvas)
const { runStatus } = storeToRefs(flows)

const selectedNode = computed(() => {
  const sel = selection.value
  if (sel?.kind !== 'node') return null
  return graph.value.nodes.find((n) => n.id === sel.id) ?? null
})

const selectedEdge = computed(() => {
  const sel = selection.value
  if (sel?.kind !== 'edge') return null
  return graph.value.edges.find((e) => e.id === sel.id) ?? null
})

const sourceNode = computed(() => {
  const e = selectedEdge.value
  if (!e) return null
  return graph.value.nodes.find((n) => n.id === e.source) ?? null
})

const targetNode = computed(() => {
  const e = selectedEdge.value
  if (!e) return null
  return graph.value.nodes.find((n) => n.id === e.target) ?? null
})

const running = computed(() => {
  const id = activeFlow.value?.id
  if (!id) return false
  const st = runStatus.value[id] || activeFlow.value?.status
  return st === 'running' || st === 'cancelling'
})

async function onRun() {
  if (!activeFlow.value) return
  if (!(activeFlow.value.operations ?? []).length) {
    toast.error(t('workspace.flowEmpty'))
    return
  }
  await canvas.persist()
  await flows.execute(activeFlow.value.id)
  toast.success(t('workspace.flowStarted', { name: activeFlow.value.name || t('workspace.untitledFlow') }))
}

async function onStop() {
  if (!activeFlow.value) return
  await flows.stop(activeFlow.value.id)
}

async function onSave() {
  if (!activeFlow.value) return
  await canvas.persist()
  toast.success(t('workspace.flowSaved', { name: activeFlow.value.name || t('workspace.untitledFlow') }))
}

function setNodePath(nodeId: string, composed: string) {
  const { remote, path } = parseComposed(composed)
  void canvas.updateNode(nodeId, { remote, path })
}

function setEdgeAction(action: FlowAction) {
  const e = selectedEdge.value
  if (!e) return
  void canvas.updateEdge(e.id, { action, sync_config: { ...(e.operation.sync_config as object), action } })
}

function setEdgeSync(cfg: Record<string, unknown>) {
  const e = selectedEdge.value
  if (!e) return
  void canvas.updateEdge(e.id, { sync_config: cfg, action: String(cfg.action ?? e.action) })
}

const WIDTH_KEY = 'gn-drive:inspector-width'
const DEFAULT_WIDTH = 512
const MIN_WIDTH = 320

const panelWidth = ref(readStoredWidth())
let dragging = false

function readStoredWidth(): number {
  try {
    const n = Number(localStorage.getItem(WIDTH_KEY))
    if (Number.isFinite(n)) return clampWidth(n)
  } catch {
    /* ignore */
  }
  return DEFAULT_WIDTH
}

function clampWidth(n: number): number {
  const max = Math.max(MIN_WIDTH, Math.min(880, window.innerWidth - 280))
  return Math.min(max, Math.max(MIN_WIDTH, Math.round(n)))
}

function persistWidth() {
  try {
    localStorage.setItem(WIDTH_KEY, String(panelWidth.value))
  } catch {
    /* ignore */
  }
}

function onResizeStart(ev: PointerEvent) {
  ev.preventDefault()
  dragging = true
  const startX = ev.clientX
  const startW = panelWidth.value
  const prevCursor = document.body.style.cursor
  const prevSelect = document.body.style.userSelect
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'

  const onMove = (e: PointerEvent) => {
    panelWidth.value = clampWidth(startW + (startX - e.clientX))
  }
  const onUp = () => {
    dragging = false
    document.body.style.cursor = prevCursor
    document.body.style.userSelect = prevSelect
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', onUp)
    persistWidth()
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', onUp)
}

onUnmounted(() => {
  if (dragging) {
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
  }
})
</script>

<template>
  <aside
    class="relative flex h-full max-w-full shrink-0 flex-col overflow-hidden border-l border-border bg-surface"
    :style="{ width: `${panelWidth}px` }"
    data-testid="canvas-inspector"
  >
    <button
      type="button"
      class="absolute inset-y-0 left-0 z-10 hidden w-1.5 cursor-col-resize border-0 bg-transparent p-0 hover:bg-accent-strong/35 md:block"
      data-testid="inspector-resize"
      :aria-label="t('workspace.canvas.resizePanel')"
      @pointerdown="onResizeStart"
    />
    <template v-if="activeFlow">
      <div class="border-b border-border px-3 py-2">
        <div class="flex flex-wrap items-center gap-1.5">
          <button
            v-if="running"
            type="button"
            class="btn-danger !px-2.5 !py-1"
            @click="onStop"
          >
            <PhStop :size="14" weight="bold" /> {{ t('workspace.stop') }}
          </button>
          <button
            v-else
            type="button"
            class="btn-primary !px-2.5 !py-1"
            data-testid="flows-run"
            :disabled="!(activeFlow.operations ?? []).length"
            @click="onRun"
          >
            <PhPlay :size="14" weight="bold" /> {{ t('workspace.run') }}
          </button>
          <button
            type="button"
            class="btn-primary !px-2.5 !py-1"
            :disabled="!dirty || running"
            :data-testid="`flows-save-bottom-${activeFlow.id}`"
            @click="onSave"
          >
            <PhFloppyDisk :size="14" weight="bold" /> {{ t('workspace.saveFlow') }}
          </button>
        </div>
      </div>

      <div class="min-h-0 flex-1 space-y-3 overflow-auto p-3">
        <label class="field-label">
          <span>{{ t('workspace.flowName') }}</span>
          <input
            class="field-input !font-sans"
            :value="activeFlow.name"
            data-testid="flows-name-inline"
            :disabled="running"
            @change="canvas.updateFlowMeta({ name: ($event.target as HTMLInputElement).value })"
          />
        </label>

        <div class="grid grid-cols-[1fr_auto] items-end gap-2">
          <label class="field-label !mb-0">
            <span>{{ t('flows.schedule') }}</span>
            <CronField
              :model-value="activeFlow.schedule_cron || activeFlow.cron_expr || ''"
              allow-none
              test-id="flows-cron"
              :disabled="running"
              @update:model-value="canvas.updateFlowMeta({ schedule_cron: $event, cron_expr: $event })"
            />
          </label>
          <AppCheckbox
            :model-value="!!(activeFlow.schedule_enabled ?? activeFlow.enabled)"
            :label="t('common.enabled')"
            :disabled="running"
            @update:model-value="canvas.updateFlowMeta({ schedule_enabled: $event, enabled: $event })"
          />
        </div>

        <template v-if="selectedNode">
          <h3 class="m-0 text-xs font-semibold uppercase tracking-wide text-text-muted">
            {{ t('workspace.canvas.nodeSettings') }}
          </h3>
          <WorkspaceRemotesSection />
          <label class="field-label">
            <span>{{ t('workspace.canvas.label') }}</span>
            <input
              class="field-input !font-sans"
              :value="selectedNode.label"
              :disabled="running"
              @change="canvas.updateNode(selectedNode.id, { label: ($event.target as HTMLInputElement).value })"
            />
          </label>
          <RemotePathField
            :model-value="composeOp(selectedNode.remote, selectedNode.path)"
            :remotes="remotes.items"
            :test-id="`canvas-node-path-${selectedNode.id}`"
            :disabled="running"
            @update:model-value="setNodePath(selectedNode.id, $event)"
          />
        </template>

        <template v-if="selectedEdge && sourceNode && targetNode">
          <h3 class="m-0 text-xs font-semibold uppercase tracking-wide text-text-muted">
            {{ t('workspace.canvas.edgeSettings') }}
          </h3>
          <label class="field-label">
            <span>{{ t('workspace.source') }}</span>
            <RemotePathField
              :model-value="composeOp(sourceNode.remote, sourceNode.path)"
              :remotes="remotes.items"
              :test-id="`op-src-${selectedEdge.id}`"
              :disabled="running"
              @update:model-value="setNodePath(sourceNode.id, $event)"
            />
          </label>
          <label class="field-label">
            <span>{{ t('workspace.target') }}</span>
            <RemotePathField
              :model-value="composeOp(targetNode.remote, targetNode.path)"
              :remotes="remotes.items"
              :test-id="`op-dst-${selectedEdge.id}`"
              :disabled="running"
              @update:model-value="setNodePath(targetNode.id, $event)"
            />
          </label>
          <OperationSettingsPanel
            :model-value="
              (selectedEdge.operation.sync_config && typeof selectedEdge.operation.sync_config === 'object'
                ? selectedEdge.operation.sync_config
                : { action: resolveOpAction(selectedEdge.operation) }) as Record<string, unknown>
            "
            :action="resolveOpAction(selectedEdge.operation)"
            :disabled="running"
            @update:model-value="setEdgeSync($event)"
            @update:action="setEdgeAction($event)"
          />
        </template>
      </div>
    </template>
    <p v-else class="p-4 text-sm text-text-muted">{{ t('workspace.canvas.pickFlow') }}</p>
  </aside>
</template>
