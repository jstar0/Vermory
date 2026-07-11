-- name: CreateProject :one
INSERT INTO projects (name, description, project_type, objective)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateSource :one
INSERT INTO sources (project_id, source_type, title)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateSourceVersion :one
INSERT INTO source_versions (source_id, version, artifact_uri, content_hash, byte_size)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateClaim :one
INSERT INTO claims (project_id, source_id, source_version_id, claim_type, content, status, verified_by_user)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ConfirmClaim :one
UPDATE claims
SET status = 'confirmed', verified_by_user = true, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListConfirmedClaimsByProject :many
SELECT *
FROM claims
WHERE project_id = $1 AND status IN ('confirmed', 'active') AND verified_by_user = true
ORDER BY created_at, id;

-- name: CreateCapsule :one
INSERT INTO capsules (project_id, title, purpose, summary)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: AddClaimToCapsule :exec
INSERT INTO capsule_claims (project_id, capsule_id, claim_id, order_index)
VALUES ($1, $2, $3, $4);

-- name: CreatePacket :one
INSERT INTO packets (project_id, capsule_id, target_profile_id, title, body, redaction_count, token_estimate)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateAuditLog :one
INSERT INTO audit_logs (project_id, action, target_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateWCEFRun :one
INSERT INTO wcef_runs (project_id, case_id, report_artifact_uri, scores_artifact_uri)
VALUES ($1, $2, $3, $4)
RETURNING *;
