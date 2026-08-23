package service

import (
	"strings"
	"testing"

	"goreadwise/internal/engine"
)

func TestMergeTagPaths(t *testing.T) {
	got := mergeTagPaths([]string{"Tech/Go", " tech/go "}, []string{"inbox/clip", "Tech/Go"})
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
}

func TestLeafName(t *testing.T) {
	if leafName("a/b/c") != "c" || leafName("solo") != "solo" {
		t.Fatal("leaf")
	}
}

func TestParseThenDiffIdempotent(t *testing.T) {
	src := "See [[Alpha]] and [[Beta|b]]"
	p1 := engine.Parse(src)
	p2 := engine.Parse(src)
	d := engine.DiffOutgoing(nil, p1.Links)
	if len(d.ToInsert) != 2 {
		t.Fatalf("%+v", d)
	}
	if p1.ContentHash != p2.ContentHash {
		t.Fatal("hash drift")
	}
}

func TestRewriteThenParse(t *testing.T) {
	src := "link [[Old Name]] and `[[Old Name]]`"
	out := engine.RewriteWikilinks(src, "Old Name", "New Name")
	r := engine.Parse(out)
	if len(r.Links) != 1 || r.Links[0].Target != "New Name" {
		t.Fatalf("%+v", r.Links)
	}
	if !strings.Contains(out, "`[[Old Name]]`") {
		t.Fatal("code mutated")
	}
}
