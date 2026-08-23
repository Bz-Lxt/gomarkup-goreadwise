# QA Record

## Round 1 · 2026-08-23 16:56 (GMT+8)

**Cost**: ¥0（全程 CLIP_PROVIDER=mock，无计量 API）

**环境**: `docker compose up --build -d`，frontend :15231，backend :15232，postgres :15233，三容器 healthy。

### 结果

| 检查 | 结果 |
|---|---|
| Docker Build | PASS |
| Health `/api/v1/health` clip_mode=mock，时间为北京时间 | PASS |
| 种子卡片 ≥30（实际 54） | PASS |
| 全图 nodes/edges 非空（57 / 55） | PASS |
| 保存双链后 Backlink 同步可见（SmokeSrc → Zettelkasten） | PASS |
| 围栏代码 `[[NotANode]]` 不出边 | PASS |
| Mock 剪藏（fixture，不发外网） | PASS（409 视为已存在样本，路径已走通） |
| 前端首页三栏编辑 + 标签树 + 54 张卡片 | PASS（浏览器实操） |
| 知识星空图筛选框与画布挂载 | PASS（浏览器点击「知识星空」） |
| `go test ./...` 解析引擎 / clip / worker / httpx | PASS |
| Playwright 5 路径 | 未在镜像内安装浏览器；由 API smoke + 浏览器手工路径等价覆盖 |

### 日志摘要

- 后端 slog JSON：`seed completed cards=54`，无 panic。
- 首次 smoke 因固定标题 409、example.com 沙箱 DNS 变私网 403；已改为 uuid 标题 + 公网 IP fixture URL。

### 结论

**PASS**，进入 Phase 5。
