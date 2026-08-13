<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { PhPlus, PhTrash } from '@phosphor-icons/vue'
import { useFlowsStore } from '@/stores/flows'
import { useCanvasStore } from '@/stores/canvas'
import { useRemotesStore } from '@/stores/remotes'
import { useConfirmDialog } from '@/composables/useConfirmDialog'
import { useToast } from '@/composables/useToast'
import { cn } from '@/lib/cn'

const { t } = useI18n()
const flows = useFlowsStore()
const canvas = useCanvasStore()
const remotes = useRemotesStore()
const { confirmDialog } = useConfirmDialog()
const toast = useToast()

onMounted(() => {
  void remotes.load()
  void flows.load().then(async () => {
    await flows.pullRuntime()
    if (!canvas.activeFlowId && flows.items[0]) canvas.selectFlow(flows.items[0].id)
  })
})

async function onAddFlow() {
  const f = await canvas.addFlow(t('workspace.untitledFlow'))
  toast.success(t('workspace.flowAdded'))
  return f
}

async function onDeleteFlow(id: string, name: string) {
  if (flows.isFlowRunning(id)) {
    toast.error(t('workspace.busyLocked'))
    return
  }
  const ok = await confirmDialog({
    title: t('flows.deleteTitle'),
    message: t('flows.deleteMessage', { name: name || t('workspace.untitledFlow') }),
    confirmText: t('common.delete'),
    confirmVariant: 'danger',
  })
  if (!ok) return
  await flows.remove(id)
  if (canvas.activeFlowId === id) {
    canvas.selectFlow(flows.items[0]?.id ?? null)
  }
}


</script>

<template>
  <aside
    class="flex h-full w-[220px] shrink-0 flex-col overflow-hidden border-r border-border bg-surface"
    data-testid="workspace-flows"
  >
    <div class="flex items-center justify-between gap-2 border-b border-border px-3 py-2">
      <span class="text-xs font-semibold uppercase tracking-wide text-text-muted">{{
        t('workspace.flows')
      }}</span>
      <button type="button" class="btn-primary !px-2 !py-1 text-xs" data-testid="flows-add" @click="onAddFlow">
        <PhPlus :size="12" weight="bold" /> {{ t('flows.add') }}
      </button>
    </div>
    <div class="min-h-0 flex-1 overflow-auto p-2">
      <button
        v-for="(f, i) in flows.items"
        :key="f.id"
        type="button"
        :class="cn(
          'mb-1 flex w-full items-start gap-1 rounded-md border px-2 py-1.5 text-left',
          canvas.activeFlowId === f.id ? 'border-accent-strong bg-accent/20' : 'border-border bg-bg',
        )"
        :data-testid="`flow-card-${f.id}`"
        @click="canvas.selectFlow(f.id)"
      >
        <div class="min-w-0 flex-1">
          <div class="text-[10px] font-semibold uppercase text-text-dim">
            {{ t('workspace.flowLabel', { n: i + 1 }) }}
          </div>
          <div class="truncate text-sm font-semibold">{{ f.name || t('workspace.untitledFlow') }}</div>
        </div>
        <button
          type="button"
          class="btn-ghost !px-1 text-danger"
          :data-testid="`flows-delete-${f.id}`"
          :disabled="flows.isFlowRunning(f.id)"
          @click.stop="onDeleteFlow(f.id, f.name)"
        >
          <PhTrash :size="12" />
        </button>
      </button>
      <p v-if="!flows.items.length" class="px-1 py-3 text-xs text-text-muted">{{ t('workspace.canvas.emptyFlows') }}</p>
    </div>
  </aside>
</template>
