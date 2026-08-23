package seed

import (
	"context"
	"log/slog"

	"goreadwise/internal/logger"
	"goreadwise/internal/model"
	"goreadwise/internal/service"
	"goreadwise/internal/store"
)

type Item struct {
	Title string
	Body  string
	Tags  []string
}

func Catalog() []Item {
	return []Item{
		{Title: "Zettelkasten", Tags: []string{"method/zettel"}, Body: `# Zettelkasten
原子笔记法。每张卡片只谈一件事，用 [[双向链接]] 织网，而不是目录树。
相关：[[原子卡片]]、[[双向链接]]、[[知识星空图]]。
#method/zettel`},
		{Title: "原子卡片", Tags: []string{"method/zettel"}, Body: `一张卡片 = 一个可独立引用的想法。不要写成日记长文。
参见 [[Zettelkasten]] 与 [[渐进式总结]]。
#method/zettel`},
		{Title: "双向链接", Tags: []string{"method/link"}, Body: `语法 [[卡片名称]]。保存后立刻可查 Backlink。围栏代码里的假链接必须被遮罩。
引擎见 [[Markdown 解析引擎]]。图上的边来自这里。
#method/link`},
		{Title: "Markdown 解析引擎", Tags: []string{"tech/golang"}, Body: `两段式：遮罩状态机 + 正则扫描。必须忽略围栏代码、行内代码、$$公式$$ 与 HTML 注释。
实现语言 [[Go 并发模型]]。测试覆盖 [[悬空链接]]。
#tech/golang`},
		{Title: "Go 并发模型", Tags: []string{"tech/golang/concurrency"}, Body: `CSP：goroutine + channel。Worker Pool 必须有界，禁止无界 goroutine。
卡片保存后的图扩散见 [[异步图维护]]。
#tech/golang/concurrency`},
		{Title: "异步图维护", Tags: []string{"tech/golang"}, Body: `同步事务写出去边；协程只做快照、补链、统计。幂等键 = card_id + content_hash。
队列满则同步降级。事件：[[SSE 失效通知]]。
#tech/golang`},
		{Title: "SSE 失效通知", Tags: []string{"tech/http"}, Body: `事件名 graph:invalidated。前端收到后按需拉 [[知识星空图]]。
#tech/http`},
		{Title: "知识星空图", Tags: []string{"ui/graph"}, Body: `Cytoscape.js 力导向。节点=卡片，边=引用。幽灵节点表示 [[悬空链接]]。
点击开 [[快速编辑]]。
#ui/graph`},
		{Title: "悬空链接", Tags: []string{"method/link"}, Body: `[[尚不存在的卡片]] 这种写法合法。target_card_id 为空。新卡片创建时自动补链。
演示：[[尚未写完的论文笔记]]。
#method/link`},
		{Title: "快速编辑", Tags: []string{"ui/editor"}, Body: `图视图浮层：标题 + 源码 + 保存。不做嵌套三栏，避免与 [[三栏编辑器]] 双写。
#ui/editor`},
		{Title: "三栏编辑器", Tags: []string{"ui/editor"}, Body: `左源码、中预览、右 Backlink。自动保存 1.5s。补全由 [[双向链接]] 触发。
#ui/editor`},
		{Title: "网页剪藏", Tags: []string{"inbox/clip"}, Body: `URL → HTML → Markdown → 卡片。默认 Mock。切换 CLIP_PROVIDER。
防护见 [[SSRF 防护]]。
#inbox/clip`},
		{Title: "SSRF 防护", Tags: []string{"tech/security"}, Body: `拒绝 loopback / RFC1918 / link-local / 非 http(s)。重定向同样校验。
服务于 [[网页剪藏]]。
#tech/security`},
		{Title: "路径式标签", Tags: []string{"method/tag"}, Body: `#tech/golang/concurrency 自动建祖先。查询前缀包含子孙。
和 [[多级标签树]] 是同一模型。
#method/tag`},
		{Title: "多级标签树", Tags: []string{"method/tag"}, Body: `tags.full_path + parent_id。写入时 Ensure 祖先。
见 [[路径式标签]]。
#method/tag`},
		{Title: "Readwise 方法", Tags: []string{"method/read"}, Body: `高亮回收 + 间隔复习。本项目取其「剪藏进卡片箱」一半，另一半是 [[Obsidian 核心]]。
#method/read`},
		{Title: "Obsidian 核心", Tags: []string{"method/zettel"}, Body: `本地 Markdown + 双链 + Graph View。我们用 Go 服务端维护关系表。
对照 [[Zettelkasten]]。
#method/zettel`},
		{Title: "渐进式总结", Tags: []string{"method/read"}, Body: `Tiago Forte：图层高亮，而不是一次写完。适合从 [[网页剪藏]] 长文压成 [[原子卡片]]。
#method/read`},
		{Title: "间隔重复", Tags: []string{"method/read"}, Body: `Anki / SuperMemo。本轮不做调度算法，只把复习材料变成可链接卡片。
相关 [[Readwise 方法]]。
#method/read`},
		{Title: "PostgreSQL 图查询", Tags: []string{"tech/postgres"}, Body: `关系表 + 递归 CTE 可做 N 跳邻域。索引必须覆盖 source / target / title_norm。
#tech/postgres`},
		{Title: "chi 路由", Tags: []string{"tech/golang/http"}, Body: `标准 net/http 中间件。资源名词复数：/api/v1/cards。
#tech/golang/http`},
		{Title: "结构化日志", Tags: []string{"tech/golang"}, Body: `只用 log/slog。生产屏蔽 debug。禁止 fmt.Println。
#tech/golang`},
		{Title: "北京时区", Tags: []string{"tech/ops"}, Body: `容器 TZ=Asia/Shanghai。Go 用 clock.Now()。展示 yyyy-MM-dd HH:mm:ss。
#tech/ops`},
		{Title: "Docker 交付", Tags: []string{"tech/ops"}, Body: `compose up 即用。前端等后端 healthy。镜像需 arm64/amd64 均可拉取。
#tech/ops`},
		{Title: "Cytoscape.js", Tags: []string{"ui/graph"}, Body: `画布库。布局用 fcose。拖拽后 PATCH /graph/positions。
承载 [[知识星空图]]。
#ui/graph`},
		{Title: "CodeMirror 6", Tags: []string{"ui/editor"}, Body: `左栏编辑器。[[ 触发补全。主题跟随星空墨色。
#ui/editor`},
		{Title: "Vue 3 组合式", Tags: []string{"ui/frontend"}, Body: `Pinia 存当前卡片与图快照。编辑器与图画布共享同一 API 契约。
#ui/frontend`},
		{Title: "Worker Pool", Tags: []string{"tech/golang/concurrency"}, Body: `固定 worker + 有界 queue。满则同步。任务落库可恢复。
实现 [[Go 并发模型]]。
#tech/golang/concurrency`},
		{Title: "内容哈希", Tags: []string{"tech/golang"}, Body: `SHA-256(body)。异步任务幂等键的一半。
#tech/golang`},
		{Title: "操作日志", Tags: []string{"tech/ops"}, Body: `重命名级联必须记 op_logs，便于审计回放。
#tech/ops`},
		{Title: "幽灵节点", Tags: []string{"ui/graph"}, Body: `图中半透明节点，id 为 ghost:title。点击即以该标题创建卡片。
来自 [[悬空链接]]。
#ui/graph`},
		{Title: "Backlink 栏", Tags: []string{"ui/editor"}, Body: `显示谁引用了我，以及 excerpt 语境。数据来自同步事务，必须立刻正确。
#ui/editor`},
		{Title: "力导向布局", Tags: []string{"ui/graph"}, Body: `fcose / cose-bilkent。度数大的节点更重。
#ui/graph`},
		{Title: "种子网络", Tags: []string{"meta"}, Body: `首次启动写入 ≥30 张互链卡片，保证打开就能看见 [[知识星空图]]。
#meta`},
		{Title: "认知负荷", Tags: []string{"method/learn"}, Body: `工作记忆有限。原子化卡片降低负荷，见 [[原子卡片]] 与 [[渐进式总结]]。
#method/learn`},
		{Title: "提取练习", Tags: []string{"method/learn"}, Body: `主动回忆优于重读。把问题写成卡片并链到 [[间隔重复]]。
#method/learn`},
		{Title: "费曼技巧", Tags: []string{"method/learn"}, Body: `用自己的话讲清楚。讲不通就回到 [[原子卡片]] 拆更小。
#method/learn`},
		{Title: "第二大脑", Tags: []string{"method/read"}, Body: `外部记忆系统。本箱是开发者向的第二大脑，核心是 [[双向链接]]。
#method/read`},
		{Title: "PARA 方法", Tags: []string{"method/read"}, Body: `Projects / Areas / Resources / Archives。标签树可映射，见 [[路径式标签]]。
#method/read`},
		{Title: "SQL 注入防线", Tags: []string{"tech/security"}, Body: `全部查询参数化。和 [[SSRF 防护]] 同属输入边界。
#tech/security`},
		{Title: "幂等任务", Tags: []string{"tech/golang"}, Body: `card_id + content_hash + kind 唯一。见 [[内容哈希]] 与 [[Worker Pool]]。
#tech/golang`},
		{Title: "健康检查", Tags: []string{"tech/ops"}, Body: `compose 依赖 backend healthy。实现见 [[Docker 交付]]。
#tech/ops`},
		{Title: "前端契约", Tags: []string{"ui/frontend"}, Body: `卡片、图、反链的 JSON 形状冻结在 docs/API.md。[[Vue 3 组合式]] 按此建模。
#ui/frontend`},

		{Title: "测试金字塔", Tags: []string{"tech/qa"}, Body: `单测护住 [[Markdown 解析引擎]]，E2E 护住 [[三栏编辑器]]。
#tech/qa`},
		{Title: "契约先行", Tags: []string{"tech/qa"}, Body: `先冻结 docs/API.md 再写 [[前端契约]]。
#tech/qa`},
		{Title: "可观测性", Tags: []string{"tech/ops"}, Body: `slog + /metrics + [[健康检查]]。
#tech/ops`},
		{Title: "备份意识", Tags: []string{"tech/ops"}, Body: `Postgres volume 是唯一状态。导出留给 V2，见 [[种子网络]]。
#tech/ops`},
		{Title: "输入边界", Tags: []string{"tech/security"}, Body: `标题、正文、URL 都在 handler 校验。与 [[SQL 注入防线]] 配套。
#tech/security`},
		{Title: "展示时区", Tags: []string{"tech/ops"}, Body: `用户看见的是 yyyy-MM-dd HH:mm:ss，源是 [[北京时区]]。
#tech/ops`},
		{Title: "画布交互", Tags: []string{"ui/graph"}, Body: `拖拽保存坐标，点击进入 [[快速编辑]]。
#ui/graph`},
		{Title: "自动补全", Tags: []string{"ui/editor"}, Body: `输入 [[ 后请求 /cards/suggest，数据来自 [[chi 路由]]。
#ui/editor`},
		{Title: "防抖保存", Tags: []string{"ui/editor"}, Body: `1.5 秒防抖 + Cmd+S，避免打满 [[Worker Pool]]。
#ui/editor`},
		{Title: "空态文案", Tags: []string{"ui/frontend"}, Body: `空星图不是空白页，引导去写 [[原子卡片]]。
#ui/frontend`},
		{Title: "Quick Edit 边界", Tags: []string{"ui/editor"}, Body: `只改标题和正文，不渲染 Backlink，避免与 [[三栏编辑器]] 抢状态。
#ui/editor`},
	}
}

func Maybe(ctx context.Context, db *store.DB, cards *service.CardService) error {
	n, err := db.CountAliveCards(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		logger.L().Info("seed skipped", slog.Int("cards", n))
		return nil
	}
	for _, it := range Catalog() {
		if _, err := cards.Create(ctx, model.CreateCardInput{Title: it.Title, Body: it.Body, Tags: it.Tags}); err != nil {
			logger.L().Error("seed card", slog.String("title", it.Title), slog.String("err", err.Error()))
			return err
		}
	}
	logger.L().Info("seed completed", slog.Int("cards", len(Catalog())))
	return nil
}

// ExpectedOutgoing documents the intended first-hop network for QA assertions.
// Values are titles that should appear as outgoing wikilinks after parse.
func ExpectedOutgoing() map[string][]string {
	return map[string][]string{
		"Zettelkasten":   {"双向链接", "原子卡片", "知识星空图"},
		"原子卡片":           {"Zettelkasten", "渐进式总结"},
		"双向链接":           {"Markdown 解析引擎"},
		"Markdown 解析引擎":  {"Go 并发模型", "悬空链接"},
		"Go 并发模型":        {"异步图维护"},
		"异步图维护":          {"SSE 失效通知"},
		"SSE 失效通知":       {"知识星空图"},
		"知识星空图":          {"悬空链接", "快速编辑"},
		"悬空链接":           {"尚未写完的论文笔记"},
		"快速编辑":           {"三栏编辑器"},
		"三栏编辑器":          {"双向链接"},
		"网页剪藏":           {"SSRF 防护"},
		"SSRF 防护":        {"网页剪藏"},
		"路径式标签":          {"多级标签树"},
		"多级标签树":          {"路径式标签"},
		"Readwise 方法":    {"Obsidian 核心"},
		"Obsidian 核心":    {"Zettelkasten"},
		"渐进式总结":          {"网页剪藏", "原子卡片"},
		"间隔重复":           {"Readwise 方法"},
		"PostgreSQL 图查询": {},
		"chi 路由":         {},
		"结构化日志":          {},
		"北京时区":           {},
		"Docker 交付":      {},
		"Cytoscape.js":   {"知识星空图"},
		"CodeMirror 6":   {},
		"Vue 3 组合式":      {},
		"Worker Pool":    {"Go 并发模型"},
		"内容哈希":           {},
		"操作日志":           {},
		"幽灵节点":           {"悬空链接"},
		"Backlink 栏":     {},
		"力导向布局":          {},
		"种子网络":           {"知识星空图"},
		"认知负荷":           {"原子卡片", "渐进式总结"},
		"提取练习":           {"间隔重复"},
		"费曼技巧":           {"原子卡片"},
		"第二大脑":           {"双向链接"},
		"PARA 方法":        {"路径式标签"},
		"SQL 注入防线":       {"SSRF 防护"},
		"幂等任务":           {"内容哈希", "Worker Pool"},
		"健康检查":           {"Docker 交付"},
		"前端契约":           {"Vue 3 组合式"},
		"Quick Edit 边界":  {"三栏编辑器"},
	}
}

func Titles() []string {
	items := Catalog()
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Title)
	}
	return out
}

func Find(title string) (Item, bool) {
	for _, it := range Catalog() {
		if it.Title == title {
			return it, true
		}
	}
	return Item{}, false
}
