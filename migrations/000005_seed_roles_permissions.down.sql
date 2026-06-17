DELETE FROM role_permissions;
DELETE FROM permissions;
DELETE FROM roles WHERE name IN (
  'super_admin','admin','manager','editor','author','moderator','staff','user'
);
