CREATE TABLE IF NOT EXISTS crawled_items (
    id         BIGSERIAL PRIMARY KEY,
    url        TEXT        NOT NULL,
    data       JSONB,
    metadata   JSONB,
    crawled_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_crawled_items_url        ON crawled_items (url);
CREATE INDEX IF NOT EXISTS idx_crawled_items_crawled_at ON crawled_items (crawled_at);
