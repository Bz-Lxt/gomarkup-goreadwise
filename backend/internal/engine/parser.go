package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTitleRunes   = 200
	ExcerptRadius   = 48
	MaxHashtagDepth = 8
)

var (
	wikiRe = regexp.MustCompile(`\[\[([^\[\]\|\n]+?)(\|([^\[\]\n]+?))?\]\]`)
	hashRe = regexp.MustCompile(`#([A-Za-z\p{Han}][A-Za-z0-9_\-\p{Han}]*(?:/[A-Za-z\p{Han}][A-Za-z0-9_\-\p{Han}]*)*)`)
)

type WikiLink struct {
	Target      string
	Display     string
	OffsetStart int
	OffsetEnd   int
	Excerpt     string
	Raw         string
}

type ParseResult struct {
	Links       []WikiLink
	Tags        []string
	ContentHash string
}

func Parse(src string) ParseResult {
	mask := BuildMask(src)
	links := scanWikilinks(src, mask)
	tags := scanHashtags(src, mask)
	return ParseResult{
		Links:       links,
		Tags:        tags,
		ContentHash: HashContent(src),
	}
}

func HashContent(src string) string {
	sum := sha256.Sum256([]byte(src))
	return hex.EncodeToString(sum[:])
}

func NormalizeTitle(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}

// BuildMask marks bytes that live inside fenced code, inline code, math, or HTML comments.
// true means "do not parse wikilinks / hashtags here".
func BuildMask(src string) []bool {
	n := len(src)
	mask := make([]bool, n)
	i := 0
	for i < n {
		if src[i] == '\\' && i+1 < n {
			i += 2
			continue
		}
		if consume := tryHTMLComment(src, i, mask); consume > 0 {
			i += consume
			continue
		}
		if consume := tryFence(src, i, mask); consume > 0 {
			i += consume
			continue
		}
		if consume := tryMathBlock(src, i, mask); consume > 0 {
			i += consume
			continue
		}
		if consume := tryInlineCode(src, i, mask); consume > 0 {
			i += consume
			continue
		}
		if consume := tryInlineMath(src, i, mask); consume > 0 {
			i += consume
			continue
		}
		i++
	}
	return mask
}

func tryHTMLComment(src string, i int, mask []bool) int {
	if !hasPrefixAt(src, i, "<!--") {
		return 0
	}
	end := strings.Index(src[i+4:], "-->")
	if end < 0 {
		fill(mask, i, len(src))
		return len(src) - i
	}
	closeAt := i + 4 + end + 3
	fill(mask, i, closeAt)
	return closeAt - i
}

func tryFence(src string, i int, mask []bool) int {
	if !atLineStart(src, i) {
		return 0
	}
	ch := src[i]
	if ch != '`' && ch != '~' {
		return 0
	}
	run := 0
	for i+run < len(src) && src[i+run] == ch {
		run++
	}
	if run < 3 {
		return 0
	}
	nl := strings.IndexByte(src[i+run:], '\n')
	if nl < 0 {
		fill(mask, i, len(src))
		return len(src) - i
	}
	bodyStart := i + run + nl + 1
	j := bodyStart
	for j < len(src) {
		if atLineStart(src, j) {
			k := 0
			for j+k < len(src) && src[j+k] == ch {
				k++
			}
			if k >= run {
				rest := j + k
				for rest < len(src) && src[rest] != '\n' {
					if src[rest] != ' ' && src[rest] != '\t' {
						break
					}
					rest++
				}
				if rest >= len(src) || src[rest] == '\n' {
					end := rest
					if rest < len(src) {
						end = rest + 1
					}
					fill(mask, i, end)
					return end - i
				}
			}
		}
		j++
	}
	fill(mask, i, len(src))
	return len(src) - i
}

func tryMathBlock(src string, i int, mask []bool) int {
	if !hasPrefixAt(src, i, "$$") {
		return 0
	}
	end := strings.Index(src[i+2:], "$$")
	if end < 0 {
		fill(mask, i, len(src))
		return len(src) - i
	}
	closeAt := i + 2 + end + 2
	fill(mask, i, closeAt)
	return closeAt - i
}

func tryInlineCode(src string, i int, mask []bool) int {
	if src[i] != '`' {
		return 0
	}
	run := 0
	for i+run < len(src) && src[i+run] == '`' {
		run++
	}
	if run >= 3 && atLineStart(src, i) {
		return 0
	}
	j := i + run
	for j < len(src) {
		if src[j] == '`' {
			k := 0
			for j+k < len(src) && src[j+k] == '`' {
				k++
			}
			if k == run {
				fill(mask, i, j+k)
				return j + k - i
			}
			j += k
			continue
		}
		if src[j] == '\n' {
			return 0
		}
		j++
	}
	return 0
}

func tryInlineMath(src string, i int, mask []bool) int {
	if src[i] != '$' {
		return 0
	}
	if i+1 < len(src) && src[i+1] == '$' {
		return 0
	}
	if i+1 >= len(src) || isSpaceByte(src[i+1]) {
		return 0
	}
	j := i + 1
	for j < len(src) {
		if src[j] == '\\' && j+1 < len(src) {
			j += 2
			continue
		}
		if src[j] == '$' {
			fill(mask, i, j+1)
			return j + 1 - i
		}
		if src[j] == '\n' {
			return 0
		}
		j++
	}
	return 0
}

func scanWikilinks(src string, mask []bool) []WikiLink {
	out := make([]WikiLink, 0, 8)
	seen := make(map[string]struct{})
	locs := wikiRe.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		if start > 0 && src[start-1] == '\\' {
			continue
		}
		if maskedRange(mask, start, end) {
			continue
		}
		target := strings.TrimSpace(src[loc[2]:loc[3]])
		if target == "" || utf8.RuneCountInString(target) > MaxTitleRunes {
			continue
		}
		display := ""
		if loc[6] >= 0 && loc[7] >= 0 {
			display = strings.TrimSpace(src[loc[6]:loc[7]])
		}
		key := NormalizeTitle(target) + "\x00" + display + "\x00" + itoa(start)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, WikiLink{
			Target:      target,
			Display:     display,
			OffsetStart: start,
			OffsetEnd:   end,
			Excerpt:     excerptAround(src, start, end),
			Raw:         src[start:end],
		})
	}
	return out
}

func scanHashtags(src string, mask []bool) []string {
	uniq := make(map[string]struct{})
	var tags []string
	locs := hashRe.FindAllStringSubmatchIndex(src, -1)
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		if maskedRange(mask, start, end) {
			continue
		}
		if isMarkdownHeading(src, start) {
			continue
		}
		if start > 0 {
			prev, _ := utf8.DecodeLastRuneInString(src[:start])
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
				continue
			}
		}
		path := strings.TrimSpace(src[loc[2]:loc[3]])
		path = strings.Trim(path, "/")
		if path == "" || strings.Count(path, "/") >= MaxHashtagDepth {
			continue
		}
		norm := strings.ToLower(path)
		if _, ok := uniq[norm]; ok {
			continue
		}
		uniq[norm] = struct{}{}
		tags = append(tags, norm)
	}
	return tags
}

func isMarkdownHeading(src string, hashAt int) bool {
	lineStart := hashAt
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	j := lineStart
	for j < hashAt && (src[j] == ' ' || src[j] == '\t') {
		j++
	}
	if j != hashAt {
		return false
	}
	run := 0
	for hashAt+run < len(src) && src[hashAt+run] == '#' {
		run++
	}
	if run < 1 || run > 6 {
		return false
	}
	if hashAt+run >= len(src) {
		return true
	}
	return src[hashAt+run] == ' ' || src[hashAt+run] == '\t'
}

func excerptAround(src string, start, end int) string {
	from := start - ExcerptRadius
	if from < 0 {
		from = 0
	}
	to := end + ExcerptRadius
	if to > len(src) {
		to = len(src)
	}
	chunk := src[from:to]
	chunk = strings.ReplaceAll(chunk, "\n", " ")
	chunk = collapseSpace(chunk)
	return strings.TrimSpace(chunk)
}

func collapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return b.String()
}

func maskedRange(mask []bool, start, end int) bool {
	if start < 0 || end > len(mask) || start >= end {
		return true
	}
	for i := start; i < end; i++ {
		if mask[i] {
			return true
		}
	}
	return false
}

func fill(mask []bool, start, end int) {
	if start < 0 {
		start = 0
	}
	if end > len(mask) {
		end = len(mask)
	}
	for i := start; i < end; i++ {
		mask[i] = true
	}
}

func atLineStart(src string, i int) bool {
	return i == 0 || src[i-1] == '\n'
}

func hasPrefixAt(src string, i int, prefix string) bool {
	return i+len(prefix) <= len(src) && src[i:i+len(prefix)] == prefix
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [16]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func SplitTagPath(path string) []string {
	path = strings.ToLower(strings.Trim(strings.TrimSpace(path), "/"))
	if path == "" {
		return nil
	}
	parts := strings.Split(path, "/")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func AncestorPaths(path string) []string {
	parts := SplitTagPath(path)
	if len(parts) == 0 {
		return nil
	}
	acc := make([]string, 0, len(parts))
	cur := ""
	for i, p := range parts {
		if i == 0 {
			cur = p
		} else {
			cur = cur + "/" + p
		}
		acc = append(acc, cur)
	}
	return acc
}

func RewriteWikilinks(src, oldTitle, newTitle string) string {
	oldNorm := NormalizeTitle(oldTitle)
	mask := BuildMask(src)
	locs := wikiRe.FindAllStringSubmatchIndex(src, -1)
	if len(locs) == 0 {
		return src
	}
	var b strings.Builder
	b.Grow(len(src) + 16)
	cursor := 0
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		b.WriteString(src[cursor:start])
		if start > 0 && src[start-1] == '\\' || maskedRange(mask, start, end) {
			b.WriteString(src[start:end])
			cursor = end
			continue
		}
		target := strings.TrimSpace(src[loc[2]:loc[3]])
		if NormalizeTitle(target) != oldNorm {
			b.WriteString(src[start:end])
			cursor = end
			continue
		}
		display := ""
		if loc[6] >= 0 && loc[7] >= 0 {
			display = src[loc[6]:loc[7]]
		}
		if display != "" {
			b.WriteString("[[")
			b.WriteString(newTitle)
			b.WriteByte('|')
			b.WriteString(display)
			b.WriteString("]]")
		} else {
			b.WriteString("[[")
			b.WriteString(newTitle)
			b.WriteString("]]")
		}
		cursor = end
	}
	b.WriteString(src[cursor:])
	return b.String()
}

func (w WikiLink) Key() string {
	return NormalizeTitle(w.Target) + "|" + w.Display
}

func ValidTitle(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > MaxTitleRunes {
		return false
	}
	return !strings.ContainsAny(title, "[]\n")
}

func CountUnmaskedWikilinks(src string) int {
	return len(Parse(src).Links)
}

func HasDanglingHint(links []WikiLink, known map[string]struct{}) bool {
	for _, l := range links {
		if _, ok := known[NormalizeTitle(l.Target)]; !ok {
			return true
		}
	}
	return false
}

func FilterKnown(links []WikiLink, known map[string]struct{}) (resolved, dangling []WikiLink) {
	for _, l := range links {
		if _, ok := known[NormalizeTitle(l.Target)]; ok {
			resolved = append(resolved, l)
		} else {
			dangling = append(dangling, l)
		}
	}
	return resolved, dangling
}

func UniqueHashtags(tags []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.Trim(strings.TrimSpace(t), "/"))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}
