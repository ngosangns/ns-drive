<script setup lang="ts">
import { computed } from 'vue'
import { Handle, Position } from '@vue-flow/core'
import { PhHardDrives, PhFolder } from '@phosphor-icons/vue'

const props = defineProps<{
  id: string
  selected?: boolean
  data: {
    remote: string
    path: string
    label: string
    running?: boolean
    failed?: boolean
  }
}>()

const title = computed(() => props.data.label || props.data.remote || 'local')
const path = computed(() => props.data.path || '/')
</script>

<template>
  <div
    class="w-[192px] rounded-md border bg-surface px-3 py-2 shadow-[var(--shadow-paper)]"
    :class="[
      data.running ? 'cursor-not-allowed' : 'cursor-move',
      selected
        ? 'border-accent-strong'
        : data.failed
          ? 'border-danger'
          : data.running
            ? 'border-accent-strong'
            : 'border-border'
    ]"
    :data-testid="`canvas-node-${id}`"
  >
    <Handle
      type="target"
      :position="Position.Left"
      class="!h-2.5 !w-2.5 !cursor-crosshair !border !border-border !bg-bg"
    />
    <div class="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wide text-text-muted">
      <PhHardDrives v-if="data.remote" :size="12" weight="bold" />
      <PhFolder v-else :size="12" weight="bold" />
      <span class="truncate">{{ title }}</span>
    </div>
    <div class="mt-0.5 truncate font-mono text-[12px] text-text" :title="path">{{ path }}</div>
    <Handle
      type="source"
      :position="Position.Right"
      class="!h-2.5 !w-2.5 !cursor-crosshair !border !border-border !bg-bg"
    />
  </div>
</template>
