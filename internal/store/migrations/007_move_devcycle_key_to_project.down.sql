ALTER TABLE repositories ADD COLUMN devcycle_project_key TEXT NOT NULL DEFAULT '';
ALTER TABLE projects DROP COLUMN devcycle_project_key;
