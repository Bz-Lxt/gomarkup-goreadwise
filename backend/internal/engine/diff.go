package engine

import (
	"github.com/google/uuid"
	"goreadwise/internal/model"
)

type LinkKey struct {
	TargetNorm string
	Display    string
	Start      int
}

type LinkDiff struct {
	ToInsert  []WikiLink
	ToDelete  []uuid.UUID
	Unchanged int
}

func DiffOutgoing(existing []model.Link, parsed []WikiLink) LinkDiff {
	used := make([]bool, len(existing))
	diff := LinkDiff{}
	for _, p := range parsed {
		matched := false
		pNorm := NormalizeTitle(p.Target)
		for i, e := range existing {
			if used[i] {
				continue
			}
			if NormalizeTitle(e.TargetTitle) == pNorm && e.DisplayText == p.Display && e.OffsetStart == p.OffsetStart {
				used[i] = true
				diff.Unchanged++
				matched = true
				break
			}
		}
		if !matched {
			for i, e := range existing {
				if used[i] {
					continue
				}
				if NormalizeTitle(e.TargetTitle) == pNorm && e.DisplayText == p.Display {
					used[i] = true
					diff.Unchanged++
					matched = true
					break
				}
			}
		}
		if !matched {
			diff.ToInsert = append(diff.ToInsert, p)
		}
	}
	for i, e := range existing {
		if !used[i] {
			diff.ToDelete = append(diff.ToDelete, e.ID)
		}
	}
	return diff
}

func UniqueTargets(links []WikiLink) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(links))
	for _, l := range links {
		n := NormalizeTitle(l.Target)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, l.Target)
	}
	return out
}
