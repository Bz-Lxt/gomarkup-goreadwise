package clock

import (
	"sync"
	"time"
)

// Location is Asia/Shanghai (GMT+8). All persisted timestamps use this zone.
var (
	once sync.Once
	loc  *time.Location
)

func Zone() *time.Location {
	once.Do(func() {
		var err error
		loc, err = time.LoadLocation("Asia/Shanghai")
		if err != nil {
			loc = time.FixedZone("CST", 8*3600)
		}
	})
	return loc
}

func Now() time.Time {
	return time.Now().In(Zone())
}

func Format(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Zone()).Format("2006-01-02 15:04:05")
}

func FormatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.In(Zone()).Format(time.RFC3339)
}
