CREATE TABLE users (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE user_rsns (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  rsn          TEXT NOT NULL,
  skill_levels JSONB NOT NULL DEFAULT '{}',
  synced_at    TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (user_id, rsn)
);

CREATE TABLE catalog_goals (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  canonical_key TEXT UNIQUE NOT NULL,
  title         TEXT NOT NULL,
  type          TEXT NOT NULL,
  description   TEXT,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE catalog_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  description     TEXT NOT NULL,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE catalog_skill_requirements (
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  skill           TEXT NOT NULL,
  level           INT  NOT NULL,
  PRIMARY KEY (catalog_goal_id, skill)
);

CREATE TABLE catalog_item_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  item_name       TEXT NOT NULL,
  quantity        INT  NOT NULL DEFAULT 1
);

CREATE TABLE catalog_boss_requirements (
  id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  catalog_goal_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  boss_name       TEXT NOT NULL,
  kc              INT  NOT NULL
);

CREATE TABLE catalog_goal_prerequisites (
  goal_id   UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  prereq_id UUID NOT NULL REFERENCES catalog_goals(id) ON DELETE CASCADE,
  PRIMARY KEY (goal_id, prereq_id)
);

CREATE TABLE goals (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  rsn_id       UUID NOT NULL REFERENCES user_rsns(id) ON DELETE CASCADE,
  catalog_id   UUID REFERENCES catalog_goals(id) ON DELETE SET NULL,
  title        TEXT NOT NULL,
  type         TEXT NOT NULL,
  notes        TEXT,
  completed    BOOLEAN NOT NULL DEFAULT false,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE goal_requirement_progress (
  goal_id        UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  requirement_id UUID NOT NULL REFERENCES catalog_requirements(id) ON DELETE CASCADE,
  completed      BOOLEAN NOT NULL DEFAULT false,
  completed_at   TIMESTAMPTZ,
  PRIMARY KEY (goal_id, requirement_id)
);

CREATE TABLE custom_requirements (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  goal_id      UUID NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  description  TEXT NOT NULL,
  completed    BOOLEAN NOT NULL DEFAULT false,
  completed_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
