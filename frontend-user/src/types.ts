export interface Tag {
  id?: string
  name: string
  full_path: string
}

export interface Link {
  id: string
  source_card_id: string
  target_card_id?: string
  target_title: string
  display_text: string
  excerpt: string
  source_title?: string
  dangling: boolean
}

export interface Card {
  id: string
  title: string
  body: string
  content_hash: string
  source_url?: string
  source_site?: string
  clipped_at?: string
  graph_version: number
  created_at: string
  updated_at: string
  tags?: Tag[]
  out_links?: Link[]
  back_links?: Link[]
}

export interface GraphNode {
  id: string
  card_id?: string
  title: string
  degree: number
  dangling: boolean
  orphan: boolean
  tags: string[]
  x?: number
  y?: number
}

export interface GraphEdge {
  id: string
  source: string
  target: string
  dangling: boolean
}

export interface GraphPayload {
  version: number
  nodes: GraphNode[]
  edges: GraphEdge[]
}

export interface PageMeta {
  page: number
  page_size: number
  total: number
}
