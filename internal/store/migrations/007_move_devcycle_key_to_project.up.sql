ALTER TABLE projects ADD COLUMN devcycle_project_key TEXT NOT NULL DEFAULT '';
ALTER TABLE repositories DROP COLUMN devcycle_project_key;
