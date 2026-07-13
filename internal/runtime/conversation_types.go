package runtime

import (
	"fmt"
	"strings"
)

const (
	defaultRecentConversationObservations = 12
	maxRecentConversationObservations     = 50
)

type ConversationAnchor struct {
	Channel  string `json:"channel"`
	ThreadID string `json:"thread_id"`
}

func (a ConversationAnchor) Normalized() (ConversationAnchor, error) {
	a.Channel = strings.TrimSpace(a.Channel)
	a.ThreadID = strings.TrimSpace(a.ThreadID)
	if a.Channel == "" {
		return ConversationAnchor{}, fmt.Errorf("conversation channel is required")
	}
	if a.ThreadID == "" {
		return ConversationAnchor{}, fmt.Errorf("conversation thread_id is required")
	}
	if len(a.Channel) > 128 {
		return ConversationAnchor{}, fmt.Errorf("conversation channel is too long")
	}
	if len(a.ThreadID) > 512 {
		return ConversationAnchor{}, fmt.Errorf("conversation thread_id is too long")
	}
	return a, nil
}

type ConversationResolution struct {
	Status       ResolutionStatus `json:"status"`
	ContinuityID string           `json:"continuity_id"`
	Channel      string           `json:"channel"`
	ThreadID     string           `json:"thread_id"`
	Created      bool             `json:"created"`
}

type ConversationObservation struct {
	ID       string          `json:"id"`
	Sequence int64           `json:"sequence"`
	Kind     ObservationKind `json:"kind"`
	Content  string          `json:"content"`
}
