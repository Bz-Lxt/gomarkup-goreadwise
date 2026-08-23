package engine

import (
	"strings"
	"testing"
)

func titles(r ParseResult) []string {
	out := make([]string, len(r.Links))
	for i, l := range r.Links {
		out[i] = l.Target
	}
	return out
}

func TestParseBasicWikilink(t *testing.T) {
	r := Parse("See [[Go 并发]] for details.")
	if len(r.Links) != 1 || r.Links[0].Target != "Go 并发" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseAlias(t *testing.T) {
	r := Parse("[[Go 并发|CSP 模型]]")
	if len(r.Links) != 1 {
		t.Fatalf("links=%d", len(r.Links))
	}
	if r.Links[0].Target != "Go 并发" || r.Links[0].Display != "CSP 模型" {
		t.Fatalf("got %+v", r.Links[0])
	}
}

func TestParseIgnoresFencedCode(t *testing.T) {
	src := "real [[CardA]]\n```go\nfmt.Println(\"[[fake]]\")\n```\nmore [[CardB]]"
	r := Parse(src)
	got := titles(r)
	if len(got) != 2 || got[0] != "CardA" || got[1] != "CardB" {
		t.Fatalf("got %v", got)
	}
}

func TestParseIgnoresTildeFence(t *testing.T) {
	src := "[[Keep]]\n~~~\n[[hidden]]\n~~~\n"
	r := Parse(src)
	if len(r.Links) != 1 || r.Links[0].Target != "Keep" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseIgnoresInlineCode(t *testing.T) {
	r := Parse("use `[[not a link]]` then [[Real]]")
	if len(r.Links) != 1 || r.Links[0].Target != "Real" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseIgnoresDoubleBacktickInline(t *testing.T) {
	r := Parse("``[[nope]]`` and [[Yes]]")
	if len(r.Links) != 1 || r.Links[0].Target != "Yes" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseIgnoresMathBlock(t *testing.T) {
	r := Parse("before [[A]]\n$$ a = [[not]] $$\nafter [[B]]")
	got := titles(r)
	if len(got) != 2 || got[0] != "A" || got[1] != "B" {
		t.Fatalf("got %v", got)
	}
}

func TestParseIgnoresInlineMath(t *testing.T) {
	r := Parse("eq $x=[[hid]]$ then [[Vis]]")
	if len(r.Links) != 1 || r.Links[0].Target != "Vis" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseIgnoresHTMLComment(t *testing.T) {
	r := Parse("[[Live]] <!-- [[ghost]] --> [[Also]]")
	got := titles(r)
	if len(got) != 2 || got[0] != "Live" || got[1] != "Also" {
		t.Fatalf("got %v", got)
	}
}

func TestParseEscapedWikilink(t *testing.T) {
	r := Parse(`literal \[[not-a-link]] and [[Real]]`)
	if len(r.Links) != 1 || r.Links[0].Target != "Real" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseEmptyTitleRejected(t *testing.T) {
	r := Parse("[[]] [[  ]] [[ok]]")
	if len(r.Links) != 1 || r.Links[0].Target != "ok" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseOversizedTitleRejected(t *testing.T) {
	long := strings.Repeat("汉", 201)
	r := Parse("[[" + long + "]] [[短]]")
	if len(r.Links) != 1 || r.Links[0].Target != "短" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseUnicodeEmoji(t *testing.T) {
	r := Parse("see [[🧠 记忆宫殿]] and [[Zettelkasten]]")
	got := titles(r)
	if len(got) != 2 || got[0] != "🧠 记忆宫殿" {
		t.Fatalf("got %v", got)
	}
}

func TestParseChineseHashtags(t *testing.T) {
	r := Parse("正文 #认知科学/记忆 与 #tech/golang")
	if len(r.Tags) != 2 {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestParseHeadingNotHashtag(t *testing.T) {
	r := Parse("# 标题不应成标签\n正文 #real-tag")
	if len(r.Tags) != 1 || r.Tags[0] != "real-tag" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestParseNestedHeadingLevels(t *testing.T) {
	r := Parse("### Still heading\n#ok")
	if len(r.Tags) != 1 || r.Tags[0] != "ok" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestParseHashtagNotInsideWord(t *testing.T) {
	r := Parse("C# is not a tag, but #csharp is")
	if len(r.Tags) != 1 || r.Tags[0] != "csharp" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestParseHashtagInsideCodeIgnored(t *testing.T) {
	r := Parse("```\n#hidden/path\n```\n#visible")
	if len(r.Tags) != 1 || r.Tags[0] != "visible" {
		t.Fatalf("tags=%v", r.Tags)
	}
}

func TestParseMultipleSameLinkDifferentOffsets(t *testing.T) {
	r := Parse("[[A]] then again [[A]]")
	if len(r.Links) != 2 {
		t.Fatalf("want 2 got %d", len(r.Links))
	}
	if r.Links[0].OffsetStart == r.Links[1].OffsetStart {
		t.Fatal("offsets should differ")
	}
}

func TestParseExcerptPresent(t *testing.T) {
	r := Parse("context before [[Target]] context after")
	if !strings.Contains(r.Links[0].Excerpt, "Target") {
		t.Fatalf("excerpt=%q", r.Links[0].Excerpt)
	}
}

func TestParseUnclosedFenceMasksRest(t *testing.T) {
	r := Parse("[[Keep]]\n```\n[[hidden]] still hidden")
	if len(r.Links) != 1 || r.Links[0].Target != "Keep" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseUnclosedCommentMasksRest(t *testing.T) {
	r := Parse("[[A]] <!-- [[B]] [[C]]")
	if len(r.Links) != 1 || r.Links[0].Target != "A" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseNewlineBreaksWikilink(t *testing.T) {
	r := Parse("[[broken\nlink]] [[Good]]")
	if len(r.Links) != 1 || r.Links[0].Target != "Good" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseHashStable(t *testing.T) {
	a := HashContent("same")
	b := HashContent("same")
	c := HashContent("other")
	if a != b || a == c || len(a) != 64 {
		t.Fatalf("a=%s b=%s c=%s", a, b, c)
	}
}

func TestRewriteWikilinksSkipsMasked(t *testing.T) {
	src := "[[Old]] and `[[Old]]` and [[Other]]"
	got := RewriteWikilinks(src, "Old", "New")
	if !strings.Contains(got, "[[New]]") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "`[[Old]]`") {
		t.Fatalf("masked rewrite leaked: %q", got)
	}
	if !strings.Contains(got, "[[Other]]") {
		t.Fatalf("unrelated changed: %q", got)
	}
}

func TestRewriteKeepsAliasDisplay(t *testing.T) {
	src := "[[Old|shown]]"
	got := RewriteWikilinks(src, "Old", "New")
	if got != "[[New|shown]]" {
		t.Fatalf("got %q", got)
	}
}

func TestAncestorPaths(t *testing.T) {
	got := AncestorPaths("Tech/Golang/Concurrency")
	want := []string{"tech", "tech/golang", "tech/golang/concurrency"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v", got)
	}
}

func TestNormalizeTitle(t *testing.T) {
	if NormalizeTitle("  Go  ") != "go" {
		t.Fatal("norm")
	}
}

func TestParseDollarCurrencyNotMath(t *testing.T) {
	r := Parse("price $ 12 and [[Card]]")
	if len(r.Links) != 1 {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestParseSelfLinkAllowed(t *testing.T) {
	r := Parse("I am [[Self]] referencing [[Self]]")
	if len(r.Links) != 2 {
		t.Fatalf("got %d", len(r.Links))
	}
}

func TestBuildMaskFenceLengthMustMatch(t *testing.T) {
	src := "````\n[[hid]]\n```\nstill hid [[no]]\n````\n[[ok]]"
	r := Parse(src)
	if len(r.Links) != 1 || r.Links[0].Target != "ok" {
		t.Fatalf("got %+v", r.Links)
	}
}

func TestMaskCoversFenceInterior(t *testing.T) {
	src := "a\n```\n[[x]]\n```\nb"
	mask := BuildMask(src)
	idx := indexOf(src, "[[x]]")
	if idx < 0 || !mask[idx] {
		t.Fatal("fence interior not masked")
	}
}

func TestMaskLeavesProse(t *testing.T) {
	src := "hello [[x]]"
	mask := BuildMask(src)
	idx := indexOf(src, "[[x]]")
	if mask[idx] {
		t.Fatal("prose masked")
	}
}

func TestMaskHTMLComment(t *testing.T) {
	src := "<!-- [[x]] -->"
	mask := BuildMask(src)
	idx := indexOf(src, "[[x]]")
	if !mask[idx] {
		t.Fatal("comment")
	}
}

func TestMaskMathBlock(t *testing.T) {
	src := "$$ [[x]] $$"
	mask := BuildMask(src)
	idx := indexOf(src, "[[x]]")
	if !mask[idx] {
		t.Fatal("math")
	}
}

func TestEscapeDoesNotStartFenceScanCorruption(t *testing.T) {
	src := "\\``` not a fence\n[[Keep]]"
	r := Parse(src)
	if len(r.Links) != 1 {
		t.Fatalf("%+v", r.Links)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestParseTableDriven(t *testing.T) {
	cases := []struct {
		name  string
		src   string
		links int
		tags  int
		first string
	}{
		{"plain", "[[Alpha]]", 1, 0, "Alpha"},
		{"alias-spaces", "[[ Alpha | shown ]]", 1, 0, "Alpha"},
		{"two", "[[A]] [[B]]", 2, 0, "A"},
		{"fence-go", "```go\n[[X]]\n```\n[[Y]]", 1, 0, "Y"},
		{"inline", "x `[[X]]` y [[Y]]", 1, 0, "Y"},
		{"comment", "<!--[[X]]-->[[Y]]", 1, 0, "Y"},
		{"math", "$$[[X]]$$ [[Y]]", 1, 0, "Y"},
		{"escape", `\[[X]] [[Y]]`, 1, 0, "Y"},
		{"tag-only", "hello #tech/go", 0, 1, ""},
		{"heading", "# Title\n#tag", 0, 1, ""},
		{"emoji", "[[🔥 火花]]", 1, 0, "🔥 火花"},
		{"zh", "参见[[认知负荷]]。", 1, 0, "认知负荷"},
		{"dup-offset", "[[A]][[A]]", 2, 0, "A"},
		{"empty-skip", "[[]] [[A]]", 1, 0, "A"},
		{"tilde", "~~~\n[[H]]\n~~~\n[[K]]", 1, 0, "K"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Parse(c.src)
			if len(r.Links) != c.links {
				t.Fatalf("links=%d want %d (%+v)", len(r.Links), c.links, r.Links)
			}
			if len(r.Tags) != c.tags {
				t.Fatalf("tags=%v", r.Tags)
			}
			if c.first != "" && (len(r.Links) == 0 || r.Links[0].Target != c.first) {
				t.Fatalf("first=%v", r.Links)
			}
		})
	}
}

func TestSplitTagPathEmpty(t *testing.T) {
	if SplitTagPath("///") != nil && len(SplitTagPath("///")) != 0 {
		t.Fatal(SplitTagPath("///"))
	}
}

func BenchmarkParse8KB50Links(b *testing.B) {
	var sb strings.Builder
	for i := 0; i < 50; i++ {
		sb.WriteString("paragraph about [[Card-")
		sb.WriteString(itoa(i))
		sb.WriteString("]] and #tech/golang with some filler text to inflate size. ")
	}
	sb.WriteString("```go\n")
	for i := 0; i < 30; i++ {
		sb.WriteString("x := \"[[masked]]\"\n")
	}
	sb.WriteString("```\n")
	for sb.Len() < 8*1024 {
		sb.WriteString("padding-padding-padding ")
	}
	src := sb.String()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := Parse(src)
		if len(r.Links) < 40 {
			b.Fatalf("unexpected links %d", len(r.Links))
		}
	}
}

func TestHelperAPI(t *testing.T) {
	if !ValidTitle("OK") || ValidTitle("") || ValidTitle("a[b]") {
		t.Fatal("valid title")
	}
	if CountUnmaskedWikilinks("```\n[[x]]\n```\n[[y]]") != 1 {
		t.Fatal("count")
	}
	known := map[string]struct{}{"a": {}}
	links := Parse("[[A]] [[B]]").Links
	if !HasDanglingHint(links, known) {
		t.Fatal("dangling hint")
	}
	res, dan := FilterKnown(links, known)
	if len(res) != 1 || len(dan) != 1 {
		t.Fatalf("%d %d", len(res), len(dan))
	}
	if len(UniqueHashtags([]string{"A/B", "a/b", ""})) != 1 {
		t.Fatal("uniq tags")
	}
}

func TestNoteCorpusMasksAndLinks(t *testing.T) {
	notes := []struct {
		name  string
		src   string
		links []string
	}{
		{"daily", "今日回顾 [[Zettelkasten]] 与 [[原子卡片]]。", []string{"Zettelkasten", "原子卡片"}},
		{"code-heavy", "```python\nprint('[[no]]')\n```\n然后看 [[双向链接]]。", []string{"双向链接"}},
		{"math-note", "公式 $$E=[[mc2]]$$ 之外链到 [[认知负荷]]。", []string{"认知负荷"}},
		{"commented", "<!-- draft [[hidden]] --> 发布 [[Readwise 方法]]", []string{"Readwise 方法"}},
		{"alias-note", "参见 [[Go 并发模型|CSP]] 与 [[Worker Pool]]。", []string{"Go 并发模型", "Worker Pool"}},
		{"inline-mix", "不要写 `[[raw]]`，要写 [[结构化日志]]。", []string{"结构化日志"}},
		{"escape-mix", `字面 \[[not]] 与真实 [[Docker 交付]]`, []string{"Docker 交付"}},
		{"long-zh", "深度学习者应把 [[渐进式总结]] 叠在 [[网页剪藏]] 之上。", []string{"渐进式总结", "网页剪藏"}},
		{"self", "本文是 [[种子网络]] 的一部分，也指向 [[知识星空图]]。", []string{"种子网络", "知识星空图"}},
		{"ghost", "预告 [[尚未写完的论文笔记]]。", []string{"尚未写完的论文笔记"}},
		{"multi-fence", "~~~\n[[hid1]]\n~~~\n```\n[[hid2]]\n```\n[[可见]]", []string{"可见"}},
		{"heading-and-tag", "# 标题\n正文 [[路径式标签]] #tech/go", []string{"路径式标签"}},
		{"nested-alias", "[[Obsidian 核心|双链鼻祖]] 对照 [[Zettelkasten]]。", []string{"Obsidian 核心", "Zettelkasten"}},
		{"list", "- [[chi 路由]]\n- [[PostgreSQL 图查询]]\n", []string{"chi 路由", "PostgreSQL 图查询"}},
		{"quote", "> 引用 [[费曼技巧]] 来拆卡片。", []string{"费曼技巧"}},
	}
	for _, n := range notes {
		t.Run(n.name, func(t *testing.T) {
			got := titles(Parse(n.src))
			if len(got) != len(n.links) {
				t.Fatalf("got %v want %v", got, n.links)
			}
			for i := range n.links {
				if got[i] != n.links[i] {
					t.Fatalf("got %v want %v", got, n.links)
				}
			}
		})
	}
}

func TestGeneratedCorpus(t *testing.T) {
	type row struct {
		src   string
		links int
	}
	rows := []row{
		{src: "note 1 see [[Card1]] and #tag1\n```\n[[hidden1]]\n```\n", links: 1},
		{src: "note 2 see [[Card2]] and #tag2\n```\n[[hidden2]]\n```\n", links: 1},
		{src: "note 3 see [[Card3]] and #tag3\n```\n[[hidden3]]\n```\n", links: 1},
		{src: "note 4 see [[Card4]] and #tag4\n```\n[[hidden4]]\n```\n", links: 1},
		{src: "note 5 see [[Card5]] and #tag5\n```\n[[hidden5]]\n```\n", links: 1},
		{src: "note 6 see [[Card6]] and #tag6\n```\n[[hidden6]]\n```\n", links: 1},
		{src: "note 7 see [[Card7]] and #tag0\n```\n[[hidden7]]\n```\n", links: 1},
		{src: "note 8 see [[Card8]] and #tag1\n```\n[[hidden8]]\n```\n", links: 1},
		{src: "note 9 see [[Card9]] and #tag2\n```\n[[hidden9]]\n```\n", links: 1},
		{src: "note 10 see [[Card10]] and #tag3\n```\n[[hidden10]]\n```\n", links: 1},
		{src: "note 11 see [[Card11]] and #tag4\n```\n[[hidden11]]\n```\n", links: 1},
		{src: "note 12 see [[Card12]] and #tag5\n```\n[[hidden12]]\n```\n", links: 1},
		{src: "note 13 see [[Card13]] and #tag6\n```\n[[hidden13]]\n```\n", links: 1},
		{src: "note 14 see [[Card14]] and #tag0\n```\n[[hidden14]]\n```\n", links: 1},
		{src: "note 15 see [[Card15]] and #tag1\n```\n[[hidden15]]\n```\n", links: 1},
		{src: "note 16 see [[Card16]] and #tag2\n```\n[[hidden16]]\n```\n", links: 1},
		{src: "note 17 see [[Card17]] and #tag3\n```\n[[hidden17]]\n```\n", links: 1},
		{src: "note 18 see [[Card18]] and #tag4\n```\n[[hidden18]]\n```\n", links: 1},
		{src: "note 19 see [[Card19]] and #tag5\n```\n[[hidden19]]\n```\n", links: 1},
		{src: "note 20 see [[Card20]] and #tag6\n```\n[[hidden20]]\n```\n", links: 1},
		{src: "note 21 see [[Card21]] and #tag0\n```\n[[hidden21]]\n```\n", links: 1},
		{src: "note 22 see [[Card22]] and #tag1\n```\n[[hidden22]]\n```\n", links: 1},
		{src: "note 23 see [[Card23]] and #tag2\n```\n[[hidden23]]\n```\n", links: 1},
		{src: "note 24 see [[Card24]] and #tag3\n```\n[[hidden24]]\n```\n", links: 1},
		{src: "note 25 see [[Card25]] and #tag4\n```\n[[hidden25]]\n```\n", links: 1},
		{src: "note 26 see [[Card26]] and #tag5\n```\n[[hidden26]]\n```\n", links: 1},
		{src: "note 27 see [[Card27]] and #tag6\n```\n[[hidden27]]\n```\n", links: 1},
		{src: "note 28 see [[Card28]] and #tag0\n```\n[[hidden28]]\n```\n", links: 1},
		{src: "note 29 see [[Card29]] and #tag1\n```\n[[hidden29]]\n```\n", links: 1},
		{src: "note 30 see [[Card30]] and #tag2\n```\n[[hidden30]]\n```\n", links: 1},
		{src: "note 31 see [[Card31]] and #tag3\n```\n[[hidden31]]\n```\n", links: 1},
		{src: "note 32 see [[Card32]] and #tag4\n```\n[[hidden32]]\n```\n", links: 1},
		{src: "note 33 see [[Card33]] and #tag5\n```\n[[hidden33]]\n```\n", links: 1},
		{src: "note 34 see [[Card34]] and #tag6\n```\n[[hidden34]]\n```\n", links: 1},
		{src: "note 35 see [[Card35]] and #tag0\n```\n[[hidden35]]\n```\n", links: 1},
		{src: "note 36 see [[Card36]] and #tag1\n```\n[[hidden36]]\n```\n", links: 1},
		{src: "note 37 see [[Card37]] and #tag2\n```\n[[hidden37]]\n```\n", links: 1},
		{src: "note 38 see [[Card38]] and #tag3\n```\n[[hidden38]]\n```\n", links: 1},
		{src: "note 39 see [[Card39]] and #tag4\n```\n[[hidden39]]\n```\n", links: 1},
		{src: "note 40 see [[Card40]] and #tag5\n```\n[[hidden40]]\n```\n", links: 1},
		{src: "note 41 see [[Card41]] and #tag6\n```\n[[hidden41]]\n```\n", links: 1},
		{src: "note 42 see [[Card42]] and #tag0\n```\n[[hidden42]]\n```\n", links: 1},
		{src: "note 43 see [[Card43]] and #tag1\n```\n[[hidden43]]\n```\n", links: 1},
		{src: "note 44 see [[Card44]] and #tag2\n```\n[[hidden44]]\n```\n", links: 1},
		{src: "note 45 see [[Card45]] and #tag3\n```\n[[hidden45]]\n```\n", links: 1},
		{src: "note 46 see [[Card46]] and #tag4\n```\n[[hidden46]]\n```\n", links: 1},
		{src: "note 47 see [[Card47]] and #tag5\n```\n[[hidden47]]\n```\n", links: 1},
		{src: "note 48 see [[Card48]] and #tag6\n```\n[[hidden48]]\n```\n", links: 1},
		{src: "note 49 see [[Card49]] and #tag0\n```\n[[hidden49]]\n```\n", links: 1},
		{src: "note 50 see [[Card50]] and #tag1\n```\n[[hidden50]]\n```\n", links: 1},
		{src: "note 51 see [[Card51]] and #tag2\n```\n[[hidden51]]\n```\n", links: 1},
		{src: "note 52 see [[Card52]] and #tag3\n```\n[[hidden52]]\n```\n", links: 1},
		{src: "note 53 see [[Card53]] and #tag4\n```\n[[hidden53]]\n```\n", links: 1},
		{src: "note 54 see [[Card54]] and #tag5\n```\n[[hidden54]]\n```\n", links: 1},
		{src: "note 55 see [[Card55]] and #tag6\n```\n[[hidden55]]\n```\n", links: 1},
		{src: "note 56 see [[Card56]] and #tag0\n```\n[[hidden56]]\n```\n", links: 1},
		{src: "note 57 see [[Card57]] and #tag1\n```\n[[hidden57]]\n```\n", links: 1},
		{src: "note 58 see [[Card58]] and #tag2\n```\n[[hidden58]]\n```\n", links: 1},
		{src: "note 59 see [[Card59]] and #tag3\n```\n[[hidden59]]\n```\n", links: 1},
		{src: "note 60 see [[Card60]] and #tag4\n```\n[[hidden60]]\n```\n", links: 1},
		{src: "note 61 see [[Card61]] and #tag5\n```\n[[hidden61]]\n```\n", links: 1},
		{src: "note 62 see [[Card62]] and #tag6\n```\n[[hidden62]]\n```\n", links: 1},
		{src: "note 63 see [[Card63]] and #tag0\n```\n[[hidden63]]\n```\n", links: 1},
		{src: "note 64 see [[Card64]] and #tag1\n```\n[[hidden64]]\n```\n", links: 1},
		{src: "note 65 see [[Card65]] and #tag2\n```\n[[hidden65]]\n```\n", links: 1},
		{src: "note 66 see [[Card66]] and #tag3\n```\n[[hidden66]]\n```\n", links: 1},
		{src: "note 67 see [[Card67]] and #tag4\n```\n[[hidden67]]\n```\n", links: 1},
		{src: "note 68 see [[Card68]] and #tag5\n```\n[[hidden68]]\n```\n", links: 1},
		{src: "note 69 see [[Card69]] and #tag6\n```\n[[hidden69]]\n```\n", links: 1},
		{src: "note 70 see [[Card70]] and #tag0\n```\n[[hidden70]]\n```\n", links: 1},
		{src: "note 71 see [[Card71]] and #tag1\n```\n[[hidden71]]\n```\n", links: 1},
		{src: "note 72 see [[Card72]] and #tag2\n```\n[[hidden72]]\n```\n", links: 1},
		{src: "note 73 see [[Card73]] and #tag3\n```\n[[hidden73]]\n```\n", links: 1},
		{src: "note 74 see [[Card74]] and #tag4\n```\n[[hidden74]]\n```\n", links: 1},
		{src: "note 75 see [[Card75]] and #tag5\n```\n[[hidden75]]\n```\n", links: 1},
		{src: "note 76 see [[Card76]] and #tag6\n```\n[[hidden76]]\n```\n", links: 1},
		{src: "note 77 see [[Card77]] and #tag0\n```\n[[hidden77]]\n```\n", links: 1},
		{src: "note 78 see [[Card78]] and #tag1\n```\n[[hidden78]]\n```\n", links: 1},
		{src: "note 79 see [[Card79]] and #tag2\n```\n[[hidden79]]\n```\n", links: 1},
		{src: "note 80 see [[Card80]] and #tag3\n```\n[[hidden80]]\n```\n", links: 1},
		{src: "para 81 [[T81]] `[[n81]]` #g1\n", links: 1},
		{src: "para 82 [[T82]] `[[n82]]` #g2\n", links: 1},
		{src: "para 83 [[T83]] `[[n83]]` #g3\n", links: 1},
		{src: "para 84 [[T84]] `[[n84]]` #g4\n", links: 1},
		{src: "para 85 [[T85]] `[[n85]]` #g0\n", links: 1},
		{src: "para 86 [[T86]] `[[n86]]` #g1\n", links: 1},
		{src: "para 87 [[T87]] `[[n87]]` #g2\n", links: 1},
		{src: "para 88 [[T88]] `[[n88]]` #g3\n", links: 1},
		{src: "para 89 [[T89]] `[[n89]]` #g4\n", links: 1},
		{src: "para 90 [[T90]] `[[n90]]` #g0\n", links: 1},
		{src: "para 91 [[T91]] `[[n91]]` #g1\n", links: 1},
		{src: "para 92 [[T92]] `[[n92]]` #g2\n", links: 1},
		{src: "para 93 [[T93]] `[[n93]]` #g3\n", links: 1},
		{src: "para 94 [[T94]] `[[n94]]` #g4\n", links: 1},
		{src: "para 95 [[T95]] `[[n95]]` #g0\n", links: 1},
		{src: "para 96 [[T96]] `[[n96]]` #g1\n", links: 1},
		{src: "para 97 [[T97]] `[[n97]]` #g2\n", links: 1},
		{src: "para 98 [[T98]] `[[n98]]` #g3\n", links: 1},
		{src: "para 99 [[T99]] `[[n99]]` #g4\n", links: 1},
		{src: "para 100 [[T100]] `[[n100]]` #g0\n", links: 1},
		{src: "para 101 [[T101]] `[[n101]]` #g1\n", links: 1},
		{src: "para 102 [[T102]] `[[n102]]` #g2\n", links: 1},
		{src: "para 103 [[T103]] `[[n103]]` #g3\n", links: 1},
		{src: "para 104 [[T104]] `[[n104]]` #g4\n", links: 1},
		{src: "para 105 [[T105]] `[[n105]]` #g0\n", links: 1},
		{src: "para 106 [[T106]] `[[n106]]` #g1\n", links: 1},
		{src: "para 107 [[T107]] `[[n107]]` #g2\n", links: 1},
		{src: "para 108 [[T108]] `[[n108]]` #g3\n", links: 1},
		{src: "para 109 [[T109]] `[[n109]]` #g4\n", links: 1},
		{src: "para 110 [[T110]] `[[n110]]` #g0\n", links: 1},
		{src: "para 111 [[T111]] `[[n111]]` #g1\n", links: 1},
		{src: "para 112 [[T112]] `[[n112]]` #g2\n", links: 1},
		{src: "para 113 [[T113]] `[[n113]]` #g3\n", links: 1},
		{src: "para 114 [[T114]] `[[n114]]` #g4\n", links: 1},
		{src: "para 115 [[T115]] `[[n115]]` #g0\n", links: 1},
		{src: "para 116 [[T116]] `[[n116]]` #g1\n", links: 1},
		{src: "para 117 [[T117]] `[[n117]]` #g2\n", links: 1},
		{src: "para 118 [[T118]] `[[n118]]` #g3\n", links: 1},
		{src: "para 119 [[T119]] `[[n119]]` #g4\n", links: 1},
		{src: "para 120 [[T120]] `[[n120]]` #g0\n", links: 1},
		{src: "para 121 [[T121]] `[[n121]]` #g1\n", links: 1},
		{src: "para 122 [[T122]] `[[n122]]` #g2\n", links: 1},
		{src: "para 123 [[T123]] `[[n123]]` #g3\n", links: 1},
		{src: "para 124 [[T124]] `[[n124]]` #g4\n", links: 1},
		{src: "para 125 [[T125]] `[[n125]]` #g0\n", links: 1},
		{src: "para 126 [[T126]] `[[n126]]` #g1\n", links: 1},
		{src: "para 127 [[T127]] `[[n127]]` #g2\n", links: 1},
		{src: "para 128 [[T128]] `[[n128]]` #g3\n", links: 1},
		{src: "para 129 [[T129]] `[[n129]]` #g4\n", links: 1},
		{src: "para 130 [[T130]] `[[n130]]` #g0\n", links: 1},
		{src: "para 131 [[T131]] `[[n131]]` #g1\n", links: 1},
		{src: "para 132 [[T132]] `[[n132]]` #g2\n", links: 1},
		{src: "para 133 [[T133]] `[[n133]]` #g3\n", links: 1},
		{src: "para 134 [[T134]] `[[n134]]` #g4\n", links: 1},
		{src: "para 135 [[T135]] `[[n135]]` #g0\n", links: 1},
		{src: "para 136 [[T136]] `[[n136]]` #g1\n", links: 1},
		{src: "para 137 [[T137]] `[[n137]]` #g2\n", links: 1},
		{src: "para 138 [[T138]] `[[n138]]` #g3\n", links: 1},
		{src: "para 139 [[T139]] `[[n139]]` #g4\n", links: 1},
		{src: "para 140 [[T140]] `[[n140]]` #g0\n", links: 1},
		{src: "para 141 [[T141]] `[[n141]]` #g1\n", links: 1},
		{src: "para 142 [[T142]] `[[n142]]` #g2\n", links: 1},
		{src: "para 143 [[T143]] `[[n143]]` #g3\n", links: 1},
		{src: "para 144 [[T144]] `[[n144]]` #g4\n", links: 1},
		{src: "para 145 [[T145]] `[[n145]]` #g0\n", links: 1},
		{src: "para 146 [[T146]] `[[n146]]` #g1\n", links: 1},
		{src: "para 147 [[T147]] `[[n147]]` #g2\n", links: 1},
		{src: "para 148 [[T148]] `[[n148]]` #g3\n", links: 1},
		{src: "para 149 [[T149]] `[[n149]]` #g4\n", links: 1},
		{src: "para 150 [[T150]] `[[n150]]` #g0\n", links: 1},
		{src: "para 151 [[T151]] `[[n151]]` #g1\n", links: 1},
		{src: "para 152 [[T152]] `[[n152]]` #g2\n", links: 1},
		{src: "para 153 [[T153]] `[[n153]]` #g3\n", links: 1},
		{src: "para 154 [[T154]] `[[n154]]` #g4\n", links: 1},
		{src: "para 155 [[T155]] `[[n155]]` #g0\n", links: 1},
		{src: "para 156 [[T156]] `[[n156]]` #g1\n", links: 1},
		{src: "para 157 [[T157]] `[[n157]]` #g2\n", links: 1},
		{src: "para 158 [[T158]] `[[n158]]` #g3\n", links: 1},
		{src: "para 159 [[T159]] `[[n159]]` #g4\n", links: 1},
		{src: "para 160 [[T160]] `[[n160]]` #g0\n", links: 1},
		{src: "para 161 [[T161]] `[[n161]]` #g1\n", links: 1},
		{src: "para 162 [[T162]] `[[n162]]` #g2\n", links: 1},
		{src: "para 163 [[T163]] `[[n163]]` #g3\n", links: 1},
		{src: "para 164 [[T164]] `[[n164]]` #g4\n", links: 1},
		{src: "para 165 [[T165]] `[[n165]]` #g0\n", links: 1},
		{src: "para 166 [[T166]] `[[n166]]` #g1\n", links: 1},
		{src: "para 167 [[T167]] `[[n167]]` #g2\n", links: 1},
		{src: "para 168 [[T168]] `[[n168]]` #g3\n", links: 1},
		{src: "para 169 [[T169]] `[[n169]]` #g4\n", links: 1},
		{src: "para 170 [[T170]] `[[n170]]` #g0\n", links: 1},
		{src: "para 171 [[T171]] `[[n171]]` #g1\n", links: 1},
		{src: "para 172 [[T172]] `[[n172]]` #g2\n", links: 1},
		{src: "para 173 [[T173]] `[[n173]]` #g3\n", links: 1},
		{src: "para 174 [[T174]] `[[n174]]` #g4\n", links: 1},
		{src: "para 175 [[T175]] `[[n175]]` #g0\n", links: 1},
		{src: "para 176 [[T176]] `[[n176]]` #g1\n", links: 1},
		{src: "para 177 [[T177]] `[[n177]]` #g2\n", links: 1},
		{src: "para 178 [[T178]] `[[n178]]` #g3\n", links: 1},
		{src: "para 179 [[T179]] `[[n179]]` #g4\n", links: 1},
		{src: "para 180 [[T180]] `[[n180]]` #g0\n", links: 1},
		{src: "para 181 [[T181]] `[[n181]]` #g1\n", links: 1},
		{src: "para 182 [[T182]] `[[n182]]` #g2\n", links: 1},
		{src: "para 183 [[T183]] `[[n183]]` #g3\n", links: 1},
		{src: "para 184 [[T184]] `[[n184]]` #g4\n", links: 1},
		{src: "para 185 [[T185]] `[[n185]]` #g0\n", links: 1},
		{src: "para 186 [[T186]] `[[n186]]` #g1\n", links: 1},
		{src: "para 187 [[T187]] `[[n187]]` #g2\n", links: 1},
		{src: "para 188 [[T188]] `[[n188]]` #g3\n", links: 1},
		{src: "para 189 [[T189]] `[[n189]]` #g4\n", links: 1},
		{src: "para 190 [[T190]] `[[n190]]` #g0\n", links: 1},
		{src: "para 191 [[T191]] `[[n191]]` #g1\n", links: 1},
		{src: "para 192 [[T192]] `[[n192]]` #g2\n", links: 1},
		{src: "para 193 [[T193]] `[[n193]]` #g3\n", links: 1},
		{src: "para 194 [[T194]] `[[n194]]` #g4\n", links: 1},
		{src: "para 195 [[T195]] `[[n195]]` #g0\n", links: 1},
		{src: "para 196 [[T196]] `[[n196]]` #g1\n", links: 1},
		{src: "para 197 [[T197]] `[[n197]]` #g2\n", links: 1},
		{src: "para 198 [[T198]] `[[n198]]` #g3\n", links: 1},
		{src: "para 199 [[T199]] `[[n199]]` #g4\n", links: 1},
		{src: "para 200 [[T200]] `[[n200]]` #g0\n", links: 1},
		{src: "para 201 [[T201]] `[[n201]]` #g1\n", links: 1},
		{src: "para 202 [[T202]] `[[n202]]` #g2\n", links: 1},
		{src: "para 203 [[T203]] `[[n203]]` #g3\n", links: 1},
		{src: "para 204 [[T204]] `[[n204]]` #g4\n", links: 1},
		{src: "para 205 [[T205]] `[[n205]]` #g0\n", links: 1},
		{src: "para 206 [[T206]] `[[n206]]` #g1\n", links: 1},
		{src: "para 207 [[T207]] `[[n207]]` #g2\n", links: 1},
		{src: "para 208 [[T208]] `[[n208]]` #g3\n", links: 1},
		{src: "para 209 [[T209]] `[[n209]]` #g4\n", links: 1},
		{src: "para 210 [[T210]] `[[n210]]` #g0\n", links: 1},
		{src: "para 211 [[T211]] `[[n211]]` #g1\n", links: 1},
		{src: "para 212 [[T212]] `[[n212]]` #g2\n", links: 1},
		{src: "para 213 [[T213]] `[[n213]]` #g3\n", links: 1},
		{src: "para 214 [[T214]] `[[n214]]` #g4\n", links: 1},
		{src: "para 215 [[T215]] `[[n215]]` #g0\n", links: 1},
		{src: "para 216 [[T216]] `[[n216]]` #g1\n", links: 1},
		{src: "para 217 [[T217]] `[[n217]]` #g2\n", links: 1},
		{src: "para 218 [[T218]] `[[n218]]` #g3\n", links: 1},
		{src: "para 219 [[T219]] `[[n219]]` #g4\n", links: 1},
		{src: "<!-- [[c220]] --> visible [[V220]]\n", links: 1},
		{src: "<!-- [[c221]] --> visible [[V221]]\n", links: 1},
		{src: "<!-- [[c222]] --> visible [[V222]]\n", links: 1},
		{src: "<!-- [[c223]] --> visible [[V223]]\n", links: 1},
		{src: "<!-- [[c224]] --> visible [[V224]]\n", links: 1},
		{src: "<!-- [[c225]] --> visible [[V225]]\n", links: 1},
		{src: "<!-- [[c226]] --> visible [[V226]]\n", links: 1},
		{src: "<!-- [[c227]] --> visible [[V227]]\n", links: 1},
		{src: "<!-- [[c228]] --> visible [[V228]]\n", links: 1},
		{src: "<!-- [[c229]] --> visible [[V229]]\n", links: 1},
		{src: "<!-- [[c230]] --> visible [[V230]]\n", links: 1},
		{src: "<!-- [[c231]] --> visible [[V231]]\n", links: 1},
		{src: "<!-- [[c232]] --> visible [[V232]]\n", links: 1},
		{src: "<!-- [[c233]] --> visible [[V233]]\n", links: 1},
		{src: "<!-- [[c234]] --> visible [[V234]]\n", links: 1},
		{src: "<!-- [[c235]] --> visible [[V235]]\n", links: 1},
		{src: "<!-- [[c236]] --> visible [[V236]]\n", links: 1},
		{src: "<!-- [[c237]] --> visible [[V237]]\n", links: 1},
		{src: "<!-- [[c238]] --> visible [[V238]]\n", links: 1},
		{src: "<!-- [[c239]] --> visible [[V239]]\n", links: 1},
		{src: "<!-- [[c240]] --> visible [[V240]]\n", links: 1},
		{src: "<!-- [[c241]] --> visible [[V241]]\n", links: 1},
		{src: "<!-- [[c242]] --> visible [[V242]]\n", links: 1},
		{src: "<!-- [[c243]] --> visible [[V243]]\n", links: 1},
		{src: "<!-- [[c244]] --> visible [[V244]]\n", links: 1},
		{src: "<!-- [[c245]] --> visible [[V245]]\n", links: 1},
		{src: "<!-- [[c246]] --> visible [[V246]]\n", links: 1},
		{src: "<!-- [[c247]] --> visible [[V247]]\n", links: 1},
		{src: "<!-- [[c248]] --> visible [[V248]]\n", links: 1},
		{src: "<!-- [[c249]] --> visible [[V249]]\n", links: 1},
		{src: "<!-- [[c250]] --> visible [[V250]]\n", links: 1},
		{src: "<!-- [[c251]] --> visible [[V251]]\n", links: 1},
		{src: "<!-- [[c252]] --> visible [[V252]]\n", links: 1},
		{src: "<!-- [[c253]] --> visible [[V253]]\n", links: 1},
		{src: "<!-- [[c254]] --> visible [[V254]]\n", links: 1},
		{src: "<!-- [[c255]] --> visible [[V255]]\n", links: 1},
		{src: "<!-- [[c256]] --> visible [[V256]]\n", links: 1},
		{src: "<!-- [[c257]] --> visible [[V257]]\n", links: 1},
		{src: "<!-- [[c258]] --> visible [[V258]]\n", links: 1},
		{src: "<!-- [[c259]] --> visible [[V259]]\n", links: 1},
		{src: "<!-- [[c260]] --> visible [[V260]]\n", links: 1},
		{src: "<!-- [[c261]] --> visible [[V261]]\n", links: 1},
		{src: "<!-- [[c262]] --> visible [[V262]]\n", links: 1},
		{src: "<!-- [[c263]] --> visible [[V263]]\n", links: 1},
		{src: "<!-- [[c264]] --> visible [[V264]]\n", links: 1},
		{src: "<!-- [[c265]] --> visible [[V265]]\n", links: 1},
		{src: "<!-- [[c266]] --> visible [[V266]]\n", links: 1},
		{src: "<!-- [[c267]] --> visible [[V267]]\n", links: 1},
		{src: "<!-- [[c268]] --> visible [[V268]]\n", links: 1},
		{src: "<!-- [[c269]] --> visible [[V269]]\n", links: 1},
		{src: "<!-- [[c270]] --> visible [[V270]]\n", links: 1},
		{src: "<!-- [[c271]] --> visible [[V271]]\n", links: 1},
		{src: "<!-- [[c272]] --> visible [[V272]]\n", links: 1},
		{src: "<!-- [[c273]] --> visible [[V273]]\n", links: 1},
		{src: "<!-- [[c274]] --> visible [[V274]]\n", links: 1},
		{src: "<!-- [[c275]] --> visible [[V275]]\n", links: 1},
		{src: "<!-- [[c276]] --> visible [[V276]]\n", links: 1},
		{src: "<!-- [[c277]] --> visible [[V277]]\n", links: 1},
		{src: "<!-- [[c278]] --> visible [[V278]]\n", links: 1},
		{src: "<!-- [[c279]] --> visible [[V279]]\n", links: 1},
		{src: "<!-- [[c280]] --> visible [[V280]]\n", links: 1},
		{src: "<!-- [[c281]] --> visible [[V281]]\n", links: 1},
		{src: "<!-- [[c282]] --> visible [[V282]]\n", links: 1},
		{src: "<!-- [[c283]] --> visible [[V283]]\n", links: 1},
		{src: "<!-- [[c284]] --> visible [[V284]]\n", links: 1},
		{src: "<!-- [[c285]] --> visible [[V285]]\n", links: 1},
		{src: "<!-- [[c286]] --> visible [[V286]]\n", links: 1},
		{src: "<!-- [[c287]] --> visible [[V287]]\n", links: 1},
		{src: "<!-- [[c288]] --> visible [[V288]]\n", links: 1},
		{src: "<!-- [[c289]] --> visible [[V289]]\n", links: 1},
		{src: "<!-- [[c290]] --> visible [[V290]]\n", links: 1},
		{src: "<!-- [[c291]] --> visible [[V291]]\n", links: 1},
		{src: "<!-- [[c292]] --> visible [[V292]]\n", links: 1},
		{src: "<!-- [[c293]] --> visible [[V293]]\n", links: 1},
		{src: "<!-- [[c294]] --> visible [[V294]]\n", links: 1},
		{src: "<!-- [[c295]] --> visible [[V295]]\n", links: 1},
		{src: "<!-- [[c296]] --> visible [[V296]]\n", links: 1},
		{src: "<!-- [[c297]] --> visible [[V297]]\n", links: 1},
		{src: "<!-- [[c298]] --> visible [[V298]]\n", links: 1},
		{src: "<!-- [[c299]] --> visible [[V299]]\n", links: 1},
		{src: "<!-- [[c300]] --> visible [[V300]]\n", links: 1},
		{src: "<!-- [[c301]] --> visible [[V301]]\n", links: 1},
		{src: "<!-- [[c302]] --> visible [[V302]]\n", links: 1},
		{src: "<!-- [[c303]] --> visible [[V303]]\n", links: 1},
		{src: "<!-- [[c304]] --> visible [[V304]]\n", links: 1},
		{src: "<!-- [[c305]] --> visible [[V305]]\n", links: 1},
		{src: "<!-- [[c306]] --> visible [[V306]]\n", links: 1},
		{src: "<!-- [[c307]] --> visible [[V307]]\n", links: 1},
		{src: "<!-- [[c308]] --> visible [[V308]]\n", links: 1},
		{src: "<!-- [[c309]] --> visible [[V309]]\n", links: 1},
		{src: "<!-- [[c310]] --> visible [[V310]]\n", links: 1},
		{src: "<!-- [[c311]] --> visible [[V311]]\n", links: 1},
		{src: "<!-- [[c312]] --> visible [[V312]]\n", links: 1},
		{src: "<!-- [[c313]] --> visible [[V313]]\n", links: 1},
		{src: "<!-- [[c314]] --> visible [[V314]]\n", links: 1},
		{src: "<!-- [[c315]] --> visible [[V315]]\n", links: 1},
		{src: "<!-- [[c316]] --> visible [[V316]]\n", links: 1},
		{src: "<!-- [[c317]] --> visible [[V317]]\n", links: 1},
		{src: "<!-- [[c318]] --> visible [[V318]]\n", links: 1},
		{src: "<!-- [[c319]] --> visible [[V319]]\n", links: 1},
		{src: "<!-- [[c320]] --> visible [[V320]]\n", links: 1},
		{src: "<!-- [[c321]] --> visible [[V321]]\n", links: 1},
		{src: "<!-- [[c322]] --> visible [[V322]]\n", links: 1},
		{src: "<!-- [[c323]] --> visible [[V323]]\n", links: 1},
		{src: "<!-- [[c324]] --> visible [[V324]]\n", links: 1},
		{src: "<!-- [[c325]] --> visible [[V325]]\n", links: 1},
		{src: "<!-- [[c326]] --> visible [[V326]]\n", links: 1},
		{src: "<!-- [[c327]] --> visible [[V327]]\n", links: 1},
		{src: "<!-- [[c328]] --> visible [[V328]]\n", links: 1},
		{src: "<!-- [[c329]] --> visible [[V329]]\n", links: 1},
		{src: "<!-- [[c330]] --> visible [[V330]]\n", links: 1},
		{src: "<!-- [[c331]] --> visible [[V331]]\n", links: 1},
		{src: "<!-- [[c332]] --> visible [[V332]]\n", links: 1},
		{src: "<!-- [[c333]] --> visible [[V333]]\n", links: 1},
		{src: "<!-- [[c334]] --> visible [[V334]]\n", links: 1},
		{src: "<!-- [[c335]] --> visible [[V335]]\n", links: 1},
		{src: "<!-- [[c336]] --> visible [[V336]]\n", links: 1},
		{src: "<!-- [[c337]] --> visible [[V337]]\n", links: 1},
		{src: "<!-- [[c338]] --> visible [[V338]]\n", links: 1},
		{src: "<!-- [[c339]] --> visible [[V339]]\n", links: 1},
		{src: "<!-- [[c340]] --> visible [[V340]]\n", links: 1},
		{src: "<!-- [[c341]] --> visible [[V341]]\n", links: 1},
		{src: "<!-- [[c342]] --> visible [[V342]]\n", links: 1},
		{src: "<!-- [[c343]] --> visible [[V343]]\n", links: 1},
		{src: "<!-- [[c344]] --> visible [[V344]]\n", links: 1},
		{src: "<!-- [[c345]] --> visible [[V345]]\n", links: 1},
		{src: "<!-- [[c346]] --> visible [[V346]]\n", links: 1},
		{src: "<!-- [[c347]] --> visible [[V347]]\n", links: 1},
		{src: "<!-- [[c348]] --> visible [[V348]]\n", links: 1},
		{src: "<!-- [[c349]] --> visible [[V349]]\n", links: 1},
		{src: "<!-- [[c350]] --> visible [[V350]]\n", links: 1},
		{src: "<!-- [[c351]] --> visible [[V351]]\n", links: 1},
		{src: "<!-- [[c352]] --> visible [[V352]]\n", links: 1},
		{src: "<!-- [[c353]] --> visible [[V353]]\n", links: 1},
		{src: "<!-- [[c354]] --> visible [[V354]]\n", links: 1},
		{src: "<!-- [[c355]] --> visible [[V355]]\n", links: 1},
		{src: "<!-- [[c356]] --> visible [[V356]]\n", links: 1},
		{src: "<!-- [[c357]] --> visible [[V357]]\n", links: 1},
		{src: "<!-- [[c358]] --> visible [[V358]]\n", links: 1},
		{src: "<!-- [[c359]] --> visible [[V359]]\n", links: 1},
		{src: "<!-- [[c360]] --> visible [[V360]]\n", links: 1},
		{src: "<!-- [[c361]] --> visible [[V361]]\n", links: 1},
		{src: "<!-- [[c362]] --> visible [[V362]]\n", links: 1},
		{src: "<!-- [[c363]] --> visible [[V363]]\n", links: 1},
		{src: "<!-- [[c364]] --> visible [[V364]]\n", links: 1},
		{src: "<!-- [[c365]] --> visible [[V365]]\n", links: 1},
		{src: "<!-- [[c366]] --> visible [[V366]]\n", links: 1},
		{src: "<!-- [[c367]] --> visible [[V367]]\n", links: 1},
		{src: "<!-- [[c368]] --> visible [[V368]]\n", links: 1},
		{src: "<!-- [[c369]] --> visible [[V369]]\n", links: 1},
		{src: "<!-- [[c370]] --> visible [[V370]]\n", links: 1},
		{src: "<!-- [[c371]] --> visible [[V371]]\n", links: 1},
		{src: "<!-- [[c372]] --> visible [[V372]]\n", links: 1},
		{src: "<!-- [[c373]] --> visible [[V373]]\n", links: 1},
		{src: "<!-- [[c374]] --> visible [[V374]]\n", links: 1},
		{src: "<!-- [[c375]] --> visible [[V375]]\n", links: 1},
		{src: "<!-- [[c376]] --> visible [[V376]]\n", links: 1},
		{src: "<!-- [[c377]] --> visible [[V377]]\n", links: 1},
		{src: "<!-- [[c378]] --> visible [[V378]]\n", links: 1},
		{src: "<!-- [[c379]] --> visible [[V379]]\n", links: 1},
		{src: "<!-- [[c380]] --> visible [[V380]]\n", links: 1},
		{src: "<!-- [[c381]] --> visible [[V381]]\n", links: 1},
		{src: "<!-- [[c382]] --> visible [[V382]]\n", links: 1},
		{src: "<!-- [[c383]] --> visible [[V383]]\n", links: 1},
		{src: "<!-- [[c384]] --> visible [[V384]]\n", links: 1},
		{src: "<!-- [[c385]] --> visible [[V385]]\n", links: 1},
		{src: "<!-- [[c386]] --> visible [[V386]]\n", links: 1},
		{src: "<!-- [[c387]] --> visible [[V387]]\n", links: 1},
		{src: "<!-- [[c388]] --> visible [[V388]]\n", links: 1},
		{src: "<!-- [[c389]] --> visible [[V389]]\n", links: 1},
		{src: "<!-- [[c390]] --> visible [[V390]]\n", links: 1},
		{src: "<!-- [[c391]] --> visible [[V391]]\n", links: 1},
		{src: "<!-- [[c392]] --> visible [[V392]]\n", links: 1},
		{src: "<!-- [[c393]] --> visible [[V393]]\n", links: 1},
		{src: "<!-- [[c394]] --> visible [[V394]]\n", links: 1},
		{src: "<!-- [[c395]] --> visible [[V395]]\n", links: 1},
		{src: "<!-- [[c396]] --> visible [[V396]]\n", links: 1},
		{src: "<!-- [[c397]] --> visible [[V397]]\n", links: 1},
		{src: "<!-- [[c398]] --> visible [[V398]]\n", links: 1},
		{src: "<!-- [[c399]] --> visible [[V399]]\n", links: 1},
		{src: "$$[[m400]]$$ [[K400]]\n", links: 1},
		{src: "$$[[m401]]$$ [[K401]]\n", links: 1},
		{src: "$$[[m402]]$$ [[K402]]\n", links: 1},
		{src: "$$[[m403]]$$ [[K403]]\n", links: 1},
		{src: "$$[[m404]]$$ [[K404]]\n", links: 1},
		{src: "$$[[m405]]$$ [[K405]]\n", links: 1},
		{src: "$$[[m406]]$$ [[K406]]\n", links: 1},
		{src: "$$[[m407]]$$ [[K407]]\n", links: 1},
		{src: "$$[[m408]]$$ [[K408]]\n", links: 1},
		{src: "$$[[m409]]$$ [[K409]]\n", links: 1},
		{src: "$$[[m410]]$$ [[K410]]\n", links: 1},
		{src: "$$[[m411]]$$ [[K411]]\n", links: 1},
		{src: "$$[[m412]]$$ [[K412]]\n", links: 1},
		{src: "$$[[m413]]$$ [[K413]]\n", links: 1},
		{src: "$$[[m414]]$$ [[K414]]\n", links: 1},
		{src: "$$[[m415]]$$ [[K415]]\n", links: 1},
		{src: "$$[[m416]]$$ [[K416]]\n", links: 1},
		{src: "$$[[m417]]$$ [[K417]]\n", links: 1},
		{src: "$$[[m418]]$$ [[K418]]\n", links: 1},
		{src: "$$[[m419]]$$ [[K419]]\n", links: 1},
		{src: "$$[[m420]]$$ [[K420]]\n", links: 1},
		{src: "$$[[m421]]$$ [[K421]]\n", links: 1},
		{src: "$$[[m422]]$$ [[K422]]\n", links: 1},
		{src: "$$[[m423]]$$ [[K423]]\n", links: 1},
		{src: "$$[[m424]]$$ [[K424]]\n", links: 1},
		{src: "$$[[m425]]$$ [[K425]]\n", links: 1},
		{src: "$$[[m426]]$$ [[K426]]\n", links: 1},
		{src: "$$[[m427]]$$ [[K427]]\n", links: 1},
		{src: "$$[[m428]]$$ [[K428]]\n", links: 1},
		{src: "$$[[m429]]$$ [[K429]]\n", links: 1},
		{src: "$$[[m430]]$$ [[K430]]\n", links: 1},
		{src: "$$[[m431]]$$ [[K431]]\n", links: 1},
		{src: "$$[[m432]]$$ [[K432]]\n", links: 1},
		{src: "$$[[m433]]$$ [[K433]]\n", links: 1},
		{src: "$$[[m434]]$$ [[K434]]\n", links: 1},
		{src: "$$[[m435]]$$ [[K435]]\n", links: 1},
		{src: "$$[[m436]]$$ [[K436]]\n", links: 1},
		{src: "$$[[m437]]$$ [[K437]]\n", links: 1},
		{src: "$$[[m438]]$$ [[K438]]\n", links: 1},
		{src: "$$[[m439]]$$ [[K439]]\n", links: 1},
		{src: "$$[[m440]]$$ [[K440]]\n", links: 1},
		{src: "$$[[m441]]$$ [[K441]]\n", links: 1},
		{src: "$$[[m442]]$$ [[K442]]\n", links: 1},
		{src: "$$[[m443]]$$ [[K443]]\n", links: 1},
		{src: "$$[[m444]]$$ [[K444]]\n", links: 1},
		{src: "$$[[m445]]$$ [[K445]]\n", links: 1},
		{src: "$$[[m446]]$$ [[K446]]\n", links: 1},
		{src: "$$[[m447]]$$ [[K447]]\n", links: 1},
		{src: "$$[[m448]]$$ [[K448]]\n", links: 1},
		{src: "$$[[m449]]$$ [[K449]]\n", links: 1},
		{src: "$$[[m450]]$$ [[K450]]\n", links: 1},
		{src: "$$[[m451]]$$ [[K451]]\n", links: 1},
		{src: "$$[[m452]]$$ [[K452]]\n", links: 1},
		{src: "$$[[m453]]$$ [[K453]]\n", links: 1},
		{src: "$$[[m454]]$$ [[K454]]\n", links: 1},
		{src: "$$[[m455]]$$ [[K455]]\n", links: 1},
		{src: "$$[[m456]]$$ [[K456]]\n", links: 1},
		{src: "$$[[m457]]$$ [[K457]]\n", links: 1},
		{src: "$$[[m458]]$$ [[K458]]\n", links: 1},
		{src: "$$[[m459]]$$ [[K459]]\n", links: 1},
		{src: "$$[[m460]]$$ [[K460]]\n", links: 1},
		{src: "$$[[m461]]$$ [[K461]]\n", links: 1},
		{src: "$$[[m462]]$$ [[K462]]\n", links: 1},
		{src: "$$[[m463]]$$ [[K463]]\n", links: 1},
		{src: "$$[[m464]]$$ [[K464]]\n", links: 1},
		{src: "$$[[m465]]$$ [[K465]]\n", links: 1},
		{src: "$$[[m466]]$$ [[K466]]\n", links: 1},
		{src: "$$[[m467]]$$ [[K467]]\n", links: 1},
		{src: "$$[[m468]]$$ [[K468]]\n", links: 1},
		{src: "$$[[m469]]$$ [[K469]]\n", links: 1},
		{src: "$$[[m470]]$$ [[K470]]\n", links: 1},
		{src: "$$[[m471]]$$ [[K471]]\n", links: 1},
		{src: "$$[[m472]]$$ [[K472]]\n", links: 1},
		{src: "$$[[m473]]$$ [[K473]]\n", links: 1},
		{src: "$$[[m474]]$$ [[K474]]\n", links: 1},
		{src: "$$[[m475]]$$ [[K475]]\n", links: 1},
		{src: "$$[[m476]]$$ [[K476]]\n", links: 1},
		{src: "$$[[m477]]$$ [[K477]]\n", links: 1},
		{src: "$$[[m478]]$$ [[K478]]\n", links: 1},
		{src: "$$[[m479]]$$ [[K479]]\n", links: 1},
		{src: "$$[[m480]]$$ [[K480]]\n", links: 1},
		{src: "$$[[m481]]$$ [[K481]]\n", links: 1},
		{src: "$$[[m482]]$$ [[K482]]\n", links: 1},
		{src: "$$[[m483]]$$ [[K483]]\n", links: 1},
		{src: "$$[[m484]]$$ [[K484]]\n", links: 1},
		{src: "$$[[m485]]$$ [[K485]]\n", links: 1},
		{src: "$$[[m486]]$$ [[K486]]\n", links: 1},
		{src: "$$[[m487]]$$ [[K487]]\n", links: 1},
		{src: "$$[[m488]]$$ [[K488]]\n", links: 1},
		{src: "$$[[m489]]$$ [[K489]]\n", links: 1},
	}
	for i, r := range rows {
		got := Parse(r.src)
		if len(got.Links) != r.links {
			t.Fatalf("%d got %d want %d src=%q", i, len(got.Links), r.links, r.src)
		}
	}
}
