<script setup lang="ts">
import { useApp } from '../stores/app'
const app = useApp()
</script>

<template>
  <div class="fixed right-4 top-4 z-[80] flex w-[min(360px,calc(100vw-2rem))] flex-col gap-2">
    <div
      v-for="t in app.toasts"
      :key="t.id"
      class="flex items-start gap-3 rounded-xl border px-3 py-2 shadow-lg"
      :class="t.kind === 'err' ? 'border-rose-400/40 bg-[#2a1518] text-[#f0c8c8]' : 'border-teal-400/30 bg-[#13201f] text-[#d7f0ee]'"
    >
      <p class="flex-1 text-sm">{{ t.text }}</p>
      <button class="text-xs opacity-70 hover:opacity-100" @click="app.dismissToast(t.id)">×</button>
    </div>
  </div>
</template>
