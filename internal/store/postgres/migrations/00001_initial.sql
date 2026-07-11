-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  project_type TEXT NOT NULL,
  objective TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_type TEXT NOT NULL,
  title TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, id)
);

CREATE TABLE source_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
  version INT NOT NULL,
  artifact_uri TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  byte_size BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(source_id, id),
  UNIQUE(source_id, version)
);

CREATE TABLE claims (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_id UUID NOT NULL,
  source_version_id UUID NOT NULL,
  claim_type TEXT NOT NULL,
  content TEXT NOT NULL,
  status TEXT NOT NULL,
  verified_by_user BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, id),
  CONSTRAINT claims_project_source_fk FOREIGN KEY (project_id, source_id)
    REFERENCES sources(project_id, id) ON DELETE CASCADE,
  CONSTRAINT claims_source_version_fk FOREIGN KEY (source_id, source_version_id)
    REFERENCES source_versions(source_id, id) ON DELETE CASCADE
);

CREATE INDEX claims_project_status_type_idx ON claims(project_id, status, claim_type);

CREATE TABLE capsules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  purpose TEXT NOT NULL,
  summary TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, id)
);

CREATE TABLE capsule_claims (
  project_id UUID NOT NULL,
  capsule_id UUID NOT NULL,
  claim_id UUID NOT NULL,
  order_index INT NOT NULL,
  PRIMARY KEY (capsule_id, claim_id),
  CONSTRAINT capsule_claims_project_capsule_fk FOREIGN KEY (project_id, capsule_id)
    REFERENCES capsules(project_id, id) ON DELETE CASCADE,
  CONSTRAINT capsule_claims_project_claim_fk FOREIGN KEY (project_id, claim_id)
    REFERENCES claims(project_id, id) ON DELETE CASCADE
);

CREATE TABLE packets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  capsule_id UUID NOT NULL,
  target_profile_id TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  redaction_count INT NOT NULL DEFAULT 0,
  token_estimate INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CONSTRAINT packets_project_capsule_fk FOREIGN KEY (project_id, capsule_id)
    REFERENCES capsules(project_id, id) ON DELETE CASCADE
);

CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  action TEXT NOT NULL,
  target_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE wcef_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  case_id TEXT NOT NULL,
  report_artifact_uri TEXT NOT NULL DEFAULT '',
  scores_artifact_uri TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS wcef_runs;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS packets;
DROP TABLE IF EXISTS capsule_claims;
DROP TABLE IF EXISTS capsules;
DROP TABLE IF EXISTS claims;
DROP TABLE IF EXISTS source_versions;
DROP TABLE IF EXISTS sources;
DROP TABLE IF EXISTS projects;
