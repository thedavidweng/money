CREATE TABLE rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  condition_field TEXT NOT NULL,
  condition_op TEXT NOT NULL,
  condition_value TEXT NOT NULL,
  action_type TEXT NOT NULL,
  action_value TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX idx_rules_priority ON rules(enabled, priority DESC);
