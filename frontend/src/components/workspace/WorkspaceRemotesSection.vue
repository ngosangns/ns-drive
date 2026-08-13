<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
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
import { configFieldsFor, missingRequiredFields, toConfigKVs } from '@/lib/remoteConfig'

defineOptions({ name: 'WorkspaceRemotesSection' })



const { t } = useI18n()
const remotes = useRemotesStore()
const flows = useFlowsStore()
const { confirmDialog } = useConfirmDialog()
const toast = useToast()

const showRemoteForm = ref(false)
const remoteName = ref('')
const remoteType = ref('local')
const adding = ref(false)
const addError = ref<string | null>(null)
const configValues = ref<Record<string, string>>({})
const configFields = computed(() => configFieldsFor(remoteType.value))

watch(remoteType, () => {
  configValues.value = {}
  addError.value = null
})
type RemoteTestState = { status: 'loading' } | { status: 'ok' } | { status: 'error'; error?: string }
const remoteTest = ref<Record<string, RemoteTestState>>({})

const anyRunning = computed(() => flows.runningFlowIds.size > 0)

onMounted(() => {
  void remotes.load()
})

async function submitRemote() {
  if (!remoteName.value.trim() || adding.value) return
  const missing = missingRequiredFields(remoteType.value, configValues.value)
  if (missing.length) {
    addError.value = t('remotes.missingCreds')
    return
  }
  adding.value = true
  addError.value = null
  try {
    await remotes.add(remoteName.value.trim(), remoteType.value.trim(), toConfigKVs(configValues.value))
    showRemoteForm.value = false
    remoteName.value = ''
    remoteType.value = 'local'
    configValues.value = {}
    toast.success(t('workspace.remoteAdded'))
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : remotes.error || t('remotes.authFailed')
    addError.value = msg
    toast.error(t('remotes.authFailed'))
  } finally {
    adding.value = false
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
  <section class="space-y-2" data-testid="workspace-remotes">
    <div class="flex items-center justify-between gap-2">
      <div class="flex items-center gap-2 text-xs font-semibold uppercase tracking-wide text-text-muted">
        <PhCloud :size="14" weight="bold" />
        <span>{{ t('workspace.remotes') }}</span>
        <span class="badge">{{ remotes.items.length }}</span>
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
        <form class="grid grid-cols-1 gap-2" @submit.prevent="submitRemote">
          <label class="field-label">
            <span>{{ t('common.name') }}</span>
            <input
              v-model="remoteName"
              required
              class="field-input"
              :disabled="adding"
              data-testid="remotes-name"
            />
          </label>
          <label class="field-label">
            <span>{{ t('common.type') }}</span>
            <RemoteTypeSelect v-model="remoteType" test-id="remotes-type" :disabled="adding" />
          </label>
          <label v-for="f in configFields" :key="f.key" class="field-label">
            <span>{{ t(f.labelKey) }}</span>
            <input
              v-model="configValues[f.key]"
              :type="f.input === 'password' ? 'password' : f.input === 'url' ? 'url' : 'text'"
              :required="f.required"
              class="field-input"
              :disabled="adding"
              :autocomplete="f.input === 'password' ? 'new-password' : 'off'"
              :data-testid="`remotes-cfg-${f.key}`"
            />
          </label>
          <p v-if="addError" class="m-0 text-[11px] font-semibold text-danger" data-testid="remotes-add-error">
            {{ addError }}
          </p>
          <button
            type="submit"
            class="btn-primary justify-center"
            :disabled="adding || !remoteName.trim()"
            data-testid="remotes-submit"
          >
            {{ adding ? t('remotes.testing') : t('remotes.testAndAdd') }}
          </button>
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
  </section>
</template>
