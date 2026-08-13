<script setup lang="ts">
/**
 * Workspace: flow list + canvas + inspector. Remotes are managed in the node panel.
 */
import FlowRail from '@/components/canvas/FlowRail.vue'
import FlowCanvas from '@/components/canvas/FlowCanvas.vue'
import CanvasInspector from '@/components/canvas/CanvasInspector.vue'
import FlowRunBottomBar from '@/components/canvas/FlowRunBottomBar.vue'

defineOptions({ name: 'WorkspacePage' })
</script>

<template>
  <div class="flex h-full min-h-0 flex-col overflow-hidden" data-testid="page-workspace">
    <div class="flex min-h-0 flex-1 flex-col overflow-hidden md:flex-row">
      <div class="flex min-h-0 min-w-0 flex-1">
        <FlowRail />
        <FlowCanvas />
      </div>
      <CanvasInspector class="max-h-[46%] md:max-h-none" />
    </div>
    <FlowRunBottomBar />
  </div>
</template>
