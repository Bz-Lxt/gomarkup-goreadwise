import MarkdownIt from 'markdown-it'

const wikiRe = /\[\[([^\[\]|\n]+?)(?:\|([^\[\]\n]+?))?\]\]/g

export function renderMarkdown(src: string, known: Set<string>) {
  const md = new MarkdownIt({ html: false, linkify: true, breaks: false })
  const raw = md.render(src || '')
  return raw.replace(wikiRe, (_full: string, target: string, display?: string) => {
    const title = (target || '').trim()
    const label = (display || title).trim()
    const dangling = !known.has(title.toLowerCase())
    const cls = dangling ? 'wiki-link dangling' : 'wiki-link'
    return `<span class="${cls}" data-wiki="${escapeAttr(title)}">${escapeHtml(label)}</span>`
  })
}

function escapeHtml(s: string) {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] || c))
}

function escapeAttr(s: string) {
  return escapeHtml(s).replace(/`/g, '')
}
