<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { api } from './api'
import { useApp } from './stores/app'
import Sidebar from './components/Sidebar.vue'
import EditorPane from './components/EditorPane.vue'
import PreviewPane from './components/PreviewPane.vue'
import BacklinkPane from './components/BacklinkPane.vue'
import GraphView from './components/GraphView.vue'
import QuickEdit from './components/QuickEdit.vue'
import ClipDialog from './components/ClipDialog.vue'
import ToastHost from './components/ToastHost.vue'
import ConfirmModal from './components/ConfirmModal.vue'

const app = useApp()
const body = ref('')
const title = ref('')
const clipOpen = ref(false)
const quick = ref({ open: false, id: '', title: '', dangling: false })
const mobileTab = ref<'edit' | 'preview' | 'links'>('edit')
let timer: number | undefined
let es: EventSource | null = null

const known = computed(() => new Set(Object.keys(app.knownTitles)))

onMounted(async () => {
  const saved = localStorage.getItem('gr-widths')
  if (saved) {
    try { app.widths = JSON.parse(saved) } catch { /* ignore */ }
  }
  await app.bootstrap()
  persistWidths()
  es = new EventSource('/api/v1/events')
  es.addEventListener('graph:invalidated', () => { app.refreshGraph() })
})
onUnmounted(() => { es?.close() })

watch(() => app.current, (c) => {
  title.value = c?.title || ''
  body.value = c?.body || ''
})

watch([title, body], () => {
  if (!app.current) return
  if (title.value === app.current.title && body.value === app.current.body) return
  app.dirty = true
  window.clearTimeout(timer)
  timer = window.setTimeout(() => app.save({ title: title.value, body: body.value }), 1500)
})

function persistWidths() {
  localStorage.setItem('gr-widths', JSON.stringify(app.widths))
}

async function del() {
  if (!app.current) return
  app.confirm = {
    title: '删除卡片',
    body: `确认软删除「${app.current.title}」？指向它的入边会变成悬空链接。`,
    action: async () => {
      await api.deleteCard(app.current!.id)
      app.current = null
      await app.bootstrap()
    },
  }
}

function onGraphEdit(id: string, t: string, dangling: boolean) {
  quick.value = { open: true, id, title: t, dangling }
}

void known
</script>

<template>
  <div class="starfield flex h-full min-h-0 flex-col">
    <header class="flex items-center justify-between border-b border-[var(--line)] bg-[var(--ink-2)]/80 px-4 py-2 backdrop-blur">
      <div class="flex items-center gap-3">
        <button class="rounded-lg border border-[var(--line)] px-2 py-1 text-xs md:hidden" @click="app.sidebarOpen = !app.sidebarOpen">目录</button>
        <span class="font-display text-lg text-[var(--gold)]">GoReadwise</span>
      </div>
      <div class="flex items-center gap-2 text-sm">
        <button class="rounded-lg px-3 py-1.5" :class="app.view==='editor' ? 'bg-white/10' : ''" @click="app.view='editor'">三栏编辑</button>
        <button class="rounded-lg px-3 py-1.5" :class="app.view==='graph' ? 'bg-white/10' : ''" data-testid="tab-graph" @click="app.view='graph'">知识星空</button>
        <span class="hidden text-xs text-[var(--mist)] sm:inline">{{ app.saving ? '保存中…' : app.dirty ? '未保存' : '已同步' }}</span>
      </div>
    </header>

    <div class="flex min-h-0 flex-1">
      <div
        class="shrink-0 overflow-hidden border-r border-[var(--line)] transition-all"
        :class="app.sidebarOpen ? 'w-[min(300px,88vw)]' : 'w-0'"
      >
        <Sidebar @clip="clipOpen = true" />
      </div>

      <main class="flex min-w-0 flex-1 flex-col">
        <div v-if="app.loading" class="flex flex-1 items-center justify-center text-[var(--mist)]">正在展开星图…</div>

        <template v-else-if="app.view==='graph'">
          <GraphView @edit="onGraphEdit" />
        </template>

        <template v-else-if="app.current">
          <div class="flex flex-wrap items-center gap-2 border-b border-[var(--line)] px-4 py-2">
            <input v-model="title" class="min-w-[200px] flex-1 bg-transparent font-display text-xl outline-none" data-testid="card-title" />
            <button class="rounded-lg border border-[var(--line)] px-2 py-1 text-xs" @click="app.save({ title, body })">保存</button>
            <button class="rounded-lg border border-[var(--rose)]/40 px-2 py-1 text-xs text-[var(--rose)]" @click="del">删除</button>
          </div>
          <div class="flex gap-1 border-b border-[var(--line)] px-3 py-1 md:hidden">
            <button v-for="t in (['edit','preview','links'] as const)" :key="t" class="rounded px-2 py-1 text-xs" :class="mobileTab===t ? 'bg-white/10' : ''" @click="mobileTab=t">
              {{ t === 'edit' ? '编辑' : t === 'preview' ? '预览' : '反链' }}
            </button>
          </div>
          <div class="flex min-h-0 flex-1 flex-col md:flex-row">
            <section class="min-h-0 min-w-0 flex-1 border-[var(--line)] md:border-r" :class="mobileTab==='edit' ? '' : 'hidden md:block'" :style="{ flexBasis: app.widths.left + '%' }">
              <EditorPane v-model="body" data-testid="editor" />
            </section>
            <section class="min-h-0 min-w-0 flex-1 border-[var(--line)] md:border-r" :class="mobileTab==='preview' ? '' : 'hidden md:block'" :style="{ flexBasis: app.widths.mid + '%' }">
              <PreviewPane :body="body" />
            </section>
            <section class="min-h-0 min-w-0" :class="mobileTab==='links' ? '' : 'hidden md:block'" :style="{ flexBasis: app.widths.right + '%', minWidth: '220px' }">
              <BacklinkPane />
            </section>
          </div>
        </template>

        <div v-else class="flex flex-1 flex-col items-center justify-center text-center">
          <p class="font-display text-3xl text-[var(--gold)]">空星图</p>
          <p class="mt-2 max-w-md text-sm text-[var(--mist)]">还没有打开卡片。从左侧选一张，或写下第一颗恒星。</p>
        </div>
      </main>
    </div>

    <ToastHost />
    <ConfirmModal />
    <ClipDialog :open="clipOpen" @close="clipOpen=false" />
    <QuickEdit
      :open="quick.open"
      :card-id="quick.id"
      :title="quick.title"
      :dangling="quick.dangling"
      @close="quick.open=false"
      @opened="(id) => app.open(id)"
    />
  </div>
</template>
