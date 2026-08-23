<script setup lang="ts">
import { useApp } from '../stores/app'
const app = useApp()
async function ok() {
  const c = app.confirm
  app.confirm = null
  if (!c) return
  try {
    await c.action()
  } catch (e) {
    app.toast((e as Error).message, 'err')
  }
}
</script>

<template>
  <div v-if="app.confirm" class="fixed inset-0 z-[70] flex items-center justify-center bg-black/55 p-4">
    <div class="w-full max-w-md rounded-2xl border border-[var(--line)] bg-[var(--ink-2)] p-5 shadow-2xl">
      <h3 class="font-display text-xl text-[var(--gold)]">{{ app.confirm.title }}</h3>
      <p class="mt-2 text-sm text-[var(--mist)]">{{ app.confirm.body }}</p>
      <div class="mt-5 flex justify-end gap-2">
        <button class="rounded-lg px-3 py-1.5 text-sm text-[var(--mist)] hover:text-white" @click="app.confirm = null">取消</button>
        <button class="rounded-lg bg-[var(--gold)] px-3 py-1.5 text-sm text-[#1b1306]" @click="ok">确认</button>
      </div>
    </div>
  </div>
</template>
