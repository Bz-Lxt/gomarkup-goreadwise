<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import cytoscape from 'cytoscape'
import fcose from 'cytoscape-fcose'
import { api } from '../api'
import { useApp } from '../stores/app'

cytoscape.use(fcose)
const app = useApp()
const host = ref<HTMLDivElement | null>(null)
const filter = ref('')
const emit = defineEmits<{ edit: [id: string, title: string, dangling: boolean] }>()
let cy: cytoscape.Core | null = null

const palette = ['#d4a053', '#6bb3b0', '#c9846a', '#8aa2d4', '#b88bcf', '#7fbe7a']

function colorFor(tags: string[]) {
  const top = (tags[0] || 'misc').split('/')[0]
  let h = 0
  for (const c of top) h = (h * 33 + c.charCodeAt(0)) % palette.length
  return palette[h]
}

function render() {
  if (!host.value) return
  cy?.destroy()
  const elements = [
    ...app.graph.nodes.map((n) => ({
      data: { id: n.id, title: n.title, dangling: n.dangling, degree: n.degree, tags: n.tags, card_id: n.card_id },
      position: n.x != null && n.y != null ? { x: n.x, y: n.y } : undefined,
    })),
    ...app.graph.edges.map((e) => ({ data: { id: e.id, source: e.source, target: e.target, dangling: e.dangling } })),
  ]
  cy = cytoscape({
    container: host.value,
    elements,
    style: [
      {
        selector: 'node',
        style: {
          label: 'data(title)',
          color: '#e8dcc8',
          'font-size': 10,
          'font-family': 'IBM Plex Sans',
          'background-color': (ele: cytoscape.NodeSingular) => colorFor(ele.data('tags') || []),
          width: (ele: cytoscape.NodeSingular) => 18 + Math.min(28, Number(ele.data('degree') || 0) * 3),
          height: (ele: cytoscape.NodeSingular) => 18 + Math.min(28, Number(ele.data('degree') || 0) * 3),
          'text-outline-color': '#070b14',
          'text-outline-width': 2,
        },
      },
      {
        selector: 'node[dangling = true]',
        style: { 'background-opacity': 0.35, 'border-width': 1, 'border-style': 'dashed', 'border-color': '#d36b6b' },
      },
      {
        selector: 'edge',
        style: {
          width: 1.2,
          'line-color': '#3d4a66',
          'target-arrow-color': '#3d4a66',
          'target-arrow-shape': 'triangle',
          'curve-style': 'bezier',
        },
      },
      {
        selector: 'edge[dangling = true]',
        style: { 'line-style': 'dashed', 'line-color': '#d36b6b', 'target-arrow-color': '#d36b6b' },
      },
    ],
    layout: { name: 'fcose', animate: app.graph.nodes.length < 300, quality: 'default', randomize: true },
  })
  cy.on('tap', 'node', (ev) => {
    const d = ev.target.data()
    emit('edit', d.card_id || d.id, d.title, !!d.dangling)
  })
  cy.on('dragfree', 'node', async () => {
    if (!cy) return
    const positions = cy.nodes().map((n) => {
      const p = n.position()
      return { id: n.id(), x: p.x, y: p.y }
    }).filter((p) => !p.id.startsWith('ghost:'))
    try { await api.savePositions(positions) } catch { /* ignore persist failure */ }
  })
  applyFilter()
}

function applyFilter() {
  if (!cy) return
  const q = filter.value.trim().toLowerCase()
  cy.nodes().forEach((n) => {
    const hit = !q || String(n.data('title')).toLowerCase().includes(q) || (n.data('tags') || []).some((t: string) => t.includes(q))
    n.style('opacity', hit ? 1 : 0.12)
  })
}

watch(() => app.graph, render, { deep: true })
watch(filter, applyFilter)
onMounted(render)
onBeforeUnmount(() => cy?.destroy())
</script>

<template>
  <div class="relative h-full min-h-[360px]">
    <div class="absolute left-4 top-4 z-10 flex gap-2">
      <input
        v-model="filter"
        class="rounded-lg border border-[var(--line)] bg-[var(--ink-2)]/90 px-3 py-1.5 text-sm outline-none"
        placeholder="筛选节点 / 标签"
      />
      <span class="self-center text-xs text-[var(--mist)]">{{ app.graph.nodes.length }} 节点 · {{ app.graph.edges.length }} 边</span>
    </div>
    <div ref="host" class="h-full w-full" data-testid="graph-canvas" />
  </div>
</template>
