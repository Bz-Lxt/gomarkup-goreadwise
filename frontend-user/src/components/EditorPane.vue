<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { EditorView, keymap, placeholder } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { markdown } from '@codemirror/lang-markdown'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { autocompletion, type CompletionContext } from '@codemirror/autocomplete'
import { oneDark } from '@codemirror/theme-one-dark'
import { api } from '../api'
import { useApp } from '../stores/app'

const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [string] }>()
const app = useApp()
const host = ref<HTMLDivElement | null>(null)
let view: EditorView | null = null
let applying = false

async function wikiComplete(ctx: CompletionContext) {
  const match = ctx.matchBefore(/\[\[([^\]]*)$/)
  if (!match) return null
  const q = match.text.replace(/^\[\[/, '')
  let titles = Object.keys(app.knownTitles)
  try {
    const { data } = await api.suggest(q)
    titles = data.map((c) => c.title)
  } catch {
    /* local fallback */
  }
  return {
    from: match.from + 2,
    options: titles.map((t) => ({ label: t, type: 'text' })),
  }
}

function mount() {
  if (!host.value) return
  view = new EditorView({
    parent: host.value,
    state: EditorState.create({
      doc: props.modelValue,
      extensions: [
        markdown(),
        history(),
        oneDark,
        placeholder('写下原子想法，用 [[卡片名]] 织网…'),
        autocompletion({ override: [wikiComplete] }),
        keymap.of([
          ...defaultKeymap,
          ...historyKeymap,
          { key: 'Mod-s', preventDefault: true, run: () => { app.save({ body: view?.state.doc.toString() }); return true } },
        ]),
        EditorView.updateListener.of((u) => {
          if (!u.docChanged || applying) return
          emit('update:modelValue', u.state.doc.toString())
        }),
      ],
    }),
  })
}

watch(() => props.modelValue, (v) => {
  if (!view) return
  if (v === view.state.doc.toString()) return
  applying = true
  view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: v } })
  applying = false
})

onMounted(mount)
onBeforeUnmount(() => view?.destroy())
</script>

<template>
  <div ref="host" class="h-full min-h-[240px]" />
</template>
