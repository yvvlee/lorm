CREATE TABLE bench_users (
  id BIGSERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  alias TEXT DEFAULT NULL,
  age INTEGER NOT NULL,
  age_p INTEGER DEFAULT NULL,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  active_p BOOLEAN DEFAULT NULL,
  email TEXT NOT NULL UNIQUE,
  tags JSONB NOT NULL DEFAULT '[]'::jsonb,
  meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  profile JSONB NOT NULL DEFAULT '{}'::jsonb,
  contacts JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_bench_users_name ON bench_users(name);
