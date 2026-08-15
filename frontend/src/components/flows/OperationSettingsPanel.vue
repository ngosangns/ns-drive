<script setup lang="ts">
/**
 * Wails `operation-settings-panel` port: Performance / Filtering / Safety /
 * Comparison / Sync|Bisync options for a flow operation.
 * Schedule stays on the flow card (not per-op).
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PhPlus, PhX } from '@phosphor-icons/vue'
import { FLOW_ACTIONS, type FlowAction } from '@/constants/forms'
import {
  type SyncConfig,
  parseSyncConfig,
  serializeSyncConfig,
} from '@/lib/syncConfig'
import AppCheckbox from '@/components/ui/Checkbox.vue'
import CustomField from '@/components/forms/CustomField.vue'
import FieldUnit from '@/components/forms/FieldUnit.vue'

const props = withDefaults(
  defineProps<{
    /** Raw operation.sync_config */
    modelValue: Record<string, unknown> | null | undefined
    /** Column action fallback */
    action?: string
    disabled?: boolean
  }>(),
  { disabled: false },
)

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
  /** When action changes, parent should sync op.action column */
  'update:action': [action: FlowAction]
}>()

const { t } = useI18n()

const cfg = ref<SyncConfig>(parseSyncConfig(props.modelValue, props.action || 'push'))

// Path list drafts
const includeDrafts = ref<string[]>([])
const excludeDrafts = ref<string[]>([])

// Size/age split UI
const minSizeNum = ref('')
const minSizeUnit = ref('M')
const maxSizeNum = ref('')
const maxSizeUnit = ref('G')
const minAgeNum = ref('')
const minAgeUnit = ref('h')
const maxAgeNum = ref('')
const maxAgeUnit = ref('d')

const sizeUnits = [
  { value: 'k', label: 'KB' },
  { value: 'M', label: 'MB' },
  { value: 'G', label: 'GB' },
  { value: 'T', label: 'TB' },
]
const ageUnits = [
  { value: 's', label: 's' },
  { value: 'm', label: 'm' },
  { value: 'h', label: 'h' },
  { value: 'd', label: 'd' },
  { value: 'w', label: 'w' },
  { value: 'M', label: 'M' },
  { value: 'y', label: 'y' },
]

const conflictOptions = [
  { value: 'newer', labelKey: 'workspace.opSettings.conflictNewer' },
  { value: 'older', labelKey: 'workspace.opSettings.conflictOlder' },
  { value: 'larger', labelKey: 'workspace.opSettings.conflictLarger' },
  { value: 'smaller', labelKey: 'workspace.opSettings.conflictSmaller' },
  { value: 'path1', labelKey: 'workspace.opSettings.conflictSource' },
  { value: 'path2', labelKey: 'workspace.opSettings.conflictTarget' },
]
const conflictLoserOptions = [
  { value: 'delete', labelKey: 'workspace.opSettings.loserDelete' },
  { value: 'num', labelKey: 'workspace.opSettings.loserNum' },
  { value: 'pathname', labelKey: 'workspace.opSettings.loserPath' },
]
const deleteTimingOptions = [
  { value: '', labelKey: 'workspace.opSettings.deleteDefault' },
  { value: 'before', labelKey: 'workspace.opSettings.deleteBefore' },
  { value: 'during', labelKey: 'workspace.opSettings.deleteDuring' },
  { value: 'after', labelKey: 'workspace.opSettings.deleteAfter' },
]

const isBi = computed(
  () => cfg.value.action === 'bi' || cfg.value.action === 'bi-resync',
)
const isPush = computed(() => cfg.value.action === 'push')

const actionOptions = computed(() =>
  FLOW_ACTIONS.map((a) => ({ value: a, label: t(`workspace.actionOptions.${a}`) })),
)
const conflictOptionList = computed(() => [
  { value: '', label: t('common.select') },
  ...conflictOptions.map((o) => ({ value: o.value, label: t(o.labelKey) })),
])
const conflictLoserOptionList = computed(() => [
  { value: '', label: t('common.select') },
  ...conflictLoserOptions.map((o) => ({ value: o.value, label: t(o.labelKey) })),
])
const deleteTimingOptionList = computed(() =>
  deleteTimingOptions.map((o) => ({ value: o.value, label: t(o.labelKey) })),
)

function splitSizeAge(val: string | undefined): { n: string; u: string } {
  if (!val) return { n: '', u: 'M' }
  const m = /^([0-9.]+)\s*([a-zA-Z]+)?$/.exec(val.trim())
  if (!m) return { n: val, u: 'M' }
  return { n: m[1], u: m[2] || 'M' }
}

function joinSizeAge(n: string, u: string): string | undefined {
  const t = n.trim()
  if (!t || Number(t) === 0) return undefined
  return `${t}${u}`
}

function hydrateFromProps() {
  cfg.value = parseSyncConfig(props.modelValue, props.action || 'push')
  includeDrafts.value = [...(cfg.value.includedPaths ?? [])]
  excludeDrafts.value = [...(cfg.value.excludedPaths ?? [])]
  const minS = splitSizeAge(cfg.value.minSize)
  minSizeNum.value = minS.n
  minSizeUnit.value = minS.u === 'k' || minS.u === 'M' || minS.u === 'G' || minS.u === 'T' ? minS.u : 'M'
  const maxS = splitSizeAge(cfg.value.maxSize)
  maxSizeNum.value = maxS.n
  maxSizeUnit.value = maxS.u === 'k' || maxS.u === 'M' || maxS.u === 'G' || maxS.u === 'T' ? maxS.u : 'G'
  const minA = splitSizeAge(cfg.value.minAge)
  minAgeNum.value = minA.n
  minAgeUnit.value = minA.u || 'h'
  const maxA = splitSizeAge(cfg.value.maxAge)
  maxAgeNum.value = maxA.n
  maxAgeUnit.value = maxA.u || 'd'
}

watch(
  () => [props.modelValue, props.action] as const,
  () => hydrateFromProps(),
  { immediate: true, deep: true },
)

function emitConfig() {
  cfg.value.minSize = joinSizeAge(minSizeNum.value, minSizeUnit.value)
  cfg.value.maxSize = joinSizeAge(maxSizeNum.value, maxSizeUnit.value)
  cfg.value.minAge = joinSizeAge(minAgeNum.value, minAgeUnit.value)
  cfg.value.maxAge = joinSizeAge(maxAgeNum.value, maxAgeUnit.value)
  cfg.value.includedPaths = includeDrafts.value.map((s) => s.trim()).filter(Boolean)
  cfg.value.excludedPaths = excludeDrafts.value.map((s) => s.trim()).filter(Boolean)
  const serialized = serializeSyncConfig(cfg.value)
  emit('update:modelValue', serialized)
  emit('update:action', cfg.value.action)
}

function setAction(a: string) {
  cfg.value.action = a as FlowAction
  emitConfig()
}

function patchNum(key: keyof SyncConfig, raw: string) {
  const bag = cfg.value as unknown as Record<string, unknown>
  const t = raw.trim()
  if (t === '') {
    delete bag[key as string]
  } else {
    const n = Number(t)
    if (!Number.isNaN(n)) bag[key as string] = n
  }
  emitConfig()
}

function patchStr(key: keyof SyncConfig, raw: string) {
  const bag = cfg.value as unknown as Record<string, unknown>
  const t = raw.trim()
  if (t === '') delete bag[key as string]
  else bag[key as string] = t
  emitConfig()
}

function patchBool(key: keyof SyncConfig, v: boolean) {
  const bag = cfg.value as unknown as Record<string, unknown>
  bag[key as string] = v
  emitConfig()
}

function addInclude() {
  includeDrafts.value = [...includeDrafts.value, '']
}
function addExclude() {
  excludeDrafts.value = [...excludeDrafts.value, '']
}
function removeInclude(i: number) {
  includeDrafts.value = includeDrafts.value.filter((_, idx) => idx !== i)
  emitConfig()
}
function removeExclude(i: number) {
  excludeDrafts.value = excludeDrafts.value.filter((_, idx) => idx !== i)
  emitConfig()
}
</script>

<template>
  <div
    class="space-y-3 border-t border-border bg-bg p-3"
    :class="disabled && 'pointer-events-none'"
    :aria-disabled="disabled"
    data-testid="op-settings-panel"
  >
    <!-- Action + Dry run -->
    <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 sm:items-end">
      <CustomField
        kind="select"
        :model-value="cfg.action"
        :label="t('workspace.action')"
        :options="actionOptions"
        :disabled="disabled"
        :help="t(`workspace.actionHelp.${cfg.action}`)"
        test-id="op-settings-action"
        @change="setAction($event)"
      />
      <div class="flex items-end pb-1">
        <AppCheckbox
          :model-value="!!cfg.dryRun"
          :disabled="disabled"
          :label="t('workspace.dryRun')"
          test-id="op-settings-dry-run"
          @update:model-value="patchBool('dryRun', !!$event)"
        />
      </div>
    </div>

    <!-- Performance -->
    <section class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-info,#5ca8e8)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.performance') }}
      </header>
      <div class="space-y-3 p-3">
        <div class="grid grid-cols-2 gap-2">
          <CustomField
            :model-value="cfg.parallel ?? ''"
            :label="t('workspace.opSettings.parallel')"
            type="number"
            :min="0"
            placeholder="8"
            :disabled="disabled"
            @change="patchNum('parallel', $event)"
          />
          <CustomField
            :model-value="cfg.bandwidth ?? ''"
            :label="t('workspace.opSettings.bandwidth')"
            type="number"
            :min="0"
            placeholder="0"
            :disabled="disabled"
            @change="patchNum('bandwidth', $event)"
          />
        </div>
        <div class="border-t border-border pt-3">
          <div class="mb-2 text-xs font-bold uppercase tracking-wide text-text-muted">
            {{ t('workspace.opSettings.advanced') }}
          </div>
          <div class="grid grid-cols-2 gap-2">
            <CustomField
              :model-value="cfg.multiThreadStreams ?? ''"
              :label="t('workspace.opSettings.multiThread')"
              type="number"
              :disabled="disabled"
              @change="patchNum('multiThreadStreams', $event)"
            />
            <CustomField
              :model-value="cfg.bufferSize ?? ''"
              :label="t('workspace.opSettings.bufferSize')"
              placeholder="16M"
              :disabled="disabled"
              @change="patchStr('bufferSize', $event)"
            />
            <CustomField
              :model-value="cfg.retries ?? ''"
              :label="t('workspace.opSettings.retries')"
              type="number"
              :disabled="disabled"
              @change="patchNum('retries', $event)"
            />
            <CustomField
              :model-value="cfg.lowLevelRetries ?? ''"
              :label="t('workspace.opSettings.lowLevelRetries')"
              type="number"
              :disabled="disabled"
              @change="patchNum('lowLevelRetries', $event)"
            />
            <CustomField
              :model-value="cfg.maxDuration ?? ''"
              :label="t('workspace.opSettings.maxDuration')"
              placeholder="1h30m"
              :disabled="disabled"
              @change="patchStr('maxDuration', $event)"
            />
            <CustomField
              :model-value="cfg.retriesSleep ?? ''"
              :label="t('workspace.opSettings.retriesSleep')"
              placeholder="10s"
              :disabled="disabled"
              @change="patchStr('retriesSleep', $event)"
            />
            <CustomField
              :model-value="cfg.tpsLimit ?? ''"
              :label="t('workspace.opSettings.tpsLimit')"
              type="number"
              :disabled="disabled"
              @change="patchNum('tpsLimit', $event)"
            />
            <CustomField
              :model-value="cfg.connTimeout ?? ''"
              :label="t('workspace.opSettings.connTimeout')"
              placeholder="30s"
              :disabled="disabled"
              @change="patchStr('connTimeout', $event)"
            />
            <CustomField
              :model-value="cfg.ioTimeout ?? ''"
              :label="t('workspace.opSettings.ioTimeout')"
              placeholder="5m"
              :disabled="disabled"
              @change="patchStr('ioTimeout', $event)"
            />
            <CustomField
              :model-value="cfg.orderBy ?? ''"
              :label="t('workspace.opSettings.orderBy')"
              placeholder="size,desc"
              :disabled="disabled"
              @change="patchStr('orderBy', $event)"
            />
          </div>
          <div class="mt-2">
            <AppCheckbox
              :model-value="!!cfg.checkFirst"
              :disabled="disabled"
              :label="t('workspace.opSettings.checkFirst')"
              @update:model-value="patchBool('checkFirst', !!$event)"
            />
          </div>
        </div>
      </div>
    </section>

    <!-- Filtering -->
    <section class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-accent-strong,#6ee7df)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.filtering') }}
      </header>
      <div class="space-y-3 p-3">
        <div>
          <div class="mb-1 text-xs font-bold text-text-muted">{{ t('workspace.opSettings.includePaths') }}</div>
          <div
            v-for="(row, i) in includeDrafts"
            :key="`inc-${i}`"
            class="mb-1 flex items-center gap-2"
          >
            <CustomField
              class="min-w-0 flex-1"
              :model-value="includeDrafts[i]"
              placeholder="/path or *.ext"
              mono
              :disabled="disabled"
              @input="includeDrafts[i] = $event"
              @change="emitConfig"
            />
            <button type="button" class="btn-ghost !px-1" :disabled="disabled" @click="removeInclude(i)">
              <PhX :size="12" class="text-danger" />
            </button>
          </div>
          <button
            type="button"
            class="text-xs text-text-muted hover:text-text disabled:cursor-not-allowed disabled:text-text-dim disabled:opacity-60"
            :disabled="disabled"
            @click="addInclude"
          >
            <PhPlus :size="12" class="mr-0.5 inline" /> {{ t('workspace.opSettings.addPath') }}
          </button>
        </div>
        <div>
          <div class="mb-1 text-xs font-bold text-text-muted">{{ t('workspace.opSettings.excludePaths') }}</div>
          <div
            v-for="(row, i) in excludeDrafts"
            :key="`exc-${i}`"
            class="mb-1 flex items-center gap-2"
          >
            <CustomField
              class="min-w-0 flex-1"
              :model-value="excludeDrafts[i]"
              placeholder="*.tmp or node_modules/"
              mono
              :disabled="disabled"
              @input="excludeDrafts[i] = $event"
              @change="emitConfig"
            />
            <button type="button" class="btn-ghost !px-1" :disabled="disabled" @click="removeExclude(i)">
              <PhX :size="12" class="text-danger" />
            </button>
          </div>
          <button
            type="button"
            class="text-xs text-text-muted hover:text-text disabled:cursor-not-allowed disabled:text-text-dim disabled:opacity-60"
            :disabled="disabled"
            @click="addExclude"
          >
            <PhPlus :size="12" class="mr-0.5 inline" /> {{ t('workspace.opSettings.addPath') }}
          </button>
        </div>
        <div class="border-t border-border pt-3">
          <div class="mb-2 text-xs font-bold uppercase tracking-wide text-text-muted">
            {{ t('workspace.opSettings.advanced') }}
          </div>
          <div class="grid grid-cols-2 gap-2">
            <FieldUnit
              :label="t('workspace.opSettings.minSize')"
              :model-value="minSizeNum"
              :unit-value="minSizeUnit"
              :units="sizeUnits"
              :disabled="disabled"
              @input="minSizeNum = $event"
              @change="emitConfig"
              @change-unit="minSizeUnit = $event; emitConfig()"
            />
            <FieldUnit
              :label="t('workspace.opSettings.maxSize')"
              :model-value="maxSizeNum"
              :unit-value="maxSizeUnit"
              :units="sizeUnits"
              :disabled="disabled"
              @input="maxSizeNum = $event"
              @change="emitConfig"
              @change-unit="maxSizeUnit = $event; emitConfig()"
            />
            <FieldUnit
              :label="t('workspace.opSettings.maxAge')"
              :model-value="maxAgeNum"
              :unit-value="maxAgeUnit"
              :units="ageUnits"
              :disabled="disabled"
              @input="maxAgeNum = $event"
              @change="emitConfig"
              @change-unit="maxAgeUnit = $event; emitConfig()"
            />
            <FieldUnit
              :label="t('workspace.opSettings.minAge')"
              :model-value="minAgeNum"
              :unit-value="minAgeUnit"
              :units="ageUnits"
              :disabled="disabled"
              @input="minAgeNum = $event"
              @change="emitConfig"
              @change-unit="minAgeUnit = $event; emitConfig()"
            />
            <CustomField
              :model-value="cfg.maxDepth ?? ''"
              :label="t('workspace.opSettings.maxDepth')"
              type="number"
              :disabled="disabled"
              @change="patchNum('maxDepth', $event)"
            />
            <CustomField
              :model-value="cfg.excludeIfPresent ?? ''"
              :label="t('workspace.opSettings.excludeIfPresent')"
              placeholder=".nosync"
              :disabled="disabled"
              @change="patchStr('excludeIfPresent', $event)"
            />
          </div>
          <div class="mt-2 flex flex-wrap gap-4">
            <AppCheckbox
              :model-value="!!cfg.useRegex"
              :disabled="disabled"
              :label="t('workspace.opSettings.useRegex')"
              @update:model-value="patchBool('useRegex', !!$event)"
            />
            <AppCheckbox
              :model-value="!!cfg.deleteExcluded"
              :disabled="disabled"
              :label="t('workspace.opSettings.deleteExcluded')"
              @update:model-value="patchBool('deleteExcluded', !!$event)"
            />
          </div>
        </div>
      </div>
    </section>

    <!-- Safety -->
    <section class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-warning,#e7b866)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.safety') }}
      </header>
      <div class="space-y-3 p-3">
        <div class="grid grid-cols-2 gap-2">
          <CustomField
            :model-value="cfg.maxDelete ?? ''"
            :label="t('workspace.opSettings.maxDelete')"
            type="number"
            placeholder="100"
            :disabled="disabled"
            @change="patchNum('maxDelete', $event)"
          />
          <CustomField
            :model-value="cfg.maxTransfer ?? ''"
            :label="t('workspace.opSettings.maxTransfer')"
            placeholder="10G"
            :disabled="disabled"
            @change="patchStr('maxTransfer', $event)"
          />
          <CustomField
            :model-value="cfg.maxDeleteSize ?? ''"
            :label="t('workspace.opSettings.maxDeleteSize')"
            placeholder="1G"
            :disabled="disabled"
            @change="patchStr('maxDeleteSize', $event)"
          />
          <CustomField
            :model-value="cfg.suffix ?? ''"
            :label="t('workspace.opSettings.suffix')"
            placeholder=".bak"
            :disabled="disabled"
            @change="patchStr('suffix', $event)"
          />
          <CustomField
            class="sm:col-span-2"
            :model-value="cfg.backupPath ?? ''"
            :label="t('workspace.opSettings.backupPath')"
            :disabled="disabled"
            @change="patchStr('backupPath', $event)"
          />
        </div>
        <div class="flex flex-wrap gap-4">
          <AppCheckbox
            :model-value="!!cfg.immutable"
            :disabled="disabled"
            :label="t('workspace.opSettings.immutable')"
            @update:model-value="patchBool('immutable', !!$event)"
          />
          <AppCheckbox
            :model-value="!!cfg.suffixKeepExtension"
            :disabled="disabled"
            :label="t('workspace.opSettings.suffixKeepExt')"
            @update:model-value="patchBool('suffixKeepExtension', !!$event)"
          />
        </div>
      </div>
    </section>

    <!-- Comparison -->
    <section class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-success,#5cc786)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.comparison') }}
      </header>
      <div class="flex flex-wrap gap-4 p-3">
        <AppCheckbox
          :model-value="!!cfg.sizeOnly"
          :disabled="disabled"
          :label="t('workspace.opSettings.sizeOnly')"
          @update:model-value="patchBool('sizeOnly', !!$event)"
        />
        <AppCheckbox
          :model-value="!!cfg.updateMode"
          :disabled="disabled"
          :label="t('workspace.opSettings.updateMode')"
          @update:model-value="patchBool('updateMode', !!$event)"
        />
        <AppCheckbox
          :model-value="!!cfg.ignoreExisting"
          :disabled="disabled"
          :label="t('workspace.opSettings.ignoreExisting')"
          @update:model-value="patchBool('ignoreExisting', !!$event)"
        />
      </div>
    </section>

    <!-- Sync options (push) -->
    <section v-if="isPush" class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-info,#5ca8e8)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.syncOptions') }}
      </header>
      <div class="p-3">
        <CustomField
          kind="select"
          :model-value="cfg.deleteTiming ?? ''"
          :label="t('workspace.opSettings.deleteTiming')"
          :options="deleteTimingOptionList"
          :disabled="disabled"
          @change="patchStr('deleteTiming', $event)"
        />
      </div>
    </section>

    <!-- Bisync options -->
    <section v-if="isBi" class="rounded-md border border-border">
      <header
        class="flex items-center gap-2 border-b border-border border-l-4 border-l-[var(--color-danger,#eb6b6f)] bg-bg-secondary px-3 py-2 text-xs font-semibold uppercase tracking-wide"
      >
        {{ t('workspace.opSettings.bisyncOptions') }}
      </header>
      <div class="space-y-3 p-3">
        <CustomField
          kind="select"
          :model-value="cfg.conflictResolution ?? ''"
          :label="t('workspace.opSettings.conflictResolution')"
          :options="conflictOptionList"
          :disabled="disabled"
          @change="patchStr('conflictResolution', $event)"
        />
        <div class="border-t border-border pt-3">
          <div class="mb-2 text-xs font-bold uppercase tracking-wide text-text-muted">
            {{ t('workspace.opSettings.advanced') }}
          </div>
          <div class="grid grid-cols-2 gap-2">
            <CustomField
              kind="select"
              :model-value="cfg.conflictLoser ?? ''"
              :label="t('workspace.opSettings.conflictLoser')"
              :options="conflictLoserOptionList"
              :disabled="disabled"
              @change="patchStr('conflictLoser', $event)"
            />
            <CustomField
              :model-value="cfg.conflictSuffix ?? ''"
              :label="t('workspace.opSettings.conflictSuffix')"
              :disabled="disabled"
              @change="patchStr('conflictSuffix', $event)"
            />
            <CustomField
              :model-value="cfg.maxLock ?? ''"
              :label="t('workspace.opSettings.maxLock')"
              placeholder="15m"
              :disabled="disabled"
              @change="patchStr('maxLock', $event)"
            />
          </div>
          <div class="mt-2 flex flex-wrap gap-4">
            <AppCheckbox
              :model-value="!!cfg.resilient"
              :disabled="disabled"
              :label="t('workspace.opSettings.resilient')"
              @update:model-value="patchBool('resilient', !!$event)"
            />
            <AppCheckbox
              :model-value="!!cfg.checkAccess"
              :disabled="disabled"
              :label="t('workspace.opSettings.checkAccess')"
              @update:model-value="patchBool('checkAccess', !!$event)"
            />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
