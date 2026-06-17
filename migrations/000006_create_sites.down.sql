ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS site_id;
DROP TABLE IF EXISTS user_site_roles;
DROP TABLE IF EXISTS sites;
