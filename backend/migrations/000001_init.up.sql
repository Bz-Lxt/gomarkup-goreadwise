CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS cards (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    title_norm    TEXT NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    content_hash  TEXT NOT NULL DEFAULT '',
    source_url    TEXT,
    source_site   TEXT,
    clipped_at    TIMESTAMPTZ,
    pos_x         DOUBLE PRECISION,
    pos_y         DOUBLE PRECISION,
    graph_version BIGINT NOT NULL DEFAULT 0,
    deleted_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS cards_title_norm_alive
    ON cards (title_norm) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS cards_updated_at ON cards (updated_at DESC);
CREATE INDEX IF NOT EXISTS cards_deleted_at ON cards (deleted_at);

CREATE TABLE IF NOT EXISTS card_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    target_card_id UUID REFERENCES cards(id) ON DELETE SET NULL,
    target_title   TEXT NOT NULL,
    display_text   TEXT NOT NULL DEFAULT '',
    offset_start   INTEGER NOT NULL DEFAULT 0,
    offset_end     INTEGER NOT NULL DEFAULT 0,
    excerpt        TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS card_links_source ON card_links (source_card_id);
CREATE INDEX IF NOT EXISTS card_links_target ON card_links (target_card_id);
CREATE INDEX IF NOT EXISTS card_links_target_title ON card_links (lower(target_title));

CREATE TABLE IF NOT EXISTS tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    full_path  TEXT NOT NULL UNIQUE,
    parent_id  UUID REFERENCES tags(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS tags_parent ON tags (parent_id);
CREATE INDEX IF NOT EXISTS tags_path_prefix ON tags (full_path text_pattern_ops);

CREATE TABLE IF NOT EXISTS card_tags (
    card_id UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (card_id, tag_id)
);

CREATE TABLE IF NOT EXISTS graph_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id      UUID NOT NULL,
    content_hash TEXT NOT NULL,
    kind         TEXT NOT NULL,
    status       TEXT NOT NULL,
    attempts     INTEGER NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TIMESTAMPTZ NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL,
    UNIQUE (card_id, content_hash, kind)
);

CREATE INDEX IF NOT EXISTS graph_jobs_status ON graph_jobs (status, created_at);

CREATE TABLE IF NOT EXISTS graph_meta (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS op_logs (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kind       TEXT NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS op_logs_kind ON op_logs (kind, created_at DESC);
