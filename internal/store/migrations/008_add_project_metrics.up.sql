CREATE TABLE IF NOT EXISTS project_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    metric_type_id INTEGER NOT NULL REFERENCES metric_types(id),
    value REAL NOT NULL,
    collected_at TEXT NOT NULL DEFAULT (datetime('now'))
);
