import { defineStore } from 'pinia'
import { api } from '../api'
import type { Card, GraphPayload, Tag } from '../types'

export const useApp = defineStore('app', {
  state: () => ({
    cards: [] as Card[],
    total: 0,
    tags: [] as Tag[],
    current: null as Card | null,
    graph: { version: 0, nodes: [], edges: [] } as GraphPayload,
    query: '',
    tagFilter: '',
    view: 'editor' as 'editor' | 'graph',
    loading: false,
    saving: false,
    dirty: false,
    toasts: [] as { id: number; text: string; kind: 'ok' | 'err' }[],
    knownTitles: {} as Record<string, string>,
    sidebarOpen: true,
    widths: { left: 22, mid: 40, right: 20 },
    confirm: null as null | { title: string; body: string; action: () => Promise<void> },
  }),
  actions: {
    toast(text: string, kind: 'ok' | 'err' = 'ok') {
      const id = Date.now() + Math.random()
      this.toasts.push({ id, text, kind })
      setTimeout(() => {
        this.toasts = this.toasts.filter((t) => t.id !== id)
      }, 5000)
    },
    dismissToast(id: number) {
      this.toasts = this.toasts.filter((t) => t.id !== id)
    },
    async bootstrap() {
      this.loading = true
      try {
        await Promise.all([this.refreshList(), this.refreshTags(), this.refreshGraph()])
        if (!this.current && this.cards[0]) await this.open(this.cards[0].id)
      } catch (e) {
        this.toast((e as Error).message, 'err')
      } finally {
        this.loading = false
      }
    },
    async refreshList() {
      const { data, meta } = await api.listCards(this.query, this.tagFilter, 1)
      this.cards = data || []
      this.total = meta?.total || this.cards.length
      const next: Record<string, string> = {}
      for (const c of this.cards) next[c.title.toLowerCase()] = c.id
      this.knownTitles = next
    },
    async refreshTags() {
      const { data } = await api.tags()
      this.tags = data || []
    },
    async refreshGraph() {
      const { data } = await api.graph()
      this.graph = data
    },
    async open(id: string) {
      const { data } = await api.getCard(id)
      this.current = data
      this.dirty = false
      this.view = 'editor'
    },
    async openByTitle(title: string) {
      const id = this.knownTitles[title.toLowerCase()]
      if (id) return this.open(id)
      this.confirm = {
        title: '创建悬空卡片',
        body: `「${title}」尚不存在，要现在创建吗？`,
        action: async () => {
          const { data } = await api.createCard(title, `# ${title}\n\n从悬空链接创建。\n`)
          this.toast('已创建卡片')
          await this.bootstrap()
          await this.open(data.id)
        },
      }
    },
    async save(partial?: { title?: string; body?: string }) {
      if (!this.current) return
      const title = partial?.title ?? this.current.title
      const body = partial?.body ?? this.current.body
      if (!title.trim()) {
        this.toast('标题不能为空', 'err')
        return
      }
      this.saving = true
      try {
        const { data } = await api.updateCard(this.current.id, { title, body })
        this.current = data
        this.dirty = false
        await this.refreshList()
        await this.refreshGraph()
        await this.refreshTags()
      } catch (e) {
        this.toast((e as Error).message, 'err')
      } finally {
        this.saving = false
      }
    },
    async createBlank() {
      const title = `未命名 ${new Date().toLocaleString('zh-CN', { hour12: false })}`
      const { data } = await api.createCard(title, '# 新卡片\n\n写下 [[双链]]。\n')
      await this.bootstrap()
      await this.open(data.id)
    },
  },
})
