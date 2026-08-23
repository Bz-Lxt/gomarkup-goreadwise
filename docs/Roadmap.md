# GoReadwise 路线图

> 权威性：本文件定义 **WHEN**。`docs/Requirements.md` 定义 **WHAT**。
> 规模：10k–40k LoC 区间，MVP / V1 / V2 边界为强制前置。
> 冻结时间：2026-08-23 16:08 (GMT+8)

---

## 构建顺序决策（PHASE ORDER）

**裁定：Logic-First（交换 SOP 默认的 Phase 2 与 Phase 3）**

理由：前端主体是 Markdown 编辑器 + Cytoscape 图画布，组件结构、Props 与 Pinia store 完全派生于双链关系表与图数据结构。先落地解析引擎与图 API，再按真实契约构建 UI，避免契约确定后的大规模返工。

执行顺序：Phase 1 架构 → Phase 3 后端/契约 → Phase 2 前端 → Phase 4 QA → Phase 5 审计。

---

## 目录结构

```
GoReadwise/
├── backend/                 # Go 1.23 + chi + pgx
├── frontend-user/           # Vue 3.5 用户端（唯一真实 UI）
├── frontend-admin/          # SOP 占位：本项目无管理端，见 README
├── frontend-mp/             # SOP 占位：本项目无小程序，见 README
├── tests/                   # Playwright E2E（Mock 模式，¥0）
├── docs/
└── docker-compose.yml
```

单用户知识卡片箱不存在管理后台与微信小程序。`frontend-admin` / `frontend-mp` 仅作 SOP 目录占位并声明 N/A，禁止空壳 Vue 工程。

---

## 开发端口（随机，已探测空闲）

| 服务 | 宿主端口 | 容器端口 |
|---|---|---|
| frontend-user (nginx) | **15231** | 80 |
| backend (chi) | **15232** | 8080 |
| postgres | **15233** | 5432 |

`/deploy` 阶段再标准化为 8081+。

---

## MVP（本轮 `/auto` 必须交付）

数据模型、双链解析引擎（遮罩 + 正则）、卡片 CRUD、同步出边 diff、异步 Worker Pool（有界队列）、全图 API、三栏编辑器、知识星空图、Quick Edit、≥30 张种子卡片、Docker 一键启动。

对应 FR-1.1–1.5、FR-2.1–2.7、FR-3.1–3.4、FR-5.1、FR-6.1–6.4/6.6、FR-7.1–7.4、全部 NFR 交付门槛。

- [x] T-ARCH-1 Git / .gitignore / 目录骨架 / docker-compose
- [x] T-BE-1 配置、日志、时区、HTTP 信封
- [x] T-BE-2 PostgreSQL 迁移与 store 层
- [x] T-BE-3 双链解析引擎 + ≥25 边界测试
- [x] T-BE-4 卡片服务（同步出边 + 悬空链接 + 补链）
- [x] T-BE-5 Worker Pool + 幂等任务
- [x] T-BE-6 REST Handler + 图 API
- [x] T-BE-7 种子数据 ≥30 张互链卡片
- [x] T-FE-1 DesignSpec + 星空主题
- [x] T-FE-2 卡片列表 / 标签树 / 三栏编辑器
- [x] T-FE-3 Graph View + Quick Edit
- [x] T-QA-1 单测 + Playwright ≥5 路径
- [x] T-DOC-1 docs/API.md

## V1（本轮一并交付，否则 Mock 合法性与原 Prompt 剪藏能力无法闭环）

- [x] T-V1-1 网页剪藏 Mock/Real + SSRF（FR-4）
- [x] T-V1-2 重命名级联改写（FR-1.6）
- [x] T-V1-3 引用偏移与语境摘录（FR-2.8）
- [x] T-V1-4 SSE `graph:invalidated`（FR-3.6）
- [x] T-V1-5 子图查询 / 坐标持久化 / 孤立标记（FR-5.2–5.4）
- [x] T-V1-6 栏宽拖拽、节点着色与度数映射（FR-6.5 / FR-7.5–7.6）
- [x] T-V1-7 任务落库可恢复（FR-3.5）

## V2（明确不在本轮范围）

- [ ] T-V2-1 导出 Obsidian Vault zip（FR-8.1）
- [ ] T-V2-2 批量导入重建全图（FR-8.2）
- [ ] T-V2-3 >300 节点渲染降级（FR-7.7）

---

## 数据访问适配说明

需求冻结栈写了 sqlc。为满足 Docker 一键构建（镜像内不跑 `sqlc generate`），store 层使用 **pgx/v5 手写 SQL + 强类型结构体**。SQL 集中在 `internal/store`，语义等价于预生成 sqlc，不引入隐式 ORM。
