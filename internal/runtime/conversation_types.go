package runtime

import (
	"fmt"
	"strings"
	"time"
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
	FailureMessage         string         `json:"-"`
	Replayed               bool           `json:"replayed"`
	RequestFingerprint     string         `json:"-"`
	AnswerFingerprint      string         `json:"-"`
}

type ExternalConversationTurnRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	Message     string             `json:"message"`
}

func (r *ExternalConversationTurnRequest) Validate() error {
	request := ChatTurnRequest{OperationID: r.OperationID, Anchor: r.Anchor, Message: r.Message}
	if err := request.Validate(); err != nil {
		return err
	}
	r.OperationID = request.OperationID
	r.Anchor = request.Anchor
	r.Message = request.Message
	return nil
}

type PreparedConversationTurn struct {
	ChatTurnReceipt
	Context string `json:"context,omitempty"`
}

type CompleteExternalConversationTurnRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	Answer      string             `json:"answer"`
	Model       string             `json:"model"`
}

func (r *CompleteExternalConversationTurnRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.Answer = strings.TrimSpace(r.Answer)
	r.Model = strings.TrimSpace(r.Model)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.Answer == "" {
		return fmt.Errorf("answer is required")
	}
	if r.Model == "" {
		return fmt.Errorf("model is required")
	}
	if len(r.OperationID) > 512 {
		return fmt.Errorf("operation_id is too long")
	}
	if len(r.Answer) > 128*1024 {
		return fmt.Errorf("answer is too long")
	}
	if len(r.Model) > 512 {
		return fmt.Errorf("model is too long")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type FailExternalConversationTurnRequest struct {
	OperationID    string             `json:"operation_id"`
	Anchor         ConversationAnchor `json:"-"`
	FailureCode    string             `json:"failure_code"`
	FailureMessage string             `json:"failure_message"`
}

func (r *FailExternalConversationTurnRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.FailureCode = strings.TrimSpace(r.FailureCode)
	r.FailureMessage = strings.TrimSpace(r.FailureMessage)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.FailureCode == "" {
		return fmt.Errorf("failure_code is required")
	}
	if len(r.OperationID) > 512 {
		return fmt.Errorf("operation_id is too long")
	}
	if len(r.FailureCode) > 128 {
		return fmt.Errorf("failure_code is too long")
	}
	if len(r.FailureMessage) > 512 {
		return fmt.Errorf("failure_message is too long")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type ConversationServiceConfig struct {
	MemoryLimit int
	RecentLimit int
	Retriever   MemoryRetriever
}

type ReviewConversationCandidateRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	MemoryID    string             `json:"memory_id"`
}

type ConversationReviewCandidate struct {
	CandidateMemoryID   string                  `json:"candidate_memory_id"`
	MemoryKey           string                  `json:"memory_key"`
	Content             string                  `json:"content"`
	SourceQuote         string                  `json:"source_quote"`
	SourceObservationID string                  `json:"source_observation_id"`
	Decision            SourceFormationDecision `json:"decision"`
	TargetMemoryID      string                  `json:"target_memory_id,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
}

type ConversationReviewInbox struct {
	Resolution ConversationResolution        `json:"resolution"`
	Candidates []ConversationReviewCandidate `json:"candidates"`
}

func (r *ReviewConversationCandidateRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.MemoryID = strings.TrimSpace(r.MemoryID)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.MemoryID == "" {
		return fmt.Errorf("memory_id is required")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type ConfirmConversationMemoryRequest struct {
	OperationID   string             `json:"operation_id"`
	Anchor        ConversationAnchor `json:"-"`
	ObservationID string             `json:"observation_id"`
}

func (r *ConfirmConversationMemoryRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.ObservationID = strings.TrimSpace(r.ObservationID)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.ObservationID == "" {
		return fmt.Errorf("observation_id is required")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type CorrectConversationMemoryRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	MemoryID    string             `json:"memory_id"`
	Content     string             `json:"content"`
}

func (r *CorrectConversationMemoryRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.MemoryID = strings.TrimSpace(r.MemoryID)
	r.Content = strings.TrimSpace(r.Content)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.MemoryID == "" {
		return fmt.Errorf("memory_id is required")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type ForgetConversationMemoryRequest struct {
	OperationID string             `json:"operation_id"`
	Anchor      ConversationAnchor `json:"-"`
	MemoryID    string             `json:"memory_id"`
}

func (r *ForgetConversationMemoryRequest) Validate() error {
	r.OperationID = strings.TrimSpace(r.OperationID)
	r.MemoryID = strings.TrimSpace(r.MemoryID)
	if r.OperationID == "" {
		return fmt.Errorf("operation_id is required")
	}
	if r.MemoryID == "" {
		return fmt.Errorf("memory_id is required")
	}
	anchor, err := r.Anchor.Normalized()
	if err != nil {
		return err
	}
	r.Anchor = anchor
	return nil
}

type ConversationInspection struct {
	Resolution   ConversationResolution    `json:"conversation"`
	Observations []ConversationObservation `json:"observations"`
	Memories     []GovernedMemory          `json:"memories"`
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
