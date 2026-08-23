package clip

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"goreadwise/internal/clock"
	"goreadwise/internal/httpx"
)

type Result struct {
	URL       string
	Site      string
	Title     string
	Markdown  string
	ClippedAt time.Time
	Provider  string
}

type Provider interface {
	Clip(ctx context.Context, rawURL string) (Result, error)
}

type MockProvider struct {
	Dir string
}

func (m MockProvider) Clip(_ context.Context, rawURL string) (Result, error) {
	u, err := ParseHTTPURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	name := fixtureName(u.Host, u.Path)
	path := filepath.Join(m.Dir, name)
	body, err := os.ReadFile(path)
	if err != nil {
		path = filepath.Join(m.Dir, "default.html")
		body, err = os.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("%w: mock fixture missing", httpx.ErrValidation)
		}
	}
	if !utf8.Valid(body) {
		return Result{}, fmt.Errorf("%w: fixture is not valid utf-8", httpx.ErrValidation)
	}
	title, md, site := Extract(string(body), u.Host)
	return Result{
		URL: u.String(), Site: site, Title: title, Markdown: md,
		ClippedAt: clock.Now(), Provider: "mock",
	}, nil
}

type RealProvider struct {
	Client   *http.Client
	MaxBytes int64
}

func NewReal(timeout time.Duration, maxBytes int64) RealProvider {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxBytes <= 0 {
		maxBytes = 5 << 20
	}
	return RealProvider{
		MaxBytes: maxBytes,
		Client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("%w: too many redirects", httpx.ErrDenied)
				}
				if _, err := ValidatePublicURL(req.URL.String()); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

func (r RealProvider) Clip(ctx context.Context, rawURL string) (Result, error) {
	u, err := ValidatePublicURL(rawURL)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "GoReadwise/1.0 (+https://localhost; research clipper)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	checkRedirect := r.Client.CheckRedirect
	if checkRedirect != nil {
		r.Client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return checkRedirect(req, via)
		}
		defer func() {
			r.Client.CheckRedirect = checkRedirect
		}()
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Result{}, fmt.Errorf("%w: upstream status %d", httpx.ErrValidation, resp.StatusCode)
	}
	limited := io.LimitReader(resp.Body, r.MaxBytes+1)
	raw, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}
	if int64(len(raw)) > r.MaxBytes {
		return Result{}, fmt.Errorf("%w: response exceeds size limit", httpx.ErrValidation)
	}
	if !utf8.Valid(raw) {
		return Result{}, fmt.Errorf("%w: response is not valid utf-8", httpx.ErrValidation)
	}
	title, md, site := Extract(string(raw), u.Hostname())
	return Result{
		URL: u.String(), Site: site, Title: title, Markdown: md,
		ClippedAt: clock.Now(), Provider: "real",
	}, nil
}

func fixtureName(host, path string) string {
	host = strings.ToLower(host)
	switch {
	case strings.Contains(host, "blog"):
		return "tech-blog.html"
	case strings.Contains(host, "arxiv") || strings.Contains(path, "paper"):
		return "paper.html"
	case strings.Contains(host, "doc") || strings.Contains(path, "docs"):
		return "docs.html"
	default:
		return "default.html"
	}
}

func Extract(htmlDoc, host string) (title, markdown, site string) {
	site = host
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlDoc))
	if err != nil {
		return "Untitled clip", fallbackText(htmlDoc), host
	}
	title = strings.TrimSpace(doc.Find("title").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1").First().Text())
	}
	if title == "" {
		title = "Untitled clip"
	}
	if n := []rune(title); len(n) > 180 {
		title = string(n[:180])
	}
	root := doc.Find("article").First()
	if root.Length() == 0 {
		root = doc.Find("main").First()
	}
	if root.Length() == 0 {
		root = doc.Find("body")
	}
	root.Find("script,style,nav,footer,noscript").Remove()
	md := strings.TrimSpace(ToMarkdown(root))
	if md == "" {
		md = strings.TrimSpace(root.Text())
	}
	if md == "" {
		md = fallbackText(htmlDoc)
	}
	return title, md, site
}

func fallbackText(raw string) string {
	var buf bytes.Buffer
	z := html.NewTokenizer(strings.NewReader(raw))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.TextToken {
			t := strings.TrimSpace(string(z.Text()))
			if t != "" {
				buf.WriteString(t)
				buf.WriteByte('\n')
			}
		}
	}
	out := strings.TrimSpace(buf.String())
	if out == "" {
		return raw
	}
	return out
}
