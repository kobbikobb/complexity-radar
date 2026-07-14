ALTER TABLE repositories ADD COLUMN include_prereleases INTEGER NOT NULL DEFAULT 0;
ALTER TABLE repositories ADD COLUMN release_tag_prefix TEXT NOT NULL DEFAULT '';
