<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  PhPlus,
  PhCloud,
  PhTrash,
  PhCheckCircle,
  PhXCircle,
  PhSpinner,
} from '@phosphor-icons/vue'
import { useRemotesStore } from '@/stores/remotes'
import { useFlowsStore } from '@/stores/flows'
import RemoteTypeSelect from '@/components/forms/RemoteTypeSelect.vue'
import { useConfirmDialog } from '@/composables/useConfirmDialog'
import { useToast } from '@/composables/useToast'

defineOptions({ name: 'WorkspaceRemotesSection' })

defineProps<{
  eventsConnected: boolean
}>()

const { t } = useI18n()
const remotes = useRemotesStore()
const flows = useFlowsStore()
const { confirmDialog } = useConfirmDialog()
const toast = useToast()

const showRemoteForm = ref(false)
const remoteName = ref('')
const remoteType = ref('local')
type RemoteTestState = { status: 'loading' } | { status: 'ok' } | { status: 'error'; error?: string }
const remoteTest = ref<Record<string, RemoteTestState>>({})

const anyRunning = computed(() => flows.runningFlowIds.size > 0)

async function submitRemote() {
  if (!remoteName.value.trim()) return
  try {
    await remotes.add(remoteName.value.trim(), remoteType.value.trim())
    showRemoteForm.value = false
    remoteName.value = ''
    remoteType.value = 'local'
    toast.success(t('workspace.remoteAdded'))
  } catch {
    /* store surfaces error */
  }
}

async function testRemote(name: string) {
  remoteTest.value = { ...remoteTest.value, [name]: { status: 'loading' } }
  const r = await remotes.test(name)
  remoteTest.value = {
    ...remoteTest.value,
    [name]: r.ok ? { status: 'ok' } : { status: 'error', error: r.error },
  }
}

async function deleteRemote(name: string) {
  if (anyRunning.value) {
    toast.error(t('workspace.busyLocked'))
    return
  }
  const ok = await confirmDialog({
    title: t('remotes.deleteTitle'),
    message: t('remotes.deleteMessage', { name }),
    confirmText: t('common.delete'),
    confirmVariant: 'danger',
  })
  if (!ok) return
  await remotes.remove(name)
}
</script>

<template>
  <section class="shrink-0 border-b-2 border-border bg-bg py-3" data-testid="workspace-remotes">
    <div class="page-content-wide">
      <div class="mb-2 flex items-center justify-between gap-2">
        <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-wide text-text-muted">
          <PhCloud :size="14" weight="bold" />
          <span>{{ t('workspace.remotes') }}</span>
          <span class="badge">{{ remotes.items.length }}</span>
          <span
            class="font-mono text-[10px] uppercase"
            :class="eventsConnected ? 'text-success' : 'text-text-dim'"
          >
            {{ eventsConnected ? t('workspace.live') : t('workspace.polling') }}
          </span>
        </div>
        <button
          type="button"
          class="btn-secondary !px-2 !py-1 text-xs"
          data-testid="remotes-add"
          :disabled="anyRunning"
          @click="showRemoteForm = !showRemoteForm"
        >
          <PhPlus :size="14" weight="bold" /> {{ t('remotes.add') }}
        </button>
      </div>
      <div v-if="showRemoteForm" class="neo-inset mb-3 p-3" data-testid="remotes-add-form">
        <form class="grid grid-cols-1 gap-2 sm:grid-cols-[1fr_1fr_auto]" @submit.prevent="submitRemote">
          <label class="field-label">
            <span>{{ t('common.name') }}</span>
            <input v-model="remoteName" required class="field-input" data-testid="remotes-name" />
          </label>
          <label class="field-label">
            <span>{{ t('common.type') }}</span>
            <RemoteTypeSelect v-model="remoteType" test-id="remotes-type" />
          </label>
          <div class="flex items-end">
            <button type="submit" class="btn-primary" data-testid="remotes-submit">{{ t('common.save') }}</button>
          </div>
        </form>
      </div>
      <div v-if="remotes.items.length" class="flex flex-wrap gap-2">
        <div
          v-for="r in remotes.items"
          :key="r.name"
          class="neo-inset flex items-center gap-2 px-2.5 py-1.5"
          :data-testid="`remote-chip-${r.name}`"
        >
          <span class="font-bold text-sm">{{ r.name }}</span>
          <span class="text-[11px] text-text-dim">{{ r.type }}</span>
          <button type="button" class="btn-ghost !px-1 !text-[11px]" @click="testRemote(r.name)">
            <PhSpinner v-if="remoteTest[r.name]?.status === 'loading'" :size="12" class="animate-spin" />
            <PhCheckCircle v-else-if="remoteTest[r.name]?.status === 'ok'" :size="12" class="text-success" weight="fill" />
            <PhXCircle v-else-if="remoteTest[r.name]?.status === 'error'" :size="12" class="text-danger" weight="fill" />
            <span v-else>{{ t('common.test') }}</span>
          </button>
          <button type="button" class="btn-ghost !px-1 text-danger" :disabled="anyRunning" @click="deleteRemote(r.name)">
            <PhTrash :size="12" />
          </button>
        </div>
      </div>
      <p v-else class="text-sm text-text-muted">{{ t('workspace.noRemotes') }}</p>
    </div>
  </section>
</template>
