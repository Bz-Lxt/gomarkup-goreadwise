package store

import (
	"context"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"goreadwise/internal/clock"
	"goreadwise/internal/engine"
	"goreadwise/internal/model"
)

func (d *DB) IncrGlobalVersion(ctx context.Context, tx pgx.Tx) (int64, error) {
	now := clock.Now()
	var q interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}
	if tx != nil {
		q = tx
	} else {
		q = d.Pool
	}
	var raw string
	err := q.QueryRow(ctx, `
		INSERT INTO graph_meta(key, value, updated_at) VALUES ('graph_version','1',$1)
		ON CONFLICT (key) DO UPDATE SET
			value = (COALESCE(NULLIF(graph_meta.value,''),'0')::bigint + 1)::text,
			updated_at = EXCLUDED.updated_at
		RETURNING value`, now).Scan(&raw)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (d *DB) GlobalVersion(ctx context.Context) (int64, error) {
	var raw string
	err := d.Pool.QueryRow(ctx, `SELECT value FROM graph_meta WHERE key='graph_version'`).Scan(&raw)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseInt(raw, 10, 64)
}

func (d *DB) LoadGraph(ctx context.Context) (model.Graph, error) {
	ver, _ := d.GlobalVersion(ctx)
	g := model.Graph{Version: ver, Nodes: []model.GraphNode{}, Edges: []model.GraphEdge{}}

	rows, err := d.Pool.Query(ctx, `
		SELECT c.id, c.title, c.pos_x, c.pos_y,
		       COALESCE((
		         SELECT COUNT(*) FROM card_links l
		         JOIN cards s ON s.id=l.source_card_id AND s.deleted_at IS NULL
		         WHERE l.source_card_id=c.id OR l.target_card_id=c.id
		       ),0) AS degree
		FROM cards c WHERE c.deleted_at IS NULL`)
	if err != nil {
		return g, err
	}
	defer rows.Close()

	nodes := map[string]model.GraphNode{}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		var title string
		var x, y *float64
		var degree int
		if err := rows.Scan(&id, &title, &x, &y, &degree); err != nil {
			return g, err
		}
		sid := id.String()
		nodes[sid] = model.GraphNode{
			ID: sid, CardID: &sid, Title: title, Degree: degree,
			Dangling: false, Orphan: degree == 0, X: x, Y: y, Tags: []string{},
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return g, err
	}

	tagMap, err := d.ListCardTagPaths(ctx, ids)
	if err != nil {
		return g, err
	}
	for id, paths := range tagMap {
		n := nodes[id.String()]
		n.Tags = paths
		nodes[id.String()] = n
	}

	erows, err := d.Pool.Query(ctx, `
		SELECT l.id, l.source_card_id, l.target_card_id, l.target_title
		FROM card_links l
		JOIN cards s ON s.id=l.source_card_id AND s.deleted_at IS NULL`)
	if err != nil {
		return g, err
	}
	defer erows.Close()
	for erows.Next() {
		var lid, src uuid.UUID
		var tgt *uuid.UUID
		var title string
		if err := erows.Scan(&lid, &src, &tgt, &title); err != nil {
			return g, err
		}
		targetID := "ghost:" + engine.NormalizeTitle(title)
		dangling := true
		if tgt != nil {
			targetID = tgt.String()
			dangling = false
		} else if _, ok := nodes[targetID]; !ok {
			nodes[targetID] = model.GraphNode{
				ID: targetID, Title: title, Degree: 1, Dangling: true, Orphan: false, Tags: []string{},
			}
		} else {
			n := nodes[targetID]
			n.Degree++
			nodes[targetID] = n
		}
		g.Edges = append(g.Edges, model.GraphEdge{
			ID: lid.String(), Source: src.String(), Target: targetID, Dangling: dangling,
		})
	}
	if err := erows.Err(); err != nil {
		return g, err
	}
	g.Nodes = make([]model.GraphNode, 0, len(nodes))
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n)
	}
	AnnotateGraph(&g)
	return g, nil
}

func (d *DB) LoadSubgraph(ctx context.Context, root uuid.UUID, depth int) (model.Graph, error) {
	if depth < 1 {
		depth = 1
	}
	if depth > 3 {
		depth = 3
	}
	full, err := d.LoadGraph(ctx)
	if err != nil {
		return full, err
	}
	keep := map[string]struct{}{root.String(): {}}
	frontier := []string{root.String()}
	adj := map[string][]string{}
	for _, e := range full.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source)
	}
	for dpt := 0; dpt < depth; dpt++ {
		var next []string
		for _, id := range frontier {
			for _, nb := range adj[id] {
				if _, ok := keep[nb]; ok {
					continue
				}
				keep[nb] = struct{}{}
				next = append(next, nb)
			}
		}
		frontier = next
	}
	out := model.Graph{Version: full.Version}
	for _, n := range full.Nodes {
		if _, ok := keep[n.ID]; ok {
			out.Nodes = append(out.Nodes, n)
		}
	}
	for _, e := range full.Edges {
		if _, ok := keep[e.Source]; !ok {
			continue
		}
		if _, ok := keep[e.Target]; !ok {
			continue
		}
		out.Edges = append(out.Edges, e)
	}
	if out.Nodes == nil {
		out.Nodes = []model.GraphNode{}
	}
	if out.Edges == nil {
		out.Edges = []model.GraphEdge{}
	}
	return out, nil
}

// AnnotateGraph fills orphan flags after a load. Degree is already computed.
func AnnotateGraph(g *model.Graph) {
	if g == nil {
		return
	}
	deg := map[string]int{}
	for _, e := range g.Edges {
		deg[e.Source]++
		deg[e.Target]++
	}
	for i := range g.Nodes {
		d := deg[g.Nodes[i].ID]
		g.Nodes[i].Degree = d
		g.Nodes[i].Orphan = d == 0 && !g.Nodes[i].Dangling
	}
}

func IsolatedCount(g model.Graph) int {
	n := 0
	for _, node := range g.Nodes {
		if node.Orphan {
			n++
		}
	}
	return n
}

func DanglingCount(g model.Graph) int {
	n := 0
	for _, node := range g.Nodes {
		if node.Dangling {
			n++
		}
	}
	return n
}
