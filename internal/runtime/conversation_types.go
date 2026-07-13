package runtime

import (
	"fmt"
	"strings"
)

const (
	defaultRecentConversationObservations = 12
	maxRecentConversationObservations     = 50
)

type ChatTurnStatus string

const (
	ChatTurnInProgress ChatTurnStatus = "in_progress"
	ChatTurnCompleted  ChatTurnStatus = "completed"
	ChatTurnFailed     ChatTurnStatus = "failed"
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

type ChatTurnRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	Message     string             `json:"message"`
}

func (r *ChatTurnRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.Message = strings.TrimSpace(r.Message)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.Message == "" {
		return fmt.Errorf("message is required")
	}
	if len(r.OperationID) > 512 {
		return fmt.Errorf("operation_id is too long")
	}
	if len(r.Message) > 128*1024 {
		return fmt.Errorf("message is too long")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type ChatTurnReceipt struct {
	ID                     string         `json:"turn_id"`
	OperationID            string         `json:"operation_id"`
	Status                 ChatTurnStatus `json:"status"`
	ContinuityID           string         `json:"continuity_id"`
	DeliveryID             string         `json:"delivery_id,omitempty"`
	UserObservationID      string         `json:"user_observation_id"`
	AssistantObservationID string         `json:"assistant_observation_id,omitempty"`
	Answer                 string         `json:"answer,omitempty"`
	Model                  string         `json:"model,omitempty"`
	FailureCode            string         `json:"failure_code,omitempty"`
	Replayed               bool           `json:"replayed"`
}

type ConversationServiceConfig struct {
	MemoryLimit int
	RecentLimit int
}

func (c ConversationServiceConfig) normalized() ConversationServiceConfig {
	if c.MemoryLimit <= 0 {
		c.MemoryLimit = defaultContextItems
	}
	if c.MemoryLimit > maxContextItems {
		c.MemoryLimit = maxContextItems
	}
	if c.RecentLimit <= 0 {
		c.RecentLimit = defaultRecentConversationObservations
	}
	if c.RecentLimit > maxRecentConversationObservations {
		c.RecentLimit = maxRecentConversationObservations
	}
	return c
}
