package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *Store) ResolveOrCreateConversation(ctx context.Context, tenantID string, anchor ConversationAnchor) (ConversationResolution, error) {
	anchor, err := anchor.Normalized()
	if err != nil {
		return ConversationResolution{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ConversationResolution{}, fmt.Errorf("begin conversation binding: %w", err)
	}
	defer tx.Rollback(ctx)

	lockKey := fmt.Sprintf("%d:%s%d:%s%d:%s", len(tenantID), tenantID, len(anchor.Channel), anchor.Channel, len(anchor.ThreadID), anchor.ThreadID)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return ConversationResolution{}, fmt.Errorf("lock conversation binding: %w", err)
	}

	var continuityID string
	err = tx.QueryRow(ctx, `
SELECT b.continuity_id::text
FROM conversation_bindings b
JOIN continuity_spaces c ON c.id = b.continuity_id
WHERE b.tenant_id = $1 AND b.channel = $2 AND b.thread_id = $3
  AND b.binding_state = 'confirmed'
  AND c.continuity_line = 'conversation' AND c.state = 'active'`,
		tenantID, anchor.Channel, anchor.ThreadID).Scan(&continuityID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return ConversationResolution{}, fmt.Errorf("commit existing conversation binding: %w", err)
		}
		return ConversationResolution{
			Status:       ResolutionResolved,
			ContinuityID: continuityID,
			Channel:      anchor.Channel,
			ThreadID:     anchor.ThreadID,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ConversationResolution{}, fmt.Errorf("lookup conversation binding: %w", err)
	}

	if err := tx.QueryRow(ctx, `
INSERT INTO continuity_spaces (tenant_id, continuity_line, state)
VALUES ($1, 'conversation', 'active')
RETURNING id::text`, tenantID).Scan(&continuityID); err != nil {
		return ConversationResolution{}, fmt.Errorf("create conversation continuity: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO conversation_bindings (continuity_id, tenant_id, channel, thread_id, binding_state)
VALUES ($1::uuid, $2, $3, $4, 'confirmed')`, continuityID, tenantID, anchor.Channel, anchor.ThreadID); err != nil {
		return ConversationResolution{}, fmt.Errorf("create conversation binding: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ConversationResolution{}, fmt.Errorf("commit conversation binding: %w", err)
	}
	return ConversationResolution{
		Status:       ResolutionResolved,
		ContinuityID: continuityID,
		Channel:      anchor.Channel,
		ThreadID:     anchor.ThreadID,
		Created:      true,
	}, nil
}

func (s *Store) ListRecentConversationObservations(ctx context.Context, tenantID, continuityID, beforeObservationID string, limit int) ([]ConversationObservation, error) {
	if limit <= 0 {
		limit = defaultRecentConversationObservations
	}
	if limit > maxRecentConversationObservations {
		limit = maxRecentConversationObservations
	}

	var beforeSequence int64
	if beforeObservationID != "" {
		err := s.pool.QueryRow(ctx, `
SELECT o.observation_seq
FROM observations o
JOIN continuity_spaces c ON c.id = o.continuity_id
WHERE o.id = $1::uuid AND o.tenant_id = $2 AND o.continuity_id = $3::uuid
  AND c.continuity_line = 'conversation' AND c.state = 'active'`,
			beforeObservationID, tenantID, continuityID).Scan(&beforeSequence)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("before observation does not belong to this conversation")
		}
		if err != nil {
			return nil, fmt.Errorf("lookup conversation observation boundary: %w", err)
		}
	}

	rows, err := s.pool.Query(ctx, `
SELECT id::text, observation_seq, observation_kind, content
FROM observations
WHERE tenant_id = $1 AND continuity_id = $2::uuid
  AND observation_kind IN ('user_message', 'assistant_message')
  AND content <> '[redacted]'
  AND ($3::bigint = 0 OR observation_seq < $3)
ORDER BY observation_seq DESC
LIMIT $4`, tenantID, continuityID, beforeSequence, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent conversation observations: %w", err)
	}
	defer rows.Close()

	reversed := make([]ConversationObservation, 0, limit)
	for rows.Next() {
		var observation ConversationObservation
		if err := rows.Scan(&observation.ID, &observation.Sequence, &observation.Kind, &observation.Content); err != nil {
			return nil, fmt.Errorf("scan recent conversation observation: %w", err)
		}
		reversed = append(reversed, observation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent conversation observations: %w", err)
	}

	observations := make([]ConversationObservation, len(reversed))
	for i := range reversed {
		observations[len(reversed)-1-i] = reversed[i]
	}
	return observations, nil
}
