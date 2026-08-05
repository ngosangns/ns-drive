<script setup lang="ts">
/**
 * Workspace shell: remotes + flows. Boards are not a product surface.
 */
import { inject, ref, type Ref } from 'vue'
import WorkspaceRemotesSection from '@/components/workspace/WorkspaceRemotesSection.vue'
import WorkspaceFlowsSection from '@/components/workspace/WorkspaceFlowsSection.vue'

defineOptions({ name: 'WorkspacePage' })

const eventsConnected = inject<Ref<boolean>>('eventsConnected', ref(false))
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden" data-testid="page-workspace">
    <WorkspaceRemotesSection :events-connected="eventsConnected" />
    <WorkspaceFlowsSection />
  </div>
</template>
