-- ── Roles ─────────────────────────────────────────────────────────────────────
INSERT INTO roles (name) VALUES
  ('super_admin'),
  ('admin'),
  ('manager'),
  ('editor'),
  ('author'),
  ('moderator'),
  ('staff'),
  ('user')
ON CONFLICT (name) DO NOTHING;

-- ── Permissions ───────────────────────────────────────────────────────────────
INSERT INTO permissions (name) VALUES
  -- User management
  ('users.view'),
  ('users.create'),
  ('users.edit'),
  ('users.delete'),
  ('users.ban'),
  -- Role management
  ('roles.view'),
  ('roles.assign'),
  ('roles.manage'),
  -- Content
  ('content.view'),
  ('content.create'),
  ('content.edit'),
  ('content.delete'),
  ('content.publish'),
  -- Comments / moderation
  ('comments.view'),
  ('comments.approve'),
  ('comments.delete'),
  -- Reports & settings
  ('reports.view'),
  ('settings.view'),
  ('settings.edit'),
  -- Orders
  ('orders.view'),
  ('orders.manage')
ON CONFLICT (name) DO NOTHING;

-- ── Role → Permission assignments ─────────────────────────────────────────────

-- super_admin: everything
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.name = 'super_admin'
ON CONFLICT DO NOTHING;

-- admin: all except roles.manage and settings.edit (those stay with super_admin)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'users.view','users.create','users.edit','users.delete','users.ban',
  'roles.view','roles.assign',
  'content.view','content.create','content.edit','content.delete','content.publish',
  'comments.view','comments.approve','comments.delete',
  'reports.view',
  'settings.view',
  'orders.view','orders.manage'
)
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- manager: assigned resources, reports, content view, orders
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'users.view',
  'content.view','content.edit',
  'comments.view','comments.approve',
  'reports.view',
  'orders.view','orders.manage'
)
WHERE r.name = 'manager'
ON CONFLICT DO NOTHING;

-- editor: create, edit, publish any content
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'content.view','content.create','content.edit','content.publish',
  'comments.view'
)
WHERE r.name = 'editor'
ON CONFLICT DO NOTHING;

-- author: create and edit only their own content (enforced in app logic)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'content.view','content.create','content.edit',
  'comments.view'
)
WHERE r.name = 'author'
ON CONFLICT DO NOTHING;

-- moderator: approve/delete comments, ban users
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'users.view','users.ban',
  'content.view',
  'comments.view','comments.approve','comments.delete'
)
WHERE r.name = 'moderator'
ON CONFLICT DO NOTHING;

-- staff: view assigned modules and orders
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'content.view',
  'comments.view',
  'orders.view'
)
WHERE r.name = 'staff'
ON CONFLICT DO NOTHING;

-- user: view content and own orders/profile
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r
JOIN permissions p ON p.name IN (
  'content.view',
  'comments.view',
  'orders.view'
)
WHERE r.name = 'user'
ON CONFLICT DO NOTHING;
