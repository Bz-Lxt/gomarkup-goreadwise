package engine

import (
	"testing"

	"github.com/google/uuid"
	"goreadwise/internal/model"
)

func TestDiffOutgoingInsertDelete(t *testing.T) {
	idKeep := uuid.New()
	idDrop := uuid.New()
	existing := []model.Link{
		{ID: idKeep, TargetTitle: "Keep", DisplayText: "", OffsetStart: 0},
		{ID: idDrop, TargetTitle: "Drop", DisplayText: "", OffsetStart: 10},
	}
	parsed := []WikiLink{
		{Target: "Keep", Display: "", OffsetStart: 0},
		{Target: "New", Display: "n", OffsetStart: 20},
	}
	d := DiffOutgoing(existing, parsed)
	if d.Unchanged != 1 || len(d.ToInsert) != 1 || len(d.ToDelete) != 1 {
		t.Fatalf("%+v", d)
	}
	if d.ToInsert[0].Target != "New" || d.ToDelete[0] != idDrop {
		t.Fatalf("%+v", d)
	}
}

func TestDiffOutgoingEmpty(t *testing.T) {
	d := DiffOutgoing(nil, nil)
	if d.Unchanged != 0 || len(d.ToInsert) != 0 || len(d.ToDelete) != 0 {
		t.Fatalf("%+v", d)
	}
}

func TestUniqueTargets(t *testing.T) {
	got := UniqueTargets([]WikiLink{{Target: "A"}, {Target: "a"}, {Target: "B"}})
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
}
