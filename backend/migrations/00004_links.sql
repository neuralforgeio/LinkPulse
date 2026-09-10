-- +goose Up
CREATE TABLE links (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  created_by UUID NOT NULL REFERENCES users(id),
  short_code TEXT NOT NULL UNIQUE,
  destination_url TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  password_hash TEXT,
  expires_at TIMESTAMPTZ,
  max_clicks BIGINT,
  click_count BIGINT NOT NULL DEFAULT 0,
  tags TEXT[] NOT NULL DEFAULT '{}',
  utm_source TEXT,
  utm_medium TEXT,
  utm_campaign TEXT,
  utm_term TEXT,
  utm_content TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_links_tenant_created_at ON links (tenant_id, created_at DESC);
CREATE INDEX idx_links_short_code ON links (short_code);
CREATE INDEX idx_links_tenant_status ON links (tenant_id, status);

-- +goose Down
DROP TABLE IF EXISTS links;
