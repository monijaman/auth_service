CREATE TABLE IF NOT EXISTS sites (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    name       VARCHAR(100) NOT NULL UNIQUE,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    domain     VARCHAR(255) UNIQUE,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_site_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, site_id, role_id)
);

CREATE INDEX idx_user_site_roles_user_site ON user_site_roles (user_id, site_id);

-- Store site context on refresh tokens so rotation preserves it
ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS site_id UUID REFERENCES sites(id) ON DELETE SET NULL;

-- Seed sites
INSERT INTO sites (name, slug, domain) VALUES
  ('Default',      'default',     NULL),
  ('Kossti',       'kossti',      'kossti.com'),
  ('Mortalbook',   'mortalbook',  'mortalbook.com')
ON CONFLICT DO NOTHING;
