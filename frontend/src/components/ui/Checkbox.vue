<script setup lang="ts">
/**
 * Paper checkbox — 1px border, 6px radius, no offset-neo shadow.
 */
const model = defineModel<boolean>({ default: false })

defineProps<{
  label?: string
  disabled?: boolean
  /** Optional data-testid for e2e */
  testId?: string
}>()
</script>

<template>
  <label
    class="group inline-flex cursor-pointer select-none items-center gap-2.5 text-[13px] font-bold text-text"
    :class="disabled && 'cursor-not-allowed text-text-dim opacity-60'"
  >
    <input
      v-model="model"
      type="checkbox"
      class="peer sr-only"
      :disabled="disabled"
      :data-testid="testId"
    />
    <span
      class="relative inline-flex h-5 w-5 shrink-0 items-center justify-center rounded-[6px] border border-border text-text transition-colors duration-150"
      :class="[
        model ? 'bg-accent' : 'bg-bg group-hover:bg-surface-hover',
        disabled && 'border-dashed bg-surface-hover',
      ]"
      aria-hidden="true"
    >
      <svg
        v-if="model"
        class="h-3 w-3"
        viewBox="0 0 12 12"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="square"
        stroke-linejoin="miter"
      >
        <path d="M2 6.5 L4.5 9 L10 3" />
      </svg>
    </span>
    <span v-if="label" class="leading-tight">{{ label }}</span>
    <span v-else class="leading-tight"><slot /></span>
  </label>
</template>
