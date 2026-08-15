<script setup lang="ts">
/**
 * Workspace flows section: list/edit/run flow cards.
 * Remotes live in WorkspaceRemotesSection.
 */
import { computed, onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  PhPlus,
  PhPlay,
  PhStop,
  PhTrash,
  PhStack,
  PhClock,
  PhArrowRight,
  PhArrowsLeftRight,
  PhFloppyDisk,
  PhPencilSimple,
  PhCheck,
  PhX,
} from '@phosphor-icons/vue'
import { useRemotesStore } from '@/stores/remotes'
import {
  useFlowsStore,
  emptyFlow,
  emptyOperation,
  resolveOpAction,
  withSyncedAction,
} from '@/stores/flows'
import type { Flow, Operation } from '@/api/types'
import { normalizeFlowAction } from '@/constants/forms'
import RemotePathField from '@/components/forms/RemotePathField.vue'
import CronField from '@/components/forms/CronField.vue'
import AppCheckbox from '@/components/ui/Checkbox.vue'
import AppAlert from '@/components/ui/Alert.vue'
import FlowRunStatusPanel from '@/components/flows/FlowRunStatusPanel.vue'
import OperationSettingsPanel from '@/components/flows/OperationSettingsPanel.vue'
import { parseSyncConfig, syncConfigSummaryChips } from '@/lib/syncConfig'
import { composeOp, displayPath as displayPathBase, parseComposed } from '@/lib/remotePath'
import { useConfirmDialog } from '@/composables/useConfirmDialog'
import { useToast } from '@/composables/useToast'
import { storeToRefs } from 'pinia'

defineOptions({ name: 'WorkspaceFlowsSection' })

const { t } = useI18n()
const remotes = useRemotesStore()
const flows = useFlowsStore()
const { runStatus, lastError, opSyncStatus } = storeToRefs(flows)
const { confirmDialog } = useConfirmDialog()
const toast = useToast()

const editingNameId = ref<string | null>(null)
const editingNameDraft = ref('')
const editingFlowIds = ref(new Set<string>())
const loading = ref(true)
const dataLoaded = ref(false)
const dirtyFlowIds = ref(new Set<string>())
const savingFlowIds = ref(new Set<string>())
const anyRunning = computed(() => flows.runningFlowIds.size > 0)

function isFlowEditing(id: string): boolean {
  return editingFlowIds.value.has(id)
}

function enterEditFlow(id: string) {
  if (flows.isFlowRunning(id)) {
    toast.error(t('workspace.busyLocked'))
    return
  }
  const next = new Set(editingFlowIds.value)
  next.add(id)
  editingFlowIds.value = next
}

async function cancelEditFlow(id: string) {
  if (isFlowDirty(id)) {
    const ok = await confirmDialog({
      title: t('workspace.discardEditTitle'),
      message: t('workspace.discardEditMessage'),
      confirmText: t('workspace.discardEditConfirm'),
      confirmVariant: 'danger',
    })
    if (!ok) return
    await flows.load()
    clearFlowDirty(id)
  }
  const next = new Set(editingFlowIds.value)
  next.delete(id)
  editingFlowIds.value = next
  if (editingNameId.value === id) cancelEditName()
}

async function doneEditFlow(id: string) {
  if (isFlowDirty(id)) {
    const ok = await saveFlow(id, { quiet: false })
    if (!ok) return
  }
  const next = new Set(editingFlowIds.value)
  next.delete(id)
  editingFlowIds.value = next
  if (editingNameId.value === id) cancelEditName()
}

function canEditFlow(id: string): boolean {
  return isFlowEditing(id) && !flows.isFlowRunning(id)
}

function flowRuntimeStatus(f: Flow): string {
  return runStatus.value[f.id] || f.status || 'idle'
}

function showRunStatusPanel(f: Flow): boolean {
  const st = flowRuntimeStatus(f)
  if (st && st !== 'idle') return true
  return !!opSyncStatus.value[f.id]
}

function startEditName(f: Flow) {
  if (flows.isFlowRunning(f.id)) return
  if (!isFlowEditing(f.id)) enterEditFlow(f.id)
  editingNameId.value = f.id
  editingNameDraft.value = f.name || ''
}

function cancelEditName() {
  editingNameId.value = null
  editingNameDraft.value = ''
}

async function commitEditName(f: Flow) {
  const name = editingNameDraft.value.trim() || f.name
  editingNameId.value = null
  editingNameDraft.value = ''
  if (name === f.name) return
  updateFlowLocal(f.id, { name })
  await saveFlow(f.id, { quiet: true })
}

function flowStatusLabel(st: string): string {
  if (!st || st === 'idle') return ''
  return st.charAt(0).toUpperCase() + st.slice(1)
}

function flowCardBorderClass(f: Flow): string {
  const st = flowRuntimeStatus(f)
  if (st === 'running' || st === 'cancelling') return 'border-running'
  if (st === 'completed') return 'border-success'
  if (st === 'failed') return 'border-danger'
  if (st === 'cancelled') return 'border-border-muted'
  return ''
}

function isFlowDirty(id: string): boolean {
  return dirtyFlowIds.value.has(id)
}
function isFlowSaving(id: string): boolean {
  return savingFlowIds.value.has(id)
}
function markFlowDirty(id: string) {
  const next = new Set(dirtyFlowIds.value)
  next.add(id)
  dirtyFlowIds.value = next
}
function clearFlowDirty(id: string) {
  if (!dirtyFlowIds.value.has(id)) return
  const next = new Set(dirtyFlowIds.value)
  next.delete(id)
  dirtyFlowIds.value = next
}

onMounted(async () => {
  try {
    await loadAll()
    dataLoaded.value = true
  } finally {
    loading.value = false
  }
})
onActivated(() => {
  if (dataLoaded.value) void loadAll()
})
async function loadAll() {
  await Promise.all([remotes.load(), flows.load()])
}

async function addFlow() {
  if (anyRunning.value) {
    toast.error(t('workspace.busyLocked'))
    return
  }
  const f = emptyFlow()
  f.name = t('workspace.untitledFlow')
  f.operations = []
  try {
    await flows.save(f)
    clearFlowDirty(f.id)
    enterEditFlow(f.id)
    toast.success(t('workspace.flowAdded'))
  } catch (e: any) {
    toast.error(e?.message ?? 'save failed')
  }
}

function updateFlowLocal(id: string, patch: Partial<Flow>, opts?: { dirty?: boolean }) {
  if (!canEditFlow(id) && opts?.dirty !== false) {
    if (!('is_collapsed' in patch && Object.keys(patch).length === 1)) {
      return
    }
  }
  const idx = flows.items.findIndex((x) => x.id === id)
  if (idx < 0) return
  const next = { ...flows.items[idx], ...patch }
  flows.items[idx] = next
  if (opts?.dirty === false) return
  if ('is_collapsed' in patch && Object.keys(patch).length === 1) return
  markFlowDirty(id)
}

async function saveFlow(id: string, opts?: { quiet?: boolean }): Promise<boolean> {
  const f = flows.items.find((x) => x.id === id)
  if (!f) return false
  if (flows.isFlowRunning(id)) {
    toast.error(t('workspace.busyLocked'))
    return false
  }
  const nextSaving = new Set(savingFlowIds.value)
  nextSaving.add(id)
  savingFlowIds.value = nextSaving
  try {
    await flows.save(f)
    clearFlowDirty(id)
    if (!opts?.quiet) toast.success(t('workspace.flowSaved', { name: f.name }))
    return true
  } catch (e: any) {
    toast.error(e?.message ?? 'save failed')
    return false
  } finally {
    const done = new Set(savingFlowIds.value)
    done.delete(id)
    savingFlowIds.value = done
  }
}

function addOperation(flowId: string) {
  if (!canEditFlow(flowId)) return
  const f = flows.items.find((x) => x.id === flowId)
  if (!f) return
  updateFlowLocal(flowId, { operations: [...(f.operations ?? []), emptyOperation()] })
}

function removeOperation(flowId: string, opId: string) {
  if (!canEditFlow(flowId)) return
  const f = flows.items.find((x) => x.id === flowId)
  if (!f) return
  updateFlowLocal(flowId, {
    operations: (f.operations ?? []).filter((o) => o.id !== opId),
  })
}

function patchOperation(flowId: string, opId: string, patch: Partial<Operation>) {
  if (!canEditFlow(flowId)) return
  const f = flows.items.find((x) => x.id === flowId)
  if (!f) return
  updateFlowLocal(flowId, {
    operations: (f.operations ?? []).map((o) => {
      if (o.id !== opId) return o
      let next: Operation = { ...o, ...patch }
      if (patch.action !== undefined) {
        next = withSyncedAction(next, patch.action)
      } else if (patch.sync_config !== undefined) {
        next = withSyncedAction(next, resolveOpAction(next))
      }
      return next
    }),
  })
}

function setOpAction(flowId: string, op: Operation, action: string) {
  patchOperation(flowId, op.id, { action: normalizeFlowAction(action) })
}

function setOpSyncConfig(flowId: string, op: Operation, sc: Record<string, unknown>) {
  const action = normalizeFlowAction(
    typeof sc.action === 'string' ? sc.action : resolveOpAction(op),
  )
  patchOperation(flowId, op.id, {
    action,
    sync_config: { ...sc, action },
  })
}

function opSettingsChips(op: Operation): string[] {
  return syncConfigSummaryChips(parseSyncConfig(op.sync_config, resolveOpAction(op)))
}

function setOpSource(flowId: string, op: Operation, composed: string) {
  const p = parseComposed(composed)
  patchOperation(flowId, op.id, { source_remote: p.remote, source_path: p.path })
}

function setOpTarget(flowId: string, op: Operation, composed: string) {
  const p = parseComposed(composed)
  patchOperation(flowId, op.id, { target_remote: p.remote, target_path: p.path })
}

async function runFlow(id: string) {
  if (flows.isFlowRunning(id)) return
  const f = flows.items.find((x) => x.id === id)
  if (!f?.operations?.length) {
    toast.error(t('workspace.flowEmpty'))
    return
  }
  if (isFlowDirty(id)) {
    const ok = await saveFlow(id, { quiet: true })
    if (!ok) return
  }
  if (isFlowEditing(id)) {
    const next = new Set(editingFlowIds.value)
    next.delete(id)
    editingFlowIds.value = next
    if (editingNameId.value === id) cancelEditName()
  }
  if (f.is_collapsed) {
    updateFlowLocal(id, { is_collapsed: false }, { dirty: false })
  }
  try {
    await flows.execute(id)
    toast.success(t('workspace.flowStarted', { name: f.name }))
  } catch (e: any) {
    toast.error(e?.message ?? 'execute failed')
  }
}

async function stopFlow(id: string) {
  try {
    await flows.stop(id)
  } catch (e: any) {
    toast.error(e?.message ?? 'stop failed')
  }
}

async function deleteFlow(id: string, name: string) {
  if (flows.isFlowRunning(id)) {
    toast.error(t('workspace.busyLocked'))
    return
  }
  const ok = await confirmDialog({
    title: t('flows.deleteTitle'),
    message: t('flows.deleteMessage', { name }),
    confirmText: t('common.delete'),
    confirmVariant: 'danger',
  })
  if (!ok) return
  await flows.remove(id)
}

function opFlowIcon(action?: string): 'right' | 'both' {
  const a = normalizeFlowAction(action)
  if (a === 'bi' || a === 'bi-resync') return 'both'
  return 'right'
}

function opActionLabel(action?: string): string {
  return t(`workspace.actionOptions.${normalizeFlowAction(action)}`)
}

function displayPath(remote: string, path: string): string {
  return displayPathBase(remote, path, t('common.empty'))
}

function scheduleLabel(f: Flow): string {
  const cron = f.schedule_cron || f.cron_expr || ''
  if (!cron) return t('workspace.noSchedule')
  const on = !!(f.schedule_enabled ?? f.enabled)
  return on ? cron : t('workspace.scheduleOff', { cron })
}
</script>

<template>
  <div class="min-h-0 flex-1 overflow-auto py-4 md:py-5">
      <div class="page-content-wide space-y-5">
        <AppAlert v-if="remotes.error || flows.error" type="error">
          {{ remotes.error || flows.error }}
        </AppAlert>

        <!-- FLOWS (primary) -->
        <div data-testid="workspace-flows">
        <div class="mb-3 flex flex-wrap items-end justify-between gap-3">
          <div class="min-w-0 flex-1">
            <h2 class="m-0 flex items-center gap-2 text-lg font-bold">
              <PhStack :size="22" weight="bold" />
              {{ t('workspace.flows') }}
            </h2>
            <p class="m-0 mt-0.5 text-xs text-text-muted">{{ t('workspace.flowsHint') }}</p>
          </div>
          <button
            type="button"
            class="btn-primary shrink-0"
            data-testid="flows-add"
            :disabled="anyRunning"
            @click="addFlow"
          >
            <PhPlus :size="14" weight="bold" /> {{ t('flows.add') }}
          </button>
        </div>

        <!-- Flow cards -->
        <section
          v-for="(f, fi) in flows.items"
          :key="f.id"
          class="neo-card overflow-hidden"
          :class="flowCardBorderClass(f)"
          :data-testid="`flow-card-${f.id}`"
        >
          <!-- Header: typographic surface; actions wrap under the title on narrow widths -->
          <div class="flex flex-col gap-3 border-b border-border bg-surface p-3">
            <div class="flex min-w-0 flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div class="min-w-0 flex items-center gap-2">
              <template v-if="editingNameId === f.id">
                <input
                  v-model="editingNameDraft"
                  class="field-input min-w-0 flex-1 !h-auto py-1 text-base font-bold"
                  type="text"
                  :placeholder="t('workspace.flowLabel', { n: fi + 1 })"
                  data-testid="flows-name-inline"
                  autocomplete="off"
                  @keydown.enter.prevent="commitEditName(f)"
                  @keydown.escape.prevent="cancelEditName"
                />
                <button type="button" class="btn-primary !px-2 !py-1" @click="commitEditName(f)">
                  <PhCheck :size="14" weight="bold" />
                </button>
                <button type="button" class="btn-secondary !px-2 !py-1" @click="cancelEditName">
                  <PhX :size="14" weight="bold" />
                </button>
              </template>
              <template v-else>
                <div class="min-w-0">
                  <div class="text-[10px] font-bold uppercase tracking-wide text-text/70">
                    {{ t('workspace.flowLabel', { n: fi + 1 }) }}
                  </div>
                  <h2 class="m-0 truncate text-lg font-bold leading-tight">
                    {{ f.name || t('workspace.untitledFlow') }}
                  </h2>
                </div>
                <button
                  v-if="isFlowEditing(f.id)"
                  type="button"
                  class="btn-secondary !px-2 !py-1"
                  :disabled="flows.isFlowRunning(f.id)"
                  :title="t('workspace.editName')"
                  data-testid="flows-edit-name"
                  @click="startEditName(f)"
                >
                  <PhPencilSimple :size="14" weight="bold" />
                </button>
              </template>
              </div>

              <div class="flex flex-wrap items-center gap-2">
              <button
                v-if="flows.isFlowRunning(f.id)"
                type="button"
                class="btn-danger !px-2.5 !py-1"
                @click="stopFlow(f.id)"
              >
                <PhStop :size="14" weight="bold" /> {{ t('workspace.stop') }}
              </button>
              <button
                v-else
                type="button"
                class="btn-primary !px-2.5 !py-1"
                :disabled="!(f.operations ?? []).length || isFlowSaving(f.id)"
                data-testid="flows-run"
                @click="runFlow(f.id)"
              >
                <PhPlay :size="14" weight="bold" /> {{ t('workspace.run') }}
              </button>
              <template v-if="isFlowEditing(f.id)">
                <button
                  type="button"
                  class="btn-primary !px-2.5 !py-1"
                  :disabled="flows.isFlowRunning(f.id) || isFlowSaving(f.id)"
                  :data-testid="`flows-done-edit-${f.id}`"
                  @click="doneEditFlow(f.id)"
                >
                  <PhCheck :size="14" weight="bold" />
                  {{ isFlowDirty(f.id) ? t('workspace.saveAndDone') : t('workspace.doneEdit') }}
                </button>
                <button
                  type="button"
                  class="btn-secondary !px-2.5 !py-1"
                  :disabled="flows.isFlowRunning(f.id) || isFlowSaving(f.id)"
                  :data-testid="`flows-cancel-edit-${f.id}`"
                  @click="cancelEditFlow(f.id)"
                >
                  <PhX :size="14" weight="bold" /> {{ t('common.cancel') }}
                </button>
              </template>
              <button
                v-else
                type="button"
                class="btn-secondary !px-2.5 !py-1"
                :disabled="flows.isFlowRunning(f.id)"
                :data-testid="`flows-edit-${f.id}`"
                @click="enterEditFlow(f.id)"
              >
                <PhPencilSimple :size="14" weight="bold" /> {{ t('workspace.editFlow') }}
              </button>
              <button
                type="button"
                class="btn-secondary !px-2.5 !py-1"
                :disabled="flows.isFlowRunning(f.id)"
                :data-testid="`flows-delete-${f.id}`"
                @click="deleteFlow(f.id, f.name)"
              >
                <PhTrash :size="14" class="text-danger" /> {{ t('workspace.remove') }}
              </button>
              </div>
            </div>

            <!-- Meta: ops · cron · status -->
            <div class="flex flex-wrap items-center gap-2">
              <span class="badge">
                {{ (f.operations ?? []).length }}
                {{
                  (f.operations ?? []).length === 1
                    ? t('workspace.operation')
                    : t('workspace.operations')
                }}
              </span>
              <span
                v-if="(f.schedule_enabled ?? f.enabled) && (f.schedule_cron || f.cron_expr)"
                class="badge"
              >
                <PhClock :size="12" class="mr-0.5 inline" />
                {{ f.schedule_cron || f.cron_expr }}
              </span>
              <span
                v-if="isFlowDirty(f.id)"
                class="rounded-md border border-warning px-2 py-1 text-xs font-semibold text-warning"
                data-testid="flows-unsaved"
              >
                {{ t('workspace.unsaved') }}
              </span>
              <span
                v-if="flowRuntimeStatus(f) !== 'idle'"
                class="rounded-md border border-border px-2 py-1 text-xs font-semibold"
                :class="{
                  'bg-warning/20 text-running': flowRuntimeStatus(f) === 'running' || flowRuntimeStatus(f) === 'cancelling',
                  'bg-success/20 text-success': flowRuntimeStatus(f) === 'completed',
                  'bg-danger/20 text-danger': flowRuntimeStatus(f) === 'failed',
                  'bg-bg text-text-muted': flowRuntimeStatus(f) === 'cancelled',
                }"
              >
                {{ flowStatusLabel(flowRuntimeStatus(f)) }}
              </span>
            </div>
          </div>

          <!-- Wails: operation-logs-panel under header when running or has last sync -->
          <FlowRunStatusPanel
            v-if="showRunStatusPanel(f)"
            :flow="f"
            :flow-status="flowRuntimeStatus(f)"
            :sync-status="opSyncStatus[f.id] ?? null"
            :last-error="lastError[f.id] || f.last_error"
          />

          <!-- Content: view-only summary OR edit forms -->
          <div class="space-y-2 bg-bg p-3">
            <!-- —— VIEW ONLY —— -->
            <template v-if="!isFlowEditing(f.id)">
              <div
                class="flex flex-wrap items-center gap-2 text-xs"
                :data-testid="`flow-view-schedule-${f.id}`"
              >
                <span class="font-bold uppercase text-text-muted">{{ t('flows.schedule') }}</span>
                <span class="rounded-md border border-border bg-bg-secondary px-2 py-1 font-mono font-semibold">
                  <PhClock :size="12" class="mr-0.5 inline" />
                  {{ scheduleLabel(f) }}
                </span>
                <span
                  v-if="(f.schedule_enabled ?? f.enabled) && (f.schedule_cron || f.cron_expr)"
                  class="badge"
                >
                  {{ t('common.enabled') }}
                </span>
              </div>

              <template v-if="(f.operations ?? []).length">
                <template v-for="(op, oi) in f.operations ?? []" :key="op.id">
                  <div v-if="oi > 0" class="flex justify-center py-0.5 text-text-dim">
                    <PhArrowRight :size="16" class="rotate-90" weight="bold" />
                  </div>
                  <div
                    class="neo-op-card px-3 py-2.5"
                    :data-testid="`op-view-${op.id}`"
                  >
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-mono text-xs font-bold text-text-dim">#{{ oi + 1 }}</span>
                      <span class="rounded-md border border-border bg-bg px-2 py-0.5 text-[11px] font-semibold">
                        {{ opActionLabel(resolveOpAction(op)) }}
                      </span>
                      <span
                        v-for="chip in opSettingsChips(op)"
                        :key="chip"
                        class="rounded-md border border-border bg-bg px-2 py-0.5 text-[10px] font-semibold text-text-muted"
                      >
                        {{ chip }}
                      </span>
                      <span
                        v-if="op.status && op.status !== 'idle' && flowRuntimeStatus(f) !== 'idle'"
                        class="badge font-mono uppercase"
                        :class="{
                          'text-running': op.status === 'running' || op.status === 'cancelling',
                          'text-success': op.status === 'completed',
                          'text-danger': op.status === 'failed' || op.status === 'cancelled',
                        }"
                      >{{ flowStatusLabel(op.status) }}</span>
                    </div>
                    <div
                      class="mt-2 flex min-w-0 flex-col gap-1 font-mono text-[12px] sm:flex-row sm:items-center sm:gap-2"
                    >
                      <span
                        class="min-w-0 truncate font-medium text-text"
                        :title="displayPath(op.source_remote, op.source_path)"
                      >
                        {{ displayPath(op.source_remote, op.source_path) }}
                      </span>
                      <span class="shrink-0 text-text-dim" :title="opActionLabel(resolveOpAction(op))">
                        <PhArrowsLeftRight
                          v-if="opFlowIcon(resolveOpAction(op)) === 'both'"
                          :size="14"
                          weight="bold"
                          class="inline"
                        />
                        <PhArrowRight v-else :size="14" weight="bold" class="inline" />
                      </span>
                      <span
                        class="min-w-0 truncate font-medium text-text"
                        :title="displayPath(op.target_remote, op.target_path)"
                      >
                        {{ displayPath(op.target_remote, op.target_path) }}
                      </span>
                    </div>
                  </div>
                </template>
              </template>
              <p
                v-else
                class="m-0 rounded-md border border-dashed border-border-muted px-3 py-4 text-center text-sm text-text-muted"
              >
                {{ t('workspace.noOperations') }}
              </p>
            </template>

            <!-- —— EDIT MODE —— -->
            <template v-else>
              <div class="grid grid-cols-1 gap-2 md:grid-cols-[1fr_auto] md:items-end">
                <label class="field-label !mb-0">
                  <span>{{ t('flows.schedule') }}</span>
                  <CronField
                    :model-value="f.schedule_cron || f.cron_expr || ''"
                    :disabled="!canEditFlow(f.id)"
                    allow-none
                    test-id="flows-cron"
                    @update:model-value="updateFlowLocal(f.id, { schedule_cron: $event, cron_expr: $event })"
                  />
                </label>
                <AppCheckbox
                  :model-value="!!(f.schedule_enabled ?? f.enabled)"
                  :disabled="!canEditFlow(f.id)"
                  :label="t('common.enabled')"
                  @update:model-value="updateFlowLocal(f.id, { schedule_enabled: $event, enabled: $event })"
                />
              </div>

              <template v-for="(op, oi) in f.operations ?? []" :key="op.id">
                <div v-if="oi > 0" class="flex justify-center py-0.5 text-text-dim">
                  <PhArrowRight :size="16" class="rotate-90" weight="bold" />
                </div>
                <div class="neo-op-card overflow-hidden" :data-testid="`op-row-${op.id}`">
                  <div class="flex items-center gap-2 border-b border-border bg-bg-secondary px-3 py-2">
                    <span class="font-mono text-xs font-bold text-text-dim">#{{ oi + 1 }}</span>
                    <span class="rounded-md border border-border bg-bg px-2 py-0.5 font-mono text-[11px] font-semibold uppercase">
                      {{ opActionLabel(resolveOpAction(op)) }}
                    </span>
                    <span
                      v-if="op.status && op.status !== 'idle' && flowRuntimeStatus(f) !== 'idle'"
                      class="badge font-mono uppercase"
                      :class="{
                        'text-running': op.status === 'running' || op.status === 'cancelling',
                        'text-success': op.status === 'completed',
                        'text-danger': op.status === 'failed' || op.status === 'cancelled',
                      }"
                    >{{ flowStatusLabel(op.status) }}</span>
                    <button
                      type="button"
                      class="btn-secondary ml-auto !px-2 !py-1"
                      :disabled="!canEditFlow(f.id)"
                      @click="removeOperation(f.id, op.id)"
                    >
                      <PhTrash :size="12" class="text-danger" />
                    </button>
                  </div>
                  <div class="grid grid-cols-1 gap-3 p-3 md:grid-cols-2 md:items-start">
                    <label class="field-label min-w-0">
                      <span>{{ t('workspace.source') }}</span>
                      <RemotePathField
                        :model-value="composeOp(op.source_remote, op.source_path)"
                        :remotes="remotes.items"
                        :disabled="!canEditFlow(f.id)"
                        :test-id="`op-src-${op.id}`"
                        @update:model-value="setOpSource(f.id, op, $event)"
                      />
                    </label>
                    <label class="field-label min-w-0">
                      <span>{{ t('workspace.target') }}</span>
                      <RemotePathField
                        :model-value="composeOp(op.target_remote, op.target_path)"
                        :remotes="remotes.items"
                        :disabled="!canEditFlow(f.id)"
                        :test-id="`op-dst-${op.id}`"
                        @update:model-value="setOpTarget(f.id, op, $event)"
                      />
                    </label>
                  </div>
                  <!-- Wails operation-settings-panel (Performance / Filtering / Safety / …) -->
                  <OperationSettingsPanel
                    :model-value="
                      (op.sync_config && typeof op.sync_config === 'object'
                        ? op.sync_config
                        : { action: resolveOpAction(op) }) as Record<string, unknown>
                    "
                    :action="resolveOpAction(op)"
                    :disabled="!canEditFlow(f.id)"
                    @update:model-value="setOpSyncConfig(f.id, op, $event)"
                    @update:action="setOpAction(f.id, op, $event)"
                  />
                </div>
              </template>

              <div
                class="mt-1 flex cursor-pointer items-center justify-center gap-2 rounded-md border border-dashed border-border-muted bg-bg/50 p-3 text-sm font-medium text-text-muted transition-colors hover:border-accent-strong hover:bg-accent/20"
                role="button"
                tabindex="0"
                :class="!canEditFlow(f.id) && 'pointer-events-none opacity-50'"
                data-testid="flows-add-op"
                @click="addOperation(f.id)"
                @keydown.enter="addOperation(f.id)"
              >
                <PhPlus :size="14" /> {{ t('workspace.addOperation') }}
              </div>

              <div
                v-if="isFlowDirty(f.id)"
                class="flex flex-wrap items-center justify-between gap-2 border-t border-border pt-3"
              >
                <p class="m-0 text-xs text-text-muted">{{ t('workspace.saveHint') }}</p>
                <button
                  type="button"
                  class="btn-primary"
                  :disabled="!canEditFlow(f.id) || isFlowSaving(f.id)"
                  :data-testid="`flows-save-bottom-${f.id}`"
                  @click="saveFlow(f.id)"
                >
                  <PhFloppyDisk :size="16" weight="bold" />
                  {{ isFlowSaving(f.id) ? t('common.saving') : t('workspace.saveFlow') }}
                </button>
              </div>
            </template>
          </div>
        </section>

        <div
          v-if="!flows.items.length && !loading"
          class="neo-dashed flex cursor-pointer items-center justify-center gap-2 p-6"
          role="button"
          tabindex="0"
          @click="addFlow"
          @keydown.enter="addFlow"
        >
          <PhPlus :size="18" />
          <span class="font-medium text-text-muted">{{ t('flows.add') }}</span>
        </div>

        <p v-if="loading" class="text-center text-sm text-text-muted">{{ t('common.loading') }}…</p>
        </div>
      </div>
    </div>
  

</template>
