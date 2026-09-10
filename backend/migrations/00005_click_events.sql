-- +goose Up
CREATE TABLE click_events (
  id UUID PRIMARY KEY,
  link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  short_code TEXT NOT NULL,
  clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  request_id TEXT NOT NULL,
  ip_hash TEXT NOT NULL,
  user_agent_raw TEXT,
  device_type TEXT,
  browser TEXT,
  os TEXT,
  referrer TEXT,
  source TEXT,
  medium TEXT,
  campaign TEXT,
  country_code TEXT
);

CREATE INDEX idx_click_events_link_clicked_at ON click_events (link_id, clicked_at DESC);
CREATE INDEX idx_click_events_tenant_clicked_at ON click_events (tenant_id, clicked_at DESC);

-- +goose Down
DROP TABLE IF EXISTS click_events;
