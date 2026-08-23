<script setup lang="ts">
import { ref, watch } from 'vue'
import { api } from '../api'
import { useApp } from '../stores/app'

const props = defineProps<{
  open: boolean
  cardId: string
  title: string
  dangling: boolean
}>()
const emit = defineEmits<{ close: []; opened: [string] }>()
const app = useApp()
const title = ref(props.title)
const body = ref('')
const titleErr = ref('')

watch(() => props.open, async (v) => {
  title.value = props.title
  body.value = ''
  titleErr.value = ''
  if (v && !props.dangling && props.cardId) {
    try {
      const { data } = await api.getCard(props.cardId)
      title.value = data.title
      body.value = data.body
    } catch (e) {
      app.toast((e as Error).message, 'err')
    }
  }
})

function validate() {
  titleErr.value = ''
  if (!title.value.trim()) {
    titleErr.value = '标题为必填'
    return false
  }
  if (/[\[\]\n]/.test(title.value)) {
    titleErr.value = '标题不能包含括号或换行'
    return false
  }
  return true
}

async function save() {
  if (!validate()) {
    app.toast('请修正标题后再保存', 'err')
    return
  }
  try {
    if (props.dangling || props.cardId.startsWith('ghost:')) {
      const { data } = await api.createCard(title.value.trim(), body.value || `# ${title.value}\n`)
      app.toast('幽灵节点已实体化')
      await app.bootstrap()
      emit('opened', data.id)
    } else {
      await api.updateCard(props.cardId, { title: title.value, body: body.value })
      app.toast('已保存')
      await app.bootstrap()
    }
    emit('close')
  } catch (e) {
    app.toast((e as Error).message, 'err')
  }
}

async function openFull() {
  if (props.dangling) {
    await save()
    return
  }
  await app.open(props.cardId)
  emit('close')
}
</script>

<template>
  <div v-if="open" class="fixed inset-0 z-[60] flex items-center justify-center bg-black/55 p-4">
    <div class="w-full max-w-2xl rounded-2xl border border-[var(--line)] bg-[var(--ink-2)] p-5">
      <div class="flex items-center justify-between">
        <h3 class="font-display text-xl text-[var(--gold)]">快速编辑</h3>
        <button class="text-[var(--mist)]" @click="emit('close')">×</button>
      </div>
      <label class="mt-4 block text-xs text-[var(--mist)]">标题 *</label>
      <input v-model="title" class="mt-1 w-full rounded-lg border border-[var(--line)] bg-[var(--ink)] px-3 py-2 outline-none" />
      <p v-if="titleErr" class="mt-1 text-xs text-[var(--rose)]">{{ titleErr }}</p>
      <label class="mt-3 block text-xs text-[var(--mist)]">Markdown</label>
      <textarea v-model="body" rows="10" class="mt-1 w-full rounded-lg border border-[var(--line)] bg-[var(--ink)] px-3 py-2 font-mono text-sm outline-none" />
      <div class="mt-4 flex justify-end gap-2">
        <button class="rounded-lg border border-[var(--line)] px-3 py-1.5 text-sm" @click="openFull">在完整编辑器打开</button>
        <button class="rounded-lg bg-[var(--gold)] px-3 py-1.5 text-sm text-[#1b1306]" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>
