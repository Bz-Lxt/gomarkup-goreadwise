package clip

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

// ToMarkdown converts a goquery selection into a conservative Markdown subset
// used by the clip pipeline. It is intentionally not a full CommonMark writer.
func ToMarkdown(root *goquery.Selection) string {
	if root == nil || root.Length() == 0 {
		return ""
	}
	var b strings.Builder
	root.Contents().Each(func(_ int, s *goquery.Selection) {
		renderSel(&b, s, 0)
	})
	return collapseBlankLines(strings.TrimSpace(b.String()))
}

func renderSel(b *strings.Builder, s *goquery.Selection, depth int) {
	n := s.Get(0)
	if n == nil || depth > 32 {
		return
	}
	switch n.Type {
	case html.TextNode:
		t := collapseInlineSpace(n.Data)
		if t != "" {
			b.WriteString(t)
		}
	case html.ElementNode:
		name := strings.ToLower(n.Data)
		switch name {
		case "script", "style", "nav", "footer", "noscript", "svg", "iframe":
			return
		case "br":
			b.WriteByte('\n')
		case "hr":
			b.WriteString("\n\n---\n\n")
		case "h1", "h2", "h3", "h4", "h5", "h6":
			level := int(name[1] - '0')
			b.WriteString("\n\n")
			b.WriteString(strings.Repeat("#", level))
			b.WriteByte(' ')
			b.WriteString(strings.TrimSpace(s.Text()))
			b.WriteString("\n\n")
		case "p":
			b.WriteString("\n\n")
			s.Contents().Each(func(_ int, c *goquery.Selection) { renderSel(b, c, depth+1) })
			b.WriteString("\n\n")
		case "blockquote":
			inner := strings.TrimSpace(s.Text())
			for _, line := range strings.Split(inner, "\n") {
				b.WriteString("\n> ")
				b.WriteString(strings.TrimSpace(line))
			}
			b.WriteString("\n\n")
		case "ul", "ol":
			b.WriteByte('\n')
			idx := 1
			s.ChildrenFiltered("li").Each(func(_ int, li *goquery.Selection) {
				if name == "ol" {
					fmt.Fprintf(b, "%d. ", idx)
					idx++
				} else {
					b.WriteString("- ")
				}
				b.WriteString(strings.TrimSpace(li.Text()))
				b.WriteByte('\n')
			})
			b.WriteByte('\n')
		case "pre":
			b.WriteString("\n\n```\n")
			b.WriteString(s.Text())
			if !strings.HasSuffix(s.Text(), "\n") {
				b.WriteByte('\n')
			}
			b.WriteString("```\n\n")
		case "code":
			if s.Parent().Length() > 0 && strings.EqualFold(goquery.NodeName(s.Parent()), "pre") {
				return
			}
			b.WriteByte('`')
			b.WriteString(strings.TrimSpace(s.Text()))
			b.WriteByte('`')
		case "strong", "b":
			b.WriteString("**")
			s.Contents().Each(func(_ int, c *goquery.Selection) { renderSel(b, c, depth+1) })
			b.WriteString("**")
		case "em", "i":
			b.WriteByte('_')
			s.Contents().Each(func(_ int, c *goquery.Selection) { renderSel(b, c, depth+1) })
			b.WriteByte('_')
		case "a":
			href, _ := s.Attr("href")
			text := strings.TrimSpace(s.Text())
			if href != "" && text != "" {
				fmt.Fprintf(b, "[%s](%s)", text, href)
			} else {
				b.WriteString(text)
			}
		case "img":
			alt, _ := s.Attr("alt")
			src, _ := s.Attr("src")
			if src != "" {
				fmt.Fprintf(b, "![%s](%s)", alt, src)
			}
		default:
			s.Contents().Each(func(_ int, c *goquery.Selection) { renderSel(b, c, depth+1) })
		}
	}
}

func collapseInlineSpace(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	if strings.TrimSpace(s) == "" {
		if strings.Contains(s, "\n") {
			return ""
		}
		return " "
	}
	return strings.Join(strings.Fields(s), " ")
}

func collapseBlankLines(s string) string {
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}
	return s
}
