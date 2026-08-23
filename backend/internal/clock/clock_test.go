package clock

import (
	"strings"
	"testing"
	"time"
)

func TestZoneIsShanghai(t *testing.T) {
	n := Now()
	if n.Location() == nil {
		t.Fatal("nil loc")
	}
	name, off := n.Zone()
	if off != 8*3600 && name != "CST" && !strings.Contains(n.Location().String(), "Shanghai") {
		// accept either named zone or fixed +8
		if off != 8*3600 {
			t.Fatalf("offset=%d name=%s loc=%s", off, name, n.Location())
		}
	}
}

func TestFormatEmpty(t *testing.T) {
	if Format(time.Time{}) != "" || FormatRFC3339(time.Time{}) != "" {
		t.Fatal("zero")
	}
	s := Format(Now())
	if len(s) != 19 {
		t.Fatalf("format %q", s)
	}
}
