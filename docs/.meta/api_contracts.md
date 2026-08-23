# API Contracts — Phase 3 Gate

时间：2026-08-23 16:20 (GMT+8)

## 网页剪藏 Provider

| 项 | 结论 |
|---|---|
| 认证 | 无第三方 API Key。Real 模式为裸 HTTP GET。 |
| Mock | `CLIP_PROVIDER=mock`（默认）。读取 `backend/testdata/clips/*.html`。 |
| 请求形状 | `POST /api/v1/clips` body `{ "url": "https://..." }` |
| 成功响应 | 201 + 标准卡片信封 |
| 错误 | VALIDATION / CLIP_DENIED / CONFLICT |
| 单价 | ¥0（无计量 API） |
| Mock 探活 | 已用 fixture `tech-blog.html` / `paper.html` / `docs.html` / `default.html` 走 Extract，标题与正文可解析。 |
| Real | 标记 **UNVERIFIED**（本轮未对公网发起真实抓取，避免把 QA 绑在外网）。实现已接线：超时 10s、5MB、最多 3 次重定向、SSRF 黑名单。 |

结论：Contract Gate = **verified (mock) / UNVERIFIED (real, no live key/network required)**。
