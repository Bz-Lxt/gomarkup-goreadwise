<script setup lang="ts">
import { useApp } from '../stores/app'
const app = useApp()
</script>

<template>
  <div class="flex h-full flex-col overflow-auto">
    <h3 class="px-4 pt-4 font-display text-lg text-[var(--gold)]">反向链接</h3>
    <p class="px-4 text-xs text-[var(--mist)]">谁引用了当前卡片 · 同步事务可见</p>
    <div class="mt-3 flex-1 space-y-2 px-3 pb-4">
      <button
        v-for="l in app.current?.back_links || []"
        :key="l.id"
        class="w-full rounded-xl border border-[var(--line)] bg-[var(--ink)]/60 px-3 py-2 text-left hover:border-[var(--gold)]/50"
        @click="app.open(l.source_card_id)"
      >
        <div class="text-sm font-medium">{{ l.source_title || '未命名' }}</div>
        <div class="mt-1 line-clamp-2 text-xs text-[var(--mist)]">{{ l.excerpt }}</div>
      </button>
      <p v-if="!(app.current?.back_links || []).length" class="px-1 py-8 text-center text-sm text-[var(--mist)]">
        还没有人引用这张卡片。在别处写下 [[{{ app.current?.title }}]]。
      </p>
      <h4 class="pt-3 text-xs uppercase tracking-wider text-[var(--gold-dim)]">正向引用</h4>
      <div
        v-for="l in app.current?.out_links || []"
        :key="l.id"
        class="rounded-lg px-2 py-1 text-sm"
        :class="l.dangling ? 'text-[var(--rose)]' : 'text-[var(--gold)]'"
      >
        [[{{ l.target_title }}]]
        <span v-if="l.dangling" class="text-[10px]">幽灵</span>
      </div>
    </div>
  </div>
</template>
