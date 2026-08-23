package clip

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestValidatePublicURLRejectsLocal(t *testing.T) {
	bads := []string{
		"http://127.0.0.1/x",
		"http://localhost/x",
		"http://10.0.0.1/x",
		"http://192.168.1.5/x",
		"file:///etc/passwd",
		"ftp://example.com/a",
		"http://169.254.169.254/latest",
	}
	for _, u := range bads {
		if _, err := ValidatePublicURL(u); err == nil {
			t.Fatalf("expected deny for %s", u)
		}
	}
}

func TestValidatePublicURLAcceptsHTTPS(t *testing.T) {
	_, err := ValidatePublicURL("https://example.com/post")
	if err != nil {
		t.Log("lookup-dependent:", err)
	}
	if _, err := ValidatePublicURL("https://"); err == nil {
		t.Fatal("missing host")
	}
}

func TestExtractTitleAndMarkdown(t *testing.T) {
	html := `<html><head><title>Demo</title></head><body><article><h1>Hello</h1><p>World [[Link]]</p></article></body></html>`
	title, md, site := Extract(html, "example.com")
	if title != "Demo" || site != "example.com" {
		t.Fatalf("title=%s site=%s", title, site)
	}
	if !contains(md, "World") {
		t.Fatalf("md=%q", md)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && (stringIndex(s, sub) >= 0)))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestToMarkdownHeadingsAndList(t *testing.T) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(`
		<article>
		  <h2>Title</h2>
		  <p>Hello <strong>Go</strong></p>
		  <ul><li>a</li><li>b</li></ul>
		  <pre><code>x := 1</code></pre>
		</article>`))
	if err != nil {
		t.Fatal(err)
	}
	md := ToMarkdown(doc.Find("article"))
	if !strings.Contains(md, "## Title") || !strings.Contains(md, "**Go**") || !strings.Contains(md, "- a") {
		t.Fatalf("%q", md)
	}
	if !strings.Contains(md, "```") {
		t.Fatalf("code fence missing: %q", md)
	}
}

func TestExtractUsesArticle(t *testing.T) {
	title, md, site := Extract(`<html><head><title>T</title></head><body><nav>skip</nav><article><p>Keep [[A]]</p></article></body></html>`, "ex.com")
	if title != "T" || site != "ex.com" || !strings.Contains(md, "Keep") {
		t.Fatalf("%s %s %q", title, site, md)
	}
}
