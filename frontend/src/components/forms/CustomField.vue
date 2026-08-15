<script setup lang="ts">
export interface CustomFieldOption {
  value: string
  label: string
}

withDefaults(
  defineProps<{
    kind?: 'text' | 'select'
    modelValue?: string | number | null
    label?: string
    help?: string
    placeholder?: string
    options?: CustomFieldOption[]
    type?: 'text' | 'number'
    min?: number
    testId?: string
    disabled?: boolean
    mono?: boolean
    compact?: boolean
  }>(),
  {
    kind: 'text',
    modelValue: '',
    label: '',
    help: '',
    placeholder: '',
    options: () => [],
    type: 'text',
    min: undefined,
    testId: '',
    disabled: false,
    mono: false,
    compact: false,
  },
)

const emit = defineEmits<{
  change: [value: string]
  input: [value: string]
}>()

function onChange(event: Event) {
  const target = event.target as HTMLInputElement | HTMLSelectElement
  emit('change', target.value)
}

function onInput(event: Event) {
  emit('input', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <label class="field-label !mb-0" :class="disabled && 'text-text-dim'">
    <span v-if="label">{{ label }}</span>
    <select
      v-if="kind === 'select'"
      class="field-input"
      :class="compact && '!w-16 shrink-0'"
      :value="String(modelValue ?? '')"
      :disabled="disabled"
      :data-testid="testId || undefined"
      @change="onChange"
    >
      <option v-for="o in options" :key="o.value" :value="o.value">
        {{ o.label }}
      </option>
    </select>
    <input
      v-else
      class="field-input"
      :class="mono && 'font-mono'"
      :type="type"
      :min="min"
      :placeholder="placeholder || undefined"
      :value="modelValue ?? ''"
      :disabled="disabled"
      :data-testid="testId || undefined"
      @input="onInput"
      @change="onChange"
    />
    <p v-if="help" class="m-0 mt-1 text-[11px] leading-relaxed text-text-muted">{{ help }}</p>
  </label>
</template>
