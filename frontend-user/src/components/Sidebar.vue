<script setup lang="ts">
import { computed, ref } from 'vue'
import { useApp } from '../stores/app'
import { formatBeijing } from '../api'

const app = useApp()
const q = ref('')

const filtered = computed(() => {
  const s = q.value.trim().toLowerCase()
  return app.cards.filter((c) => {
    if (app.tagFilter && !(c.tags || []).some((t) => t.full_path === app.tagFilter || t.full_path.startsWith(app.tagFilter + '/'))) {
      return false
    }
    if (!s) return true
    return c.title.toLowerCase().includes(s) || (c.body || '').toLowerCase().includes(s)
  })
})

function pickTag(path: string) {
  app.tagFilter = app.tagFilter === path ? '' : path
  app.refreshList()
}
</script>

<template>
  <aside class="flex h-full w-full flex-col border-r border-[var(--line)] bg-[var(--ink-2)]/90">
    <div class="px-4 pb-3 pt-5">
      <p class="text-[11px] uppercase tracking-[0.22em] text-[var(--gold-dim)]">GoReadwise</p>
      <h1 class="font-display text-2xl leading-tight text-[var(--parchment)]">墨水天文台</h1>
      <p class="mt-1 text-xs text-[var(--mist)]">{{ app.total }} 张卡片 · Zettelkasten</p>
    </div>
    <div class="flex gap-2 px-4">
      <button class="flex-1 rounded-lg bg-[var(--gold)] px-2 py-1.5 text-sm text-[#1b1306]" @click="app.createBlank()">新卡片</button>
      <button class="rounded-lg border border-[var(--line)] px-2 py-1.5 text-sm" @click="$emit('clip')">剪藏</button>
    </div>
    <div class="px-4 pt-3">
      <input
        v-model="q"
        class="w-full rounded-lg border border-[var(--line)] bg-[var(--ink)] px-3 py-2 text-sm outline-none focus:border-[var(--gold)]"
        placeholder="搜索标题 / 正文"
      />
    </div>
    <div class="mt-3 max-h-36 overflow-auto px-3">
      <button
        v-for="t in app.tags"
        :key="t.full_path"
        class="mb-1 mr-1 inline-block rounded-full border px-2 py-0.5 text-[11px]"
        :class="app.tagFilter === t.full_path ? 'border-[var(--gold)] text-[var(--gold)]' : 'border-[var(--line)] text-[var(--mist)]'"
        @click="pickTag(t.full_path)"
      >#{{ t.full_path }}</button>
    </div>
    <div class="mt-2 flex-1 overflow-auto px-2 pb-4">
      <button
        v-for="c in filtered"
        :key="c.id"
        class="mb-1 w-full rounded-xl px-3 py-2 text-left hover:bg-white/5"
        :class="app.current?.id === c.id ? 'bg-white/8 ring-1 ring-[var(--gold)]/40' : ''"
        @click="app.open(c.id)"
      >
        <div class="truncate font-medium">{{ c.title }}</div>
        <div class="text-[11px] text-[var(--mist)]">{{ formatBeijing(c.updated_at) }}</div>
      </button>
      <p v-if="!filtered.length" class="px-3 py-8 text-center text-sm text-[var(--mist)]">没有匹配的卡片</p>
    </div>
  </aside>
</template>
