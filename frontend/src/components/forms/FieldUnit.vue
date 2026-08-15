<script setup lang="ts">
export interface FieldUnitOption {
  value: string
  label: string
}

withDefaults(
  defineProps<{
    label?: string
    modelValue?: string
    unitValue?: string
    units?: FieldUnitOption[]
    testId?: string
    disabled?: boolean
  }>(),
  {
    label: '',
    modelValue: '',
    unitValue: '',
    units: () => [],
    testId: '',
    disabled: false,
  },
)

const emit = defineEmits<{
  input: [value: string]
  change: [value: string]
  changeUnit: [value: string]
}>()
</script>

<template>
  <div
    class="field-label !mb-0"
    role="group"
    :aria-label="label || undefined"
    :class="disabled && 'text-text-dim'"
  >
    <span v-if="label">{{ label }}</span>
    <div class="flex gap-1">
      <input
        class="field-input min-w-0 flex-1"
        type="number"
        min="0"
        :value="modelValue"
        :disabled="disabled"
        :data-testid="testId || undefined"
        @input="emit('input', ($event.target as HTMLInputElement).value)"
        @change="emit('change', ($event.target as HTMLInputElement).value)"
      />
      <select
        class="field-input !w-16 shrink-0"
        :value="unitValue"
        :disabled="disabled"
        @change="emit('changeUnit', ($event.target as HTMLSelectElement).value)"
      >
        <option v-for="u in units" :key="u.value" :value="u.value">
          {{ u.label }}
        </option>
      </select>
    </div>
  </div>
</template>
