<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: number
    title?: string
  }>(),
  {
    size: 32,
    title: undefined,
  },
)
</script>

<template>
  <svg
    class="brand-mark"
    :width="size"
    :height="size"
    viewBox="0 0 40 40"
    fill="none"
    role="img"
    :aria-label="title"
    :aria-hidden="title ? undefined : 'true'"
  >
    <rect width="40" height="40" rx="11" fill="var(--color-text)" />
    <path
      d="M11.5 15.5h12.25a5.25 5.25 0 0 1 0 10.5H18"
      stroke="var(--color-accent)"
      stroke-width="4.2"
      stroke-linecap="round"
    />
    <path
      d="M28.5 24.5H16.25a5.25 5.25 0 0 1 0-10.5H22"
      stroke="var(--color-bg)"
      stroke-width="4.2"
      stroke-linecap="round"
    />
    <circle cx="11.5" cy="15.5" r="2.15" fill="var(--color-info)" />
    <circle cx="28.5" cy="24.5" r="2.15" fill="var(--color-info)" />
  </svg>
</template>
