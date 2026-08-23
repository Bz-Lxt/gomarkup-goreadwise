# GoReadwise

面向开发者的知识卡片箱（Zettelkasten）。Go 解析 `[[双链]]`，Vue 三栏编辑 + Cytoscape 星空图。

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 http://127.0.0.1:15231 。后端 API http://127.0.0.1:15232 。无需其它手工步骤。

## 2. 使用说明

左侧选卡片或「新卡片」。左栏写 Markdown，中栏实时预览，右栏看反向链接。用 `[[卡片名]]` 建立双链，用 `#tech/golang` 写路径标签。顶部切到「知识星空」可拖拽节点、点击弹出快速编辑。网页剪藏默认走本地 fixture，不访问外网。

## 3. 服务列表及API说明

| 服务 | 地址 |
|---|---|
| 用户前端 | http://127.0.0.1:15231 |
| 后端 API | http://127.0.0.1:15232/api/v1 |
| 健康检查 | http://127.0.0.1:15232/api/v1/health |
| 星空图 | http://127.0.0.1:15232/api/v1/graph |
| Postgres | 127.0.0.1:15233（仅开发探测） |

完整端点、示例与错误码见 `docs/API.md`。

## 4. 测试账号

单用户本地箱，无登录。首次启动自动写入 50+ 张互链示例卡片。

## 5. 题目内容

实现极客学术 / 技术文献卡片箱：网页剪藏、多级标签、`[[双向链接]]`、反向链接、知识网络可视化。Go 后端手写遮罩+正则解析器，保存时同步维护出边，异步协程更新图快照。

## 6. 项目结构

```
backend/           Go API、解析引擎、Worker、种子
frontend-user/     Vue 3 用户端
frontend-admin/    SOP 占位（本项目无管理端）
frontend-mp/       SOP 占位（本项目无小程序）
tests/             API smoke + Playwright
docs/              Requirements / Roadmap / API / DesignSpec
```

## 7. API 模拟与切换指南

网页剪藏是唯一外部依赖。

- **默认 Mock**：`CLIP_PROVIDER=mock`。读取 `backend/testdata/clips/*.html`（技术博客 / 论文 / 文档站 / 默认页），**不发起真实 HTTP**。QA 与 `docker compose` 均为此模式，成本 ¥0。
- **真实抓取**：将 compose 中 backend 环境改为 `CLIP_PROVIDER=real`。走 `net/http`，10s 超时、5MB 上限、最多 3 次重定向，并拒绝回环/私网/link-local/非 http(s)（SSRF）。
- 真实路径已完整接线（`internal/clip.RealProvider`），不是空壳。切换后重启 backend 即可。
