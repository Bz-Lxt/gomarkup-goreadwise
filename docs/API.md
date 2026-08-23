# GoReadwise API

Base URL：`/api/v1`  
信封：成功 `{ "data": ... }`；列表另含 `meta`；失败 `{ "error": { "code", "message" } }`。

时间字段为 RFC3339，服务器按 Asia/Shanghai 生成。前端展示统一 `yyyy-MM-dd HH:mm:ss`。

## 错误码

| HTTP | code | 含义 |
|---|---|---|
| 400 | VALIDATION | 字段缺失、类型错误、超长、非法 JSON |
| 403 | CLIP_DENIED | SSRF / 非 http(s) / 私网地址 |
| 404 | NOT_FOUND | 卡片不存在或已软删除 |
| 409 | CONFLICT | 标题唯一冲突 |
| 500 | INTERNAL | 未分类服务端错误 |

---

## GET /health

**请求**：无  
**响应示例**

```json
{"data":{"status":"ok","time":"2026-08-23 16:20:00","clip_mode":"mock","uptime_s":12}}
```

## GET /metrics

**响应示例**

```json
{"data":{"queue_depth":0,"queue_cap":64,"sync_fallback":0,"jobs_done":3,"card_count":34,"edge_count":40,"graph_version":8}}
```

## GET /cards

Query：`q` string、`tag` 路径前缀、`page`>=1、`page_size` 1–100。

```json
{"data":[{"id":"...","title":"Zettelkasten","body":"...","tags":[{"full_path":"method/zettel"}]}],"meta":{"page":1,"page_size":30,"total":34}}
```

## POST /cards

```json
{"title":"新卡片","body":"参见 [[Zettelkasten]]","tags":["inbox"]}
```

成功 201，`Location: /api/v1/cards/{id}`。标题必填，≤200 字，禁止 `[]` 与换行。

## GET /cards/{id}

返回卡片 + `out_links` + `back_links` + `tags`。保存后立即可见新的出边与反链。

## PATCH /cards/{id}

部分更新 `title` / `body` / `tags`。改标题会级联改写其他卡片正文中的 `[[旧名]]`。

## DELETE /cards/{id}

软删除。入边 `target_card_id` 置空，变为悬空链接。

## GET /cards/{id}/links

```json
{"data":{"out_links":[],"back_links":[{"source_title":"原子卡片","excerpt":"...","dangling":false}]}}
```

## GET /cards/suggest?q=

自动补全标题，最多 12 条。

## GET /graph

全图。可选 `root={uuid}&depth=1..3` 返回子图。

```json
{"data":{"version":8,"nodes":[{"id":"...","title":"双向链接","degree":4,"dangling":false,"orphan":false,"tags":["method/link"]}],"edges":[{"id":"...","source":"...","target":"...","dangling":false}]}}
```

幽灵节点 id 形如 `ghost:尚未写完的论文笔记`。

## PATCH /graph/positions

```json
{"positions":[{"id":"<card-uuid>","x":12.5,"y":-8.0}]}
```

## GET /tags

路径式标签树扁平列表，按 `full_path` 排序。

## POST /clips

```json
{"url":"https://blog.example.com/go-pool"}
```

Mock 模式读取 `testdata/clips/*.html`，不发外网。Real 模式校验 SSRF。

## GET /events

SSE。事件 `graph:invalidated`，payload 含 `version`。

## POST /admin/rebuild

全量重解析出边与标签，返回 `{ "rebuilt": true, "cards": 34 }`。
