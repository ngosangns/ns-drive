<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { PhKey, PhLock } from '@phosphor-icons/vue'
import { useAuthStore } from '@/stores/auth'
import { useApi } from '@/composables/useApi'
import { useConfirmDialog } from '@/composables/useConfirmDialog'
import AppAlert from '@/components/ui/Alert.vue'

const { t } = useI18n()
const auth = useAuthStore()
const api = useApi()
const router = useRouter()
const { confirmDialog } = useConfirmDialog()

const settings = ref<Record<string, string>>({})
const newPwd = ref('')
const oldPwd = ref('')
const removePwd = ref('')
const msg = ref<{ kind: 'ok' | 'err'; text: string } | null>(null)

onMounted(async () => {
  settings.value = (await api.get<Record<string, string>>('/api/v1/settings')) ?? {}
})

async function changePassword() {
  if (newPwd.value.length < 4) {
    msg.value = { kind: 'err', text: t('settings.pwdTooShort') }
    return
  }
  try {
    await api.post('/api/v1/auth/change-password', {
      old_password: oldPwd.value,
      new_password: newPwd.value,
    })
    auth.unlocked = false
    msg.value = { kind: 'ok', text: t('settings.pwdChanged') }
    newPwd.value = ''
    oldPwd.value = ''
    await router.push({ name: 'unlock' })
  } catch (e: any) {
    msg.value = { kind: 'err', text: e?.message ?? 'change failed' }
  }
}

async function lockApp() {
  try {
    await auth.lock()
    await router.push({ name: 'unlock' })
  } catch {
    // error already in store
  }
}

async function removePassword() {
  if (!removePwd.value) return
  const ok = await confirmDialog({
    title: t('settings.removePassword'),
    message: t('settings.removePasswordConfirm'),
    confirmText: t('settings.removePassword'),
    confirmVariant: 'danger',
  })
  if (!ok) return
  try {
    await api.post('/api/v1/auth/remove-password', { password: removePwd.value })
    removePwd.value = ''
    msg.value = { kind: 'ok', text: t('settings.removePasswordDone') }
  } catch (e: any) {
    msg.value = { kind: 'err', text: e?.message ?? 'remove failed' }
  }
}
</script>

<template>
  <div class="h-full min-h-0 overflow-auto py-6">
  <div class="page-content" data-testid="page-settings">
    <header class="mb-5">
      <h1 class="page-title">{{ t('settings.title') }}</h1>
      <p class="page-sub">{{ t('settings.sub') }}</p>
    </header>

    <AppAlert
      v-if="msg"
      :type="msg.kind === 'ok' ? 'success' : 'error'"
      data-testid="settings-msg"
    >
      {{ msg.text }}
    </AppAlert>

    <section class="card mb-3 px-5 py-4.5">
      <h2 class="mb-3.5 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-text-muted">
        <PhKey :size="14" weight="bold" /> {{ t('settings.masterPassword') }}
      </h2>
      <div class="mb-3 grid grid-cols-1 gap-2.5 md:grid-cols-2">
        <label class="field-label">
          <span>{{ t('settings.currentPassword') }}</span>
          <input
            v-model="oldPwd"
            type="password"
            autocomplete="current-password"
            class="field-input"
            data-testid="settings-old-password"
          />
        </label>
        <label class="field-label">
          <span>{{ t('settings.newPassword') }}</span>
          <input
            v-model="newPwd"
            type="password"
            autocomplete="new-password"
            class="field-input"
            data-testid="settings-new-password"
          />
        </label>
      </div>
      <div class="flex justify-end">
        <button
          class="btn-primary"
          :disabled="!oldPwd || !newPwd"
          data-testid="settings-change-password"
          @click="changePassword"
        >
          {{ t('settings.changePassword') }}
        </button>
      </div>
    </section>

    <section class="card mb-3 px-5 py-4.5">
      <h2 class="mb-3.5 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-text-muted">
        <PhLock :size="14" weight="bold" /> {{ t('settings.removePassword') }}
      </h2>
      <p class="mb-3 text-xs text-text-dim">{{ t('settings.removePasswordHelp') }}</p>
      <div class="mb-3 grid grid-cols-1 gap-2.5 md:grid-cols-2">
        <label class="field-label">
          <span>{{ t('settings.currentPassword') }}</span>
          <input
            v-model="removePwd"
            type="password"
            autocomplete="current-password"
            class="field-input"
            data-testid="settings-remove-password"
          />
        </label>
      </div>
      <div class="flex justify-end">
        <button
          class="danger !px-3.5 !py-1.5"
          :disabled="!removePwd"
          data-testid="settings-remove-password-btn"
          @click="removePassword"
        >
          {{ t('settings.removePassword') }}
        </button>
      </div>
    </section>

    <section class="card mb-3 px-5 py-4.5">
      <h2 class="mb-2 flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wide text-text-muted">
        <PhLock :size="14" weight="bold" /> {{ t('settings.lockNow') }}
      </h2>
      <p class="mb-3 text-xs text-text-dim">{{ t('settings.lockHelp') }}</p>
      <div class="flex justify-end">
        <button class="danger !px-3.5 !py-1.5" data-testid="settings-lock" @click="lockApp">
          {{ t('settings.lockApp') }}
        </button>
      </div>
    </section>
  </div>
  </div>
</template>
