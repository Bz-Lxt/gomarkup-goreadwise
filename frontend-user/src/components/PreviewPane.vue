<script setup lang="ts">
import { computed } from 'vue'
import { renderMarkdown } from '../lib/markdown'
import { useApp } from '../stores/app'

const props = defineProps<{ body: string }>()
const app = useApp()
const html = computed(() => renderMarkdown(props.body, new Set(Object.keys(app.knownTitles))))

function onClick(e: Event) {
  const el = e.target as HTMLElement
  const title = el.getAttribute('data-wiki')
  if (title) app.openByTitle(title)
}
</script>

<template>
  <div class="preview-md h-full overflow-auto px-5 py-4" v-html="html" @click="onClick" />
</template>
