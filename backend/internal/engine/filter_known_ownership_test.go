package engine

import "testing"

func TestFilterKnownResultsRemainIndependent(t *testing.T) {
	links := []WikiLink{
		{Target: "Known"},
		{Target: "Missing"},
	}
	resolved, dangling := FilterKnown(links, map[string]struct{}{"known": {}})
	if len(resolved) != 1 || len(dangling) != 1 {
		t.Fatalf("unexpected groups: resolved=%v dangling=%v", resolved, dangling)
	}

	resolved = append(resolved, WikiLink{Target: "Known Later"})
	if dangling[0].Target != "Missing" {
		t.Fatalf("appending to resolved links changed dangling links: got %q, want %q", dangling[0].Target, "Missing")
	}
}
