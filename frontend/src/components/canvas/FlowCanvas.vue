<script setup lang="ts">
import { computed, markRaw } from 'vue'
import {
  PanOnScrollMode,
  VueFlow,
  type Connection,
  type Edge,
  type EdgeTypesObject,
  type Node,
  type NodeDragEvent,
  type NodeTypesObject,
} from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import { storeToRefs } from 'pinia'
import { useCanvasStore } from '@/stores/canvas'
import { useFlowsStore } from '@/stores/flows'
import { useToast } from '@/composables/useToast'
import { isActiveRun } from '@/lib/runChrome'
import { useI18n } from 'vue-i18n'
import LocationNode from './LocationNode.vue'
import SyncEdge from './SyncEdge.vue'
import '@vue-flow/core/dist/style.css'
import '@vue-flow/core/dist/theme-default.css'

const { t } = useI18n()
const canvas = useCanvasStore()
const flows = useFlowsStore()
const toast = useToast()
const { graph, selection, activeFlow } = storeToRefs(canvas)

const nodeTypes = {
  location: markRaw(LocationNode),
} as unknown as NodeTypesObject

const edgeTypes = {
  sync: markRaw(SyncEdge),
} as unknown as EdgeTypesObject

const running = computed(() => {
  const id = activeFlow.value?.id
  if (!id) return false
  return isActiveRun(flows.flowStatusOf(id))
})

const vfNodes = computed<Node[]>(() =>
  graph.value.nodes.map((n) => ({
    id: n.id,
    type: 'location',
    position: { x: n.x, y: n.y },
    selected: selection.value?.kind === 'node' && selection.value.id === n.id,
    data: {
      remote: n.remote,
      path: n.path,
      label: n.label,
      failed: false,
      running: running.value,
    },
  })),
)

const vfEdges = computed<Edge[]>(() => {
  const flowId = activeFlow.value?.id
  const snap = flowId ? flows.opSyncStatus[flowId] : null
  return graph.value.edges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    type: 'sync',
    selected: selection.value?.kind === 'edge' && selection.value.id === e.id,
    data: {
      action: e.action,
      flowRunning: running.value,
      running: snap?.op_id === e.id && snap.status === 'running',
      syncStatus: snap?.op_id === e.id ? snap.status : undefined,
      progress: snap?.op_id === e.id ? snap.progress : undefined,
      filesTransferred: snap?.op_id === e.id ? snap.files_transferred : undefined,
      totalFiles: snap?.op_id === e.id ? snap.total_files : undefined,
      checks: snap?.op_id === e.id ? snap.checks : undefined,
      totalChecks: snap?.op_id === e.id ? snap.total_checks : undefined,
      stage: snap?.op_id === e.id ? snap.stage : undefined,
      stageDetail: snap?.op_id === e.id ? snap.stage_detail : undefined,
      transfers: snap?.op_id === e.id ? snap.transfers : undefined,
    },
  }))
})

function onConnect(c: Connection) {
  if (running.value) return
  if (!c.source || !c.target) return
  void canvas.connect(c.source, c.target).then((r) => {
    if (!r.ok && r.error === 'duplicate-edge') toast.error(t('workspace.canvas.duplicateEdge'))
    if (!r.ok && r.error === 'self-loop') toast.error(t('workspace.canvas.selfLoop'))
  })
}

function onNodeClick({ node }: { node: Node }) {
  canvas.select({ kind: 'node', id: node.id })
}

function onEdgeClick({ edge }: { edge: Edge }) {
  canvas.select({ kind: 'edge', id: edge.id })
}

function onPaneClick() {
  canvas.select({ kind: 'flow' })
}

function onNodeDragStop(ev: NodeDragEvent) {
  const nodes = ev.nodes?.length ? ev.nodes : ev.node ? [ev.node] : []
  void canvas.updatePositions(nodes.map((n) => ({ id: n.id, x: n.position.x, y: n.position.y })))
}

function onDelete() {
  if (running.value) return
  if (selection.value?.kind === 'node') void canvas.removeNode(selection.value.id)
  if (selection.value?.kind === 'edge') void canvas.removeEdge(selection.value.id)
}
</script>

<template>
  <div class="h-full min-h-0 min-w-0 flex-1 bg-bg" data-testid="flow-canvas" @keydown.delete.prevent="onDelete">
    <VueFlow
      :nodes="vfNodes"
      :edges="vfEdges"
      :node-types="nodeTypes"
      :edge-types="edgeTypes"
      fit-view-on-init
      :nodes-draggable="true"
      :nodes-connectable="!running"
      :edges-updatable="false"
      :default-viewport="graph.viewport"
      :zoom-on-scroll="false"
      pan-on-scroll
      :pan-on-scroll-mode="PanOnScrollMode.Free"
      class="h-full w-full"
      @connect="onConnect"
      @node-click="onNodeClick"
      @edge-click="onEdgeClick"
      @pane-click="onPaneClick"
      @node-drag-stop="onNodeDragStop"
    >
      <Background pattern-color="var(--color-border-muted)" :gap="18" />
      <Controls />
    </VueFlow>
  </div>
</template>
