<script setup lang="ts">
import { ref } from 'vue'
import { api } from '../api'
import { useApp } from '../stores/app'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const app = useApp()
const url = ref('https://blog.example.com/go-pool')
const err = ref('')
const busy = ref(false)

async function submit() {
  err.value = ''
  if (!url.value.trim()) {
    err.value = 'URL 为必填'
    return
  }
  busy.value = true
  try {
    const { data } = await api.clip(url.value.trim())
    app.toast('剪藏完成（当前为 Mock，未发真实请求）')
    await app.bootstrap()
    await app.open(data.id)
    emit('close')
  } catch (e) {
    err.value = (e as Error).message
    app.toast(err.value, 'err')
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div v-if="props.open" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/55 p-4">
    <div class="w-full max-w-lg rounded-2xl border border-[var(--line)] bg-[var(--ink-2)] p-5">
      <h3 class="font-display text-xl text-[var(--gold)]">网页剪藏</h3>
      <p class="mt-1 text-xs text-[var(--mist)]">默认 Mock Provider，读取本地 fixture，花费 ¥0。</p>
      <label class="mt-4 block text-xs text-[var(--mist)]">URL *</label>
      <input v-model="url" class="mt-1 w-full rounded-lg border border-[var(--line)] bg-[var(--ink)] px-3 py-2 outline-none" placeholder="https://..." />
      <p v-if="err" class="mt-1 text-xs text-[var(--rose)]">{{ err }}</p>
      <div class="mt-4 flex justify-end gap-2">
        <button class="text-sm text-[var(--mist)]" @click="emit('close')">取消</button>
        <button class="rounded-lg bg-[var(--gold)] px-3 py-1.5 text-sm text-[#1b1306] disabled:opacity-40" :disabled="busy" @click="submit">
          {{ busy ? '抓取中…' : '剪藏为卡片' }}
        </button>
      </div>
    </div>
  </div>
</template>
